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

	"github.com/zero-os/0-stor/client/datastor/pipeline"
	"github.com/zero-os/0-stor/client/datastor/pipeline/storage"
	"github.com/zero-os/0-stor/client/metastor/metatypes"
	"github.com/zero-os/0-stor/daemon/api/grpc/rpctypes"
	pb "github.com/zero-os/0-stor/daemon/api/grpc/schema"

	log "github.com/Sirupsen/logrus"
	"golang.org/x/net/context"
	"golang.org/x/sync/errgroup"
)

func newDataService(client dataClient, disableLocalFSAccess bool) *dataService {
	return &dataService{
		client:               client,
		disableLocalFSAccess: disableLocalFSAccess,
	}
}

// dataService is used write, read, delete, check and repair (processed) data.
// as data can be written to multiple servers and/or be split over multiple chunks,
// some metadata is returned which the user is expected to store somewhere,
// as that metadata is required to read and manage the data, later on.
type dataService struct {
	client               dataClient
	disableLocalFSAccess bool
}

// Write implements DataServiceServer.Write
func (service *dataService) Write(ctx context.Context, req *pb.DataWriteRequest) (*pb.DataWriteResponse, error) {
	data := req.GetData()
	if len(data) == 0 {
		return nil, rpctypes.ErrGRPCNilData
	}

	// write the given data
	chunks, err := service.client.Write(bytes.NewReader(data))
	if err != nil {
		return nil, mapDataStorError(err)
	}

	protoChunks := convertInMemoryToProtoChunkSlice(chunks)
	return &pb.DataWriteResponse{Chunks: protoChunks}, nil
}

// WriteFile implements DataServiceServer.WriteFile
func (service *dataService) WriteFile(ctx context.Context, req *pb.DataWriteFileRequest) (resp *pb.DataWriteFileResponse, err error) {
	// if the local FS is disabled, we cannot allow this action,
	// a reasonable scenario where this is the case,
	// is in case the daemon is supposed to be used by a remote user,
	// in which case local FS access does not make sense, and might even be dangerous
	if service.disableLocalFSAccess {
		return nil, rpctypes.ErrGRPCNoLocalFS
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

	// write the file data
	chunks, err := service.client.Write(file)
	if err != nil {
		return nil, mapDataStorError(err)
	}

	protoChunks := convertInMemoryToProtoChunkSlice(chunks)
	return &pb.DataWriteFileResponse{Chunks: protoChunks}, nil
}

// WriteStream implements DataServiceServer.WriteStream
func (service *dataService) WriteStream(stream pb.DataService_WriteStreamServer) error {
	reader, writer := io.Pipe()
	ctx := stream.Context()
	group, ctx := errgroup.WithContext(ctx)

	// start the writer
	var chunks []metatypes.Chunk
	group.Go(func() error {
		var err error
		chunks, err = service.client.Write(reader)
		return mapDataStorError(err)
	})

	// start the reader
	group.Go(func() (err error) {
		defer func() {
			e := writer.Close()
			if e != nil {
				if err == nil {
					err = e
				}
				log.Errorf("error while closing (*dataService).WriteStream's PipeWriter: %v", e)
			}
		}()

		var (
			data []byte
			msg  *pb.DataWriteStreamRequest
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

			data = msg.GetDataChunk()
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
	err := group.Wait()
	if err != nil {
		return err
	}

	// convert our chunks to the proto version,
	// and return it as a final response
	protoChunks := convertInMemoryToProtoChunkSlice(chunks)
	return stream.SendAndClose(&pb.DataWriteStreamResponse{
		Chunks: protoChunks,
	})
}

// Read implements DataServiceServer.Read
func (service *dataService) Read(ctx context.Context, req *pb.DataReadRequest) (*pb.DataReadResponse, error) {
	chunks := req.GetChunks()
	if len(chunks) == 0 {
		return nil, rpctypes.ErrGRPCNilChunks
	}

	buf := bytes.NewBuffer(nil)
	imChunks := convertProtoToInMemoryChunkSlice(chunks)
	err := service.client.Read(imChunks, buf)
	if err != nil {
		return nil, mapDataStorError(err)
	}
	data := buf.Bytes()
	if len(data) == 0 {
		return nil, rpctypes.ErrGRPCDataNotRead
	}
	return &pb.DataReadResponse{Data: data}, nil
}

// ReadFile implements DataServiceServer.ReadFile
func (service *dataService) ReadFile(ctx context.Context, req *pb.DataReadFileRequest) (*pb.DataReadFileResponse, error) {
	// if the local FS is disabled, we cannot allow this action,
	// a reasonable scenario where this is the case,
	// is in case the daemon is supposed to be used by a remote user,
	// in which case local FS access does not make sense, and might even be dangerous
	if service.disableLocalFSAccess {
		return nil, rpctypes.ErrGRPCNoLocalFS
	}

	chunks := req.GetChunks()
	if len(chunks) == 0 {
		return nil, rpctypes.ErrGRPCNilChunks
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

	imChunks := convertProtoToInMemoryChunkSlice(chunks)
	err = service.client.Read(imChunks, file)
	if err != nil {
		return nil, mapDataStorError(err)
	}
	return &pb.DataReadFileResponse{}, nil
}

// ReadStream implements DataServiceServer.ReadStream
func (service *dataService) ReadStream(req *pb.DataReadStreamRequest, stream pb.DataService_ReadStreamServer) error {
	chunkSize := req.GetChunkSize()
	if chunkSize <= 0 {
		return rpctypes.ErrGRPCInvalidChunkSize
	}

	chunks := req.GetChunks()
	if len(chunks) == 0 {
		return rpctypes.ErrGRPCNilChunks
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
				log.Errorf("error while closing (*dataService).ReadStream's PipeWriter: %v", e)
			}
		}()
		imChunks := convertProtoToInMemoryChunkSlice(chunks)
		err = service.client.Read(imChunks, writer)
		if err != nil {
			return mapDataStorError(err)
		}
		return nil
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
			err = stream.Send(&pb.DataReadStreamResponse{DataChunk: buf[:n]})
			if err != nil {
				return err
			}
		}
	})

	// read in chunks until we reached the end
	return group.Wait()
}

// Delete implements DataServiceServer.Delete
func (service *dataService) Delete(ctx context.Context, req *pb.DataDeleteRequest) (*pb.DataDeleteResponse, error) {
	chunks := req.GetChunks()
	if len(chunks) == 0 {
		return nil, rpctypes.ErrGRPCNilChunks
	}

	imChunks := convertProtoToInMemoryChunkSlice(chunks)
	err := service.client.Delete(imChunks)
	if err != nil {
		return nil, mapDataStorError(err)
	}
	return &pb.DataDeleteResponse{}, nil
}

// Check implements DataServiceServer.Check
func (service *dataService) Check(ctx context.Context, req *pb.DataCheckRequest) (*pb.DataCheckResponse, error) {
	chunks := req.GetChunks()
	if len(chunks) == 0 {
		return nil, rpctypes.ErrGRPCNilChunks
	}

	imChunks := convertProtoToInMemoryChunkSlice(chunks)
	status, err := service.client.Check(imChunks, req.GetFast())
	if err != nil {
		return nil, mapDataStorError(err)
	}
	protoStatus := convertStorageToProtoCheckStatus(status)
	return &pb.DataCheckResponse{Status: protoStatus}, nil
}

// Repair implements DataServiceServer.Repair
func (service *dataService) Repair(ctx context.Context, req *pb.DataRepairRequest) (*pb.DataRepairResponse, error) {
	chunks := req.GetChunks()
	if len(chunks) == 0 {
		return nil, rpctypes.ErrGRPCNilChunks
	}

	imChunks := convertProtoToInMemoryChunkSlice(chunks)
	imChunks, err := service.client.Repair(imChunks)
	if err != nil {
		return nil, mapDataStorError(err)
	}
	protoChunks := convertInMemoryToProtoChunkSlice(imChunks)
	return &pb.DataRepairResponse{Chunks: protoChunks}, nil
}

type dataClient interface {
	Write(r io.Reader) ([]metatypes.Chunk, error)
	Read(chunks []metatypes.Chunk, w io.Writer) error
	Delete(chunks []metatypes.Chunk) error
	Check(chunks []metatypes.Chunk, fast bool) (storage.CheckStatus, error)
	Repair(chunks []metatypes.Chunk) ([]metatypes.Chunk, error)
}

var (
	_ pb.DataServiceServer = (*dataService)(nil)
	_ dataClient           = (pipeline.Pipeline)(nil)
)
