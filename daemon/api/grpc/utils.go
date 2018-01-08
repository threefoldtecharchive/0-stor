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
	"fmt"
	"io"
	"os"

	"github.com/zero-os/0-stor/client/datastor"
	"github.com/zero-os/0-stor/client/metastor/metatypes"

	"github.com/zero-os/0-stor/client/datastor/pipeline/storage"
	"github.com/zero-os/0-stor/client/metastor"
	"github.com/zero-os/0-stor/daemon/api/grpc/rpctypes"
	pb "github.com/zero-os/0-stor/daemon/api/grpc/schema"
)

func convertProtoToInMemoryChunkSlice(chunks []pb.Chunk) []metatypes.Chunk {
	n := len(chunks)
	if n == 0 {
		return nil
	}
	imChunks := make([]metatypes.Chunk, n)
	for i, c := range chunks {
		chunk := &imChunks[i]
		chunk.Size = c.GetSizeInBytes()
		chunk.Hash = c.GetHash()
		objects := c.GetObjects()
		n = len(objects)
		if n == 0 {
			continue
		}
		chunk.Objects = make([]metatypes.Object, n)
		for i, o := range objects {
			object := &chunk.Objects[i]
			object.Key = o.GetKey()
			object.ShardID = o.GetShardID()
		}
	}
	return imChunks
}

func convertProtoToInMemoryMetadata(metadata *pb.Metadata) metatypes.Metadata {
	return metatypes.Metadata{
		Key:            metadata.GetKey(),
		Size:           metadata.GetSizeInBytes(),
		CreationEpoch:  metadata.GetCreationEpoch(),
		LastWriteEpoch: metadata.GetLastWriteEpoch(),
		Chunks:         convertProtoToInMemoryChunkSlice(metadata.GetChunks()),
	}
}

func convertInMemoryToProtoChunkSlice(chunks []metatypes.Chunk) []pb.Chunk {
	n := len(chunks)
	if n == 0 {
		return nil
	}

	protoChunks := make([]pb.Chunk, n)
	for i, c := range chunks {
		chunk := &protoChunks[i]
		chunk.SizeInBytes = c.Size
		chunk.Hash = c.Hash
		n = len(c.Objects)
		if n == 0 {
			continue
		}
		chunk.Objects = make([]pb.Object, n)
		for i, o := range c.Objects {
			object := &chunk.Objects[i]
			object.Key = o.Key
			object.ShardID = o.ShardID
		}
	}
	return protoChunks
}

func convertInMemoryToProtoMetadata(metadata metatypes.Metadata) *pb.Metadata {
	return &pb.Metadata{
		Key:            metadata.Key,
		SizeInBytes:    metadata.Size,
		CreationEpoch:  metadata.CreationEpoch,
		LastWriteEpoch: metadata.LastWriteEpoch,
		Chunks:         convertInMemoryToProtoChunkSlice(metadata.Chunks),
	}
}

var openFileToRead = func(path string) (io.ReadCloser, error) {
	return os.Open(path)
}

func openFileToWrite(filePath string, fileMode pb.FileMode, sync bool) (io.WriteCloser, error) {
	if len(filePath) == 0 {
		return nil, rpctypes.ErrGRPCNilFilePath
	}

	flags, err := convertFileModeToSyscallFlags(fileMode)
	if err != nil {
		return nil, err
	}
	if sync {
		flags |= os.O_SYNC
	}
	return openFile(filePath, flags, 0644)
}

var openFile = func(path string, flag int, perm os.FileMode) (io.ReadWriteCloser, error) {
	return os.OpenFile(path, flag, perm)
}

func convertFileModeToSyscallFlags(fileMode pb.FileMode) (int, error) {
	flags := os.O_RDWR | os.O_CREATE
	switch fileMode {
	case pb.FileModeTruncate:
		flags |= os.O_TRUNC
	case pb.FileModeAppend:
		flags |= os.O_APPEND
	case pb.FileModeExclusive:
		flags |= os.O_EXCL
	default:
		return 0, rpctypes.ErrGRPCInvalidFileMode
	}
	return flags, nil
}

func convertStorageToProtoCheckStatus(status storage.CheckStatus) pb.CheckStatus {
	checkStatus, ok := _StorageToProtoCheckStatusMapping[status]
	if !ok {
		panic(fmt.Sprintf("unsupported check status: %v", status))
	}
	return checkStatus
}

var _StorageToProtoCheckStatusMapping = map[storage.CheckStatus]pb.CheckStatus{
	storage.CheckStatusInvalid: pb.CheckStatusInvalid,
	storage.CheckStatusOptimal: pb.CheckStatusOptimal,
	storage.CheckStatusValid:   pb.CheckStatusValid,
}

func mapZstorError(err error) error {
	if cerr, ok := _ErrMetaStorErrorMapping[err]; ok {
		return cerr
	}
	if cerr, ok := _ErrDataStorErrorMapping[err]; ok {
		return cerr
	}
	return err
}

func mapDataStorError(err error) error {
	if cerr, ok := _ErrDataStorErrorMapping[err]; ok {
		return cerr
	}
	return err
}

var _ErrDataStorErrorMapping = map[error]error{
	datastor.ErrKeyNotFound:      rpctypes.ErrGRPCKeyNotFound,
	datastor.ErrObjectCorrupted:  rpctypes.ErrGRPCDataCorrupted,
	datastor.ErrPermissionDenied: rpctypes.ErrGRPCPermissionDenied,
}

func mapMetaStorError(err error) error {
	if cerr, ok := _ErrMetaStorErrorMapping[err]; ok {
		return cerr
	}
	return err
}

var _ErrMetaStorErrorMapping = map[error]error{
	metastor.ErrNotFound: rpctypes.ErrGRPCKeyNotFound,
}
