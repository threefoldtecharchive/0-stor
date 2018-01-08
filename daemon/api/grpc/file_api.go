/*
 * Copyright (C) 2017-2018 GIG Technology NV and Contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package grpc

import (
	"bytes"
	"io"

	"github.com/zero-os/0-stor/client"
	"github.com/zero-os/0-stor/client/datastor/pipeline/storage"
	"github.com/zero-os/0-stor/client/metastor/metatypes"
	"github.com/zero-os/0-stor/daemon/api/grpc/rpctypes"
	pb "github.com/zero-os/0-stor/daemon/api/grpc/schema"

	log "github.com/Sirupsen/logrus"
	"golang.org/x/net/context"
	"golang.org/x/sync/errgroup"
)

func newFileService(client fileClient, disableLocalFSAccess bool) *fileService {
	return &fileService{
		client:               client,
		disableLocalFSAccess: disableLocalFSAccess,
	}
}

// fileService is used write, read, delete, check and repair files.
// The fileService follows the principle of everything is a file.
// All files are written as raw binary data, but also have
// metadata bound to them, which identify where and how they are stored.
type fileService struct {
	client               fileClient
	disableLocalFSAccess bool
}

// Write implements FileServiceServer.Write
func (service *fileService) Write(ctx context.Context, req *pb.WriteRequest) (*pb.WriteResponse, error) {
	key := req.GetKey()
	if len(key) == 0 {
		return nil, rpctypes.ErrGRPCNilKey
	}
	data := req.GetData()
	if len(data) == 0 {
		return nil, rpctypes.ErrGRPCNilData
	}

	metadata, err := service.client.Write(key, bytes.NewReader(data))
	if err != nil {
		return nil, mapZstorError(err)
	}

	output := convertInMemoryToProtoMetadata(*metadata)
	return &pb.WriteResponse{
		Metadata: output,
	}, nil
}

// WriteFile implements FileServiceServer.WriteFile
func (service *fileService) WriteFile(ctx context.Context, req *pb.WriteFileRequest) (resp *pb.WriteFileResponse, err error) {
	// if the local FS is disabled, we cannot allow this action,
	// a reasonable scenario where this is the case,
	// is in case the daemon is supposed to be used by a remote user,
	// in which case local FS access does not make sense, and might even be dangerous
	if service.disableLocalFSAccess {
		return nil, rpctypes.ErrGRPCNoLocalFS
	}

	key := req.GetKey()
	if len(key) == 0 {
		return nil, rpctypes.ErrGRPCNilKey
	}
	filePath := req.GetFilePath()
	if len(filePath) == 0 {
		return nil, rpctypes.ErrGRPCNilFilePath
	}

	// open the file
	file, err := openFileToRead(filePath)
	if err != nil {
		return nil, err
	}
	defer func() {
		fileErr := file.Close()
		if fileErr != nil {
			if err == nil {
				err = fileErr
			}
			log.Errorf("error while closing file '%s': %v", filePath, fileErr)
		}
	}()

	// write directly from the file
	metadata, err := service.client.Write(key, file)
	if err != nil {
		return nil, mapZstorError(err)
	}

	output := convertInMemoryToProtoMetadata(*metadata)
	return &pb.WriteFileResponse{
		Metadata: output,
	}, nil
}

// WriteStream implements FileServiceServer.WriteStream
func (service *fileService) WriteStream(stream pb.FileService_WriteStreamServer) error {
	msg, err := stream.Recv()
	if err != nil {
		return err
	}
	key := msg.GetMetadata().GetKey()
	if len(key) == 0 {
		return rpctypes.ErrGRPCNilKey
	}

	reader, writer := io.Pipe()
	ctx := stream.Context()
	group, ctx := errgroup.WithContext(ctx)

	// start the writer
	var metadata *metatypes.Metadata
	group.Go(func() error {
		var err error
		metadata, err = service.client.Write(key, reader)
		return mapZstorError(err)
	})

	// start the reader
	group.Go(func() (err error) {
		defer func() {
			e := writer.Close()
			if e != nil {
				if err == nil {
					err = e
				}
				log.Errorf("error while closing (*fileService).WriteStream's PipeWriter: %v", e)
			}
		}()

		var (
			data []byte
			msg  *pb.WriteStreamRequest
		)
		for {
			// as long as we receive data,
			// we keep writing data
			msg, err = stream.Recv()
			if err != nil {
				if err == io.EOF {
					return nil
				}
				return err
			}

			data = msg.GetData().GetDataChunk()
			if len(data) == 0 {
				return rpctypes.ErrGRPCNilData
			}

			_, err = writer.Write(data)
			if err != nil {
				return err
			}
		}
	})

	// wait until all data has been received and written,
	// or until an error has interupted the process
	err = group.Wait()
	if err != nil {
		return err
	}

	// convert our metadata to the proto version,
	// and return it as a final response
	output := convertInMemoryToProtoMetadata(*metadata)
	return stream.SendAndClose(&pb.WriteStreamResponse{
		Metadata: output,
	})
}

// Read implements FileServiceServer.Read
func (service *fileService) Read(ctx context.Context, req *pb.ReadRequest) (*pb.ReadResponse, error) {
	var (
		err error
		buf = bytes.NewBuffer(nil)
	)

	switch v := req.GetInput().(type) {
	case *pb.ReadRequest_Key:
		if len(v.Key) == 0 {
			return nil, rpctypes.ErrGRPCNilKey
		}
		err = service.client.Read(v.Key, buf)
	case *pb.ReadRequest_Metadata:
		if v.Metadata == nil {
			return nil, rpctypes.ErrGRPCNilMetadata
		}
		metadata := convertProtoToInMemoryMetadata(v.Metadata)
		err = service.client.ReadWithMeta(metadata, buf)
	default:
		// if no key or metadata is given,
		// we'll simply assume that a key is forgotten,
		// as that is the more likely one of the 2
		return nil, rpctypes.ErrGRPCNilKey
	}
	if err != nil {
		return nil, mapZstorError(err)
	}

	data := buf.Bytes()
	if len(data) == 0 {
		return nil, rpctypes.ErrGRPCDataNotRead
	}
	return &pb.ReadResponse{Data: data}, nil
}

// ReadFile implements FileServiceServer.ReadFile
func (service *fileService) ReadFile(ctx context.Context, req *pb.ReadFileRequest) (*pb.ReadFileResponse, error) {
	// if the local FS is disabled, we cannot allow this action,
	// a reasonable scenario where this is the case,
	// is in case the daemon is supposed to be used by a remote user,
	// in which case local FS access does not make sense, and might even be dangerous
	if service.disableLocalFSAccess {
		return nil, rpctypes.ErrGRPCNoLocalFS
	}

	filePath := req.GetFilePath()
	file, err := openFileToWrite(
		filePath, req.GetFileMode(), req.GetSynchronousIO())
	if err != nil {
		return nil, err
	}
	defer func() {
		fileErr := file.Close()
		if fileErr != nil {
			if err == nil {
				err = fileErr
			}
			log.Errorf("error while closing file '%s': %v", filePath, fileErr)
		}
	}()

	switch v := req.GetInput().(type) {
	case *pb.ReadFileRequest_Key:
		if len(v.Key) == 0 {
			return nil, rpctypes.ErrGRPCNilKey
		}
		err = service.client.Read(v.Key, file)
	case *pb.ReadFileRequest_Metadata:
		if v.Metadata == nil {
			return nil, rpctypes.ErrGRPCNilMetadata
		}
		metadata := convertProtoToInMemoryMetadata(v.Metadata)
		err = service.client.ReadWithMeta(metadata, file)
	default:
		// if no key or metadata is given,
		// we'll simply assume that a key is forgotten,
		// as that is the more likely one of the 2
		return nil, rpctypes.ErrGRPCNilKey
	}
	if err != nil {
		return nil, mapZstorError(err)
	}
	return &pb.ReadFileResponse{}, nil
}

// ReadStream implements FileServiceServer.ReadStream
func (service *fileService) ReadStream(req *pb.ReadStreamRequest, stream pb.FileService_ReadStreamServer) error {
	chunkSize := req.GetChunkSize()
	if chunkSize <= 0 {
		return rpctypes.ErrGRPCInvalidChunkSize
	}

	reader, writer := io.Pipe()
	ctx := stream.Context()
	group, ctx := errgroup.WithContext(ctx)

	// start writer goroutine
	group.Go(func() (err error) {
		defer func() {
			e := writer.Close()
			if e != nil {
				if err == nil {
					err = e
				}
				log.Errorf("error while closing (*fileService).ReadStream's PipeWriter: %v", e)
			}
		}()

		switch v := req.GetInput().(type) {
		case *pb.ReadStreamRequest_Key:
			if len(v.Key) == 0 {
				return rpctypes.ErrGRPCNilKey
			}
			return service.client.Read(v.Key, writer)

		case *pb.ReadStreamRequest_Metadata:
			if v.Metadata == nil {
				return rpctypes.ErrGRPCNilMetadata
			}
			metadata := convertProtoToInMemoryMetadata(v.Metadata)
			return service.client.ReadWithMeta(metadata, writer)

		default:
			// if no key or metadata is given,
			// we'll simply assume that a key is forgotten,
			// as that is the more likely one of the 2
			return rpctypes.ErrGRPCNilKey
		}
	})

	// start reader goroutine
	group.Go(func() error {
		var (
			n   int
			err error
			buf = make([]byte, chunkSize)
		)
		for {
			n, err = reader.Read(buf)
			if err != nil {
				if err == io.EOF {
					return nil
				}
				return err
			}
			err = stream.Send(&pb.ReadStreamResponse{DataChunk: buf[:n]})
			if err != nil {
				return err
			}
		}
	})

	// read in chunks until we reached the end
	err := group.Wait()
	if err != nil {
		return mapZstorError(err)
	}
	return nil
}

// Delete implements FileServiceServer.Delete
func (service *fileService) Delete(ctx context.Context, req *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	var err error
	switch v := req.GetInput().(type) {
	case *pb.DeleteRequest_Key:
		if len(v.Key) == 0 {
			return nil, rpctypes.ErrGRPCNilKey
		}
		err = service.client.Delete(v.Key)
	case *pb.DeleteRequest_Metadata:
		if v.Metadata == nil {
			return nil, rpctypes.ErrGRPCNilMetadata
		}
		metadata := convertProtoToInMemoryMetadata(v.Metadata)
		err = service.client.DeleteWithMeta(metadata)
	default:
		// if no key or metadata is given,
		// we'll simply assume that a key is forgotten,
		// as that is the more likely one of the 2
		return nil, rpctypes.ErrGRPCNilKey
	}
	if err != nil {
		return nil, mapZstorError(err)
	}
	return &pb.DeleteResponse{}, nil
}

// Check implements FileServiceServer.Check
func (service *fileService) Check(ctx context.Context, req *pb.CheckRequest) (*pb.CheckResponse, error) {
	var (
		err    error
		status storage.CheckStatus
		fast   = req.GetFast()
	)

	switch v := req.GetInput().(type) {
	case *pb.CheckRequest_Key:
		if len(v.Key) == 0 {
			return nil, rpctypes.ErrGRPCNilKey
		}
		status, err = service.client.Check(v.Key, fast)
	case *pb.CheckRequest_Metadata:
		if v.Metadata == nil {
			return nil, rpctypes.ErrGRPCNilMetadata
		}
		metadata := convertProtoToInMemoryMetadata(v.Metadata)
		status, err = service.client.CheckWithMeta(metadata, fast)
	default:
		// if no key or metadata is given,
		// we'll simply assume that a key is forgotten,
		// as that is the more likely one of the 2
		return nil, rpctypes.ErrGRPCNilKey
	}
	if err != nil {
		return nil, mapZstorError(err)
	}
	return &pb.CheckResponse{Status: convertStorageToProtoCheckStatus(status)}, nil
}

// Repair implements FileServiceServer.Repair
func (service *fileService) Repair(ctx context.Context, req *pb.RepairRequest) (*pb.RepairResponse, error) {
	key := req.GetKey()
	if len(key) == 0 {
		return nil, rpctypes.ErrGRPCNilKey
	}

	metadata, err := service.client.Repair(key)
	if err != nil {
		return nil, mapZstorError(err)
	}

	output := convertInMemoryToProtoMetadata(*metadata)
	return &pb.RepairResponse{Metadata: output}, nil
}

type fileClient interface {
	Write(key []byte, r io.Reader) (*metatypes.Metadata, error)
	Read(key []byte, w io.Writer) error
	ReadWithMeta(meta metatypes.Metadata, w io.Writer) error
	Delete(key []byte) error
	DeleteWithMeta(meta metatypes.Metadata) error
	Check(key []byte, fast bool) (storage.CheckStatus, error)
	CheckWithMeta(meta metatypes.Metadata, fast bool) (storage.CheckStatus, error)
	Repair(key []byte) (*metatypes.Metadata, error)
}

var (
	_ pb.FileServiceServer = (*fileService)(nil)
	_ fileClient           = (*client.Client)(nil)
)
