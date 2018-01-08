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
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zero-os/0-stor/client/datastor/pipeline/storage"
	"github.com/zero-os/0-stor/client/metastor/metatypes"
	"github.com/zero-os/0-stor/daemon/api/grpc/rpctypes"
	pb "github.com/zero-os/0-stor/daemon/api/grpc/schema"
)

func TestConvertMetaChunkSlice(t *testing.T) {
	testCases := [][]metatypes.Chunk{
		nil,
		{metatypes.Chunk{Size: 0, Objects: nil, Hash: nil}},
		{metatypes.Chunk{Size: 42, Objects: nil, Hash: []byte("foo")}},
		{metatypes.Chunk{Size: 42, Objects: []metatypes.Object{metatypes.Object{}}, Hash: []byte("foo")}},
		{metatypes.Chunk{Size: 42,
			Objects: []metatypes.Object{metatypes.Object{Key: []byte("bar")}}, Hash: []byte("foo")}},
		{
			metatypes.Chunk{Size: 42,
				Objects: []metatypes.Object{metatypes.Object{Key: []byte("bar")}}, Hash: []byte("foo")},
			metatypes.Chunk{Size: 13,
				Objects: []metatypes.Object{metatypes.Object{Key: []byte("foo")}}, Hash: []byte("faz")},
		},
	}
	for _, testCase := range testCases {
		protoChunks := convertInMemoryToProtoChunkSlice(testCase)
		imChunks := convertProtoToInMemoryChunkSlice(protoChunks)
		assert.Equal(t, testCase, imChunks)
	}
}

func TestConvertMetadata(t *testing.T) {
	testCases := []metatypes.Metadata{
		{},
		{Key: []byte("foo"), Size: 1, CreationEpoch: 2, LastWriteEpoch: 3},
		{Chunks: []metatypes.Chunk{metatypes.Chunk{Size: 42, Objects: nil, Hash: []byte("foo")}}},
		{Key: []byte("foo"), Size: 1, CreationEpoch: 2, LastWriteEpoch: 3,
			Chunks: []metatypes.Chunk{metatypes.Chunk{Size: 42, Objects: nil, Hash: []byte("foo")}}},
		{Key: []byte("foo"), Size: 3, CreationEpoch: 2, LastWriteEpoch: 1,
			Chunks: []metatypes.Chunk{
				metatypes.Chunk{Size: 123, Objects: nil, Hash: []byte("foo")},
				metatypes.Chunk{Size: 321, Objects: []metatypes.Object{
					metatypes.Object{Key: []byte("foo")},
				}, Hash: []byte("bar")},
			}},
	}
	for _, testCase := range testCases {
		protoMetadata := convertInMemoryToProtoMetadata(testCase)
		imMetadata := convertProtoToInMemoryMetadata(protoMetadata)
		assert.Equal(t, testCase, imMetadata)
	}
}

func TestOpenFileToWrite(t *testing.T) {
	_, err := openFileToWrite("foo", pb.FileModeTruncate, true)
	require.NoError(t, err)
	_, err = openFileToWrite("bar", pb.FileModeTruncate, false)
	require.NoError(t, err)
}

func TestOpenFileToWriteError(t *testing.T) {
	_, err := openFileToWrite("", pb.FileModeTruncate, true)
	require.Equal(t, rpctypes.ErrGRPCNilFilePath, err)

	_, err = openFileToWrite("foo", pb.FileMode(math.MaxInt32), true)
	require.Equal(t, rpctypes.ErrGRPCInvalidFileMode, err)
}

func TestOpenFileToWriteFilePathError(t *testing.T) {
	_, err := openFileToWrite("", pb.FileMode(0), false)
	require.Equal(t, rpctypes.ErrGRPCNilFilePath, err)
}

func TestConvertFileModeToSyscallFlags(t *testing.T) {
	testCases := []struct {
		FileMode             pb.FileMode
		ExpectedSyscallFlags int
	}{
		{pb.FileModeAppend, os.O_RDWR | os.O_CREATE | os.O_APPEND},
		{pb.FileModeExclusive, os.O_RDWR | os.O_CREATE | os.O_EXCL},
		{pb.FileModeTruncate, os.O_RDWR | os.O_CREATE | os.O_TRUNC},
		{pb.FileMode(math.MaxInt32), 0},
	}
	for _, testCase := range testCases {
		flags, err := convertFileModeToSyscallFlags(testCase.FileMode)
		if testCase.ExpectedSyscallFlags == 0 {
			assert.Error(t, err)
			continue
		}
		assert.Equal(t, testCase.ExpectedSyscallFlags, flags)
	}
}

func TestConvertStorageToProtoCheckStatus(t *testing.T) {
	testCases := []struct {
		Input    storage.CheckStatus
		Expected pb.CheckStatus
	}{
		{storage.CheckStatusInvalid, pb.CheckStatusInvalid},
		{storage.CheckStatusValid, pb.CheckStatusValid},
		{storage.CheckStatusOptimal, pb.CheckStatusOptimal},
		{storage.CheckStatus(math.MaxUint8), -1},
	}
	for _, testCase := range testCases {
		if testCase.Expected == -1 {
			assert.Panicsf(t, func() {
				convertStorageToProtoCheckStatus(testCase.Input)
			}, "testCase: %v", testCase)
			continue
		}
		status := convertStorageToProtoCheckStatus(testCase.Input)
		assert.Equal(t, testCase.Expected, status)
	}
}

type nopIOCloser struct {
	io.ReadWriter
}

func (rw *nopIOCloser) Close() error { return nil }

func init() {
	openFileToRead = func(path string) (io.ReadCloser, error) {
		return _InMemoryFilesCache.OpenReadCloser(path), nil
	}
	openFile = func(path string, flag int, perm os.FileMode) (io.ReadWriteCloser, error) {
		return _InMemoryFilesCache.OpenReadWriteCloser(path), nil
	}
}

// newInMemoryFile creates a temporary in-memory file
func newInMemoryFile() *inMemoryFile {
	return _InMemoryFilesCache.CreateFile()
}

type inMemoryFiles struct {
	mux   sync.Mutex
	count int
	m     map[string]*bytes.Buffer
}

var _InMemoryFilesCache = &inMemoryFiles{m: make(map[string]*bytes.Buffer)}

func (imf *inMemoryFiles) CreateFile() *inMemoryFile {
	imf.mux.Lock()
	id := fmt.Sprintf("imf#%d", imf.count)
	imf.count++
	imf.m[id] = bytes.NewBuffer(nil)
	imf.mux.Unlock()
	return &inMemoryFile{id: id, owner: true}
}

func (imf *inMemoryFiles) DeleteFile(path string) {
	imf.mux.Lock()
	delete(imf.m, path)
	imf.mux.Unlock()
}

func (imf *inMemoryFiles) OpenReadCloser(path string) io.ReadCloser {
	imf.mux.Lock()
	_, ok := imf.m[path]
	imf.mux.Unlock()

	if !ok {
		return ioutil.NopCloser(strings.NewReader(path))
	}
	return &inMemoryFile{id: path, owner: false}
}

func (imf *inMemoryFiles) OpenReadWriteCloser(path string) io.ReadWriteCloser {
	imf.mux.Lock()
	_, ok := imf.m[path]
	imf.mux.Unlock()

	if !ok {
		return &nopIOCloser{bytes.NewBufferString(path)}
	}
	return &inMemoryFile{id: path, owner: false}
}

func (imf *inMemoryFiles) Write(path string, data []byte) (int, error) {
	imf.mux.Lock()
	buf, ok := imf.m[path]
	if !ok {
		imf.mux.Unlock()
		return 0, fmt.Errorf("%s does not exist", path)
	}
	n, err := buf.Write(data)
	imf.mux.Unlock()
	return n, err
}

func (imf *inMemoryFiles) Read(path string, data []byte) (int, error) {
	imf.mux.Lock()
	buf, ok := imf.m[path]
	if !ok {
		imf.mux.Unlock()
		return 0, fmt.Errorf("%s does not exist", path)
	}
	n, err := buf.Read(data)
	imf.mux.Unlock()
	return n, err
}

type inMemoryFile struct {
	id    string
	owner bool
}

func (imf *inMemoryFile) Read(data []byte) (int, error) {
	return _InMemoryFilesCache.Read(imf.id, data)
}

func (imf *inMemoryFile) Write(data []byte) (int, error) {
	return _InMemoryFilesCache.Write(imf.id, data)
}

func (imf *inMemoryFile) Name() string {
	return imf.id
}

func (imf *inMemoryFile) Close() error {
	if imf.owner {
		_InMemoryFilesCache.DeleteFile(imf.id)
	}
	return nil
}
