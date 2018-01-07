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
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zero-os/0-stor/client/datastor/pipeline/storage"
	"github.com/zero-os/0-stor/client/metastor/metatypes"
	"github.com/zero-os/0-stor/daemon/api/grpc/rpctypes"
	pb "github.com/zero-os/0-stor/daemon/api/grpc/schema"

	"golang.org/x/net/context"
)

func TestFileService_Write(t *testing.T) {
	fSrv := newFileService(&fileClientStub{}, false)

	_, err := fSrv.Write(context.Background(),
		&pb.WriteRequest{Key: []byte("key"), Data: []byte("data")})
	require.NoError(t, err)
}

func TestFileService_WriteError(t *testing.T) {
	fSrv := newFileService(&fileClientStub{}, false)

	_, err := fSrv.Write(context.Background(),
		&pb.WriteRequest{Key: nil, Data: []byte("data")})
	require.Equal(t, rpctypes.ErrGRPCNilKey, err)
	_, err = fSrv.Write(context.Background(),
		&pb.WriteRequest{Key: []byte("key:"), Data: nil})
	require.Equal(t, rpctypes.ErrGRPCNilData, err)
	_, err = fSrv.Write(context.Background(), &pb.WriteRequest{})
	require.Error(t, err)

	// client errors should propagate, iff those code paths hit
	fSrv = newFileService(fileErrorClient{}, false)
	_, err = fSrv.Write(context.Background(),
		&pb.WriteRequest{Key: []byte("key"), Data: []byte("data")})
	require.Equal(t, errFooFileClient, err)
}

func TestFileService_WriteFile(t *testing.T) {
	fSrv := newFileService(&fileClientStub{}, false)

	_, err := fSrv.WriteFile(context.Background(),
		&pb.WriteFileRequest{Key: []byte("key"), FilePath: "foo"})
	require.NoError(t, err)
}

func TestFileService_WriteFileError(t *testing.T) {
	fSrv := newFileService(&fileClientStub{}, false)

	_, err := fSrv.WriteFile(context.Background(),
		&pb.WriteFileRequest{Key: nil, FilePath: "foo"})
	require.Equal(t, rpctypes.ErrGRPCNilKey, err)
	_, err = fSrv.WriteFile(context.Background(),
		&pb.WriteFileRequest{Key: []byte("key"), FilePath: ""})
	require.Equal(t, rpctypes.ErrGRPCNilFilePath, err)
	_, err = fSrv.WriteFile(context.Background(),
		&pb.WriteFileRequest{Key: nil, FilePath: ""})
	require.Error(t, err)

	fSrv.disableLocalFSAccess = true
	_, err = fSrv.WriteFile(context.Background(),
		&pb.WriteFileRequest{Key: []byte("key"), FilePath: "foo"})
	require.Equal(t, rpctypes.ErrGRPCNoLocalFS, err)

	// client errors should propagate, iff those code paths hit
	fSrv = newFileService(fileErrorClient{}, false)
	_, err = fSrv.WriteFile(context.Background(),
		&pb.WriteFileRequest{Key: []byte("key"), FilePath: "foo"})
	require.Equal(t, errFooFileClient, err)
}

func TestFileService_Read(t *testing.T) {
	fSrv := newFileService(&fileClientStub{}, false)

	_, err := fSrv.Read(context.Background(),
		&pb.ReadRequest{Input: &pb.ReadRequest_Key{Key: []byte("key")}})
	require.NoError(t, err)
	_, err = fSrv.Read(context.Background(),
		&pb.ReadRequest{Input: &pb.ReadRequest_Metadata{Metadata: new(pb.Metadata)}})
	require.NoError(t, err)
}

func TestFileService_ReadError(t *testing.T) {
	fSrv := newFileService(&fileClientStub{}, false)

	_, err := fSrv.Read(context.Background(),
		&pb.ReadRequest{Input: &pb.ReadRequest_Key{Key: nil}})
	require.Equal(t, rpctypes.ErrGRPCNilKey, err)
	_, err = fSrv.Read(context.Background(),
		&pb.ReadRequest{Input: &pb.ReadRequest_Metadata{Metadata: nil}})
	require.Equal(t, rpctypes.ErrGRPCNilMetadata, err)
	_, err = fSrv.Read(context.Background(),
		&pb.ReadRequest{Input: nil})
	require.Error(t, err)

	// client errors should propagate, iff those code paths hit
	fSrv = newFileService(fileErrorClient{}, false)
	_, err = fSrv.Read(context.Background(),
		&pb.ReadRequest{Input: &pb.ReadRequest_Key{Key: []byte("key")}})
	require.Equal(t, errFooFileClient, err)
	_, err = fSrv.Read(context.Background(),
		&pb.ReadRequest{Input: &pb.ReadRequest_Metadata{Metadata: new(pb.Metadata)}})
	require.Equal(t, errFooFileClient, err)
}

func TestFileService_ReadFile(t *testing.T) {
	fSrv := newFileService(&fileClientStub{}, false)

	_, err := fSrv.ReadFile(context.Background(),
		&pb.ReadFileRequest{Input: &pb.ReadFileRequest_Key{Key: []byte("key")}, FilePath: "foo"})
	require.NoError(t, err)
	_, err = fSrv.ReadFile(context.Background(),
		&pb.ReadFileRequest{Input: &pb.ReadFileRequest_Metadata{Metadata: new(pb.Metadata)}, FilePath: "foo"})
	require.NoError(t, err)
}

func TestFileService_ReadFileError(t *testing.T) {
	fSrv := newFileService(&fileClientStub{}, false)

	_, err := fSrv.ReadFile(context.Background(),
		&pb.ReadFileRequest{Input: &pb.ReadFileRequest_Key{Key: nil}, FilePath: "foo"})
	require.Equal(t, rpctypes.ErrGRPCNilKey, err)
	_, err = fSrv.ReadFile(context.Background(),
		&pb.ReadFileRequest{Input: &pb.ReadFileRequest_Key{Key: []byte("key")}, FilePath: ""})
	require.Equal(t, rpctypes.ErrGRPCNilFilePath, err)
	_, err = fSrv.ReadFile(context.Background(),
		&pb.ReadFileRequest{Input: &pb.ReadFileRequest_Metadata{Metadata: nil}, FilePath: "foo"})
	require.Equal(t, rpctypes.ErrGRPCNilMetadata, err)
	_, err = fSrv.ReadFile(context.Background(),
		&pb.ReadFileRequest{Input: &pb.ReadFileRequest_Metadata{Metadata: new(pb.Metadata)}, FilePath: ""})
	require.Equal(t, rpctypes.ErrGRPCNilFilePath, err)
	_, err = fSrv.ReadFile(context.Background(),
		&pb.ReadFileRequest{Input: nil, FilePath: ""})
	require.Error(t, err)

	fSrv.disableLocalFSAccess = true

	_, err = fSrv.ReadFile(context.Background(),
		&pb.ReadFileRequest{Input: &pb.ReadFileRequest_Key{Key: []byte("key")}, FilePath: "foo"})
	require.Equal(t, rpctypes.ErrGRPCNoLocalFS, err)
	_, err = fSrv.ReadFile(context.Background(),
		&pb.ReadFileRequest{Input: &pb.ReadFileRequest_Metadata{Metadata: new(pb.Metadata)}, FilePath: "foo"})
	require.Equal(t, rpctypes.ErrGRPCNoLocalFS, err)

	// client errors should propagate, iff those code paths hit
	fSrv = newFileService(fileErrorClient{}, false)
	_, err = fSrv.ReadFile(context.Background(),
		&pb.ReadFileRequest{Input: &pb.ReadFileRequest_Key{Key: []byte("key")}, FilePath: "foo"})
	require.Equal(t, errFooFileClient, err)
	_, err = fSrv.ReadFile(context.Background(),
		&pb.ReadFileRequest{Input: &pb.ReadFileRequest_Metadata{Metadata: new(pb.Metadata)}, FilePath: "foo"})
	require.Equal(t, errFooFileClient, err)
}

func TestFileService_Delete(t *testing.T) {
	fSrv := newFileService(&fileClientStub{}, false)

	_, err := fSrv.Delete(context.Background(),
		&pb.DeleteRequest{Input: &pb.DeleteRequest_Key{Key: []byte("key")}})
	require.NoError(t, err)
	_, err = fSrv.Delete(context.Background(),
		&pb.DeleteRequest{Input: &pb.DeleteRequest_Metadata{Metadata: new(pb.Metadata)}})
	require.NoError(t, err)
}

func TestFileService_DeleteError(t *testing.T) {
	fSrv := newFileService(&fileClientStub{}, false)

	_, err := fSrv.Delete(context.Background(),
		&pb.DeleteRequest{Input: &pb.DeleteRequest_Key{Key: nil}})
	require.Equal(t, rpctypes.ErrGRPCNilKey, err)
	_, err = fSrv.Delete(context.Background(),
		&pb.DeleteRequest{Input: &pb.DeleteRequest_Metadata{Metadata: nil}})
	require.Equal(t, rpctypes.ErrGRPCNilMetadata, err)
	_, err = fSrv.Delete(context.Background(),
		&pb.DeleteRequest{Input: nil})
	require.Error(t, err)

	// client errors should propagate, iff those code paths hit
	fSrv = newFileService(fileErrorClient{}, false)
	_, err = fSrv.Delete(context.Background(),
		&pb.DeleteRequest{Input: &pb.DeleteRequest_Key{Key: []byte("key")}})
	require.Equal(t, errFooFileClient, err)
	_, err = fSrv.Delete(context.Background(),
		&pb.DeleteRequest{Input: &pb.DeleteRequest_Metadata{Metadata: new(pb.Metadata)}})
	require.Equal(t, errFooFileClient, err)
}

func TestFileService_Check(t *testing.T) {
	fSrv := newFileService(&fileClientStub{}, false)

	_, err := fSrv.Check(context.Background(),
		&pb.CheckRequest{Input: &pb.CheckRequest_Key{Key: []byte("key")}})
	require.NoError(t, err)
	_, err = fSrv.Check(context.Background(),
		&pb.CheckRequest{Input: &pb.CheckRequest_Metadata{Metadata: new(pb.Metadata)}})
	require.NoError(t, err)
}

func TestFileService_CheckError(t *testing.T) {
	fSrv := newFileService(&fileClientStub{}, false)

	_, err := fSrv.Check(context.Background(),
		&pb.CheckRequest{Input: &pb.CheckRequest_Key{Key: nil}})
	require.Equal(t, rpctypes.ErrGRPCNilKey, err)
	_, err = fSrv.Check(context.Background(),
		&pb.CheckRequest{Input: &pb.CheckRequest_Metadata{Metadata: nil}})
	require.Equal(t, rpctypes.ErrGRPCNilMetadata, err)
	_, err = fSrv.Check(context.Background(),
		&pb.CheckRequest{Input: nil})
	require.Error(t, err)

	// client errors should propagate, iff those code paths hit
	fSrv = newFileService(fileErrorClient{}, false)
	_, err = fSrv.Check(context.Background(),
		&pb.CheckRequest{Input: &pb.CheckRequest_Key{Key: []byte("key")}})
	require.Equal(t, errFooFileClient, err)
	_, err = fSrv.Check(context.Background(),
		&pb.CheckRequest{Input: &pb.CheckRequest_Metadata{Metadata: new(pb.Metadata)}})
	require.Equal(t, errFooFileClient, err)
}

func TestFileService_Repair(t *testing.T) {
	fSrv := newFileService(&fileClientStub{}, false)

	_, err := fSrv.Repair(context.Background(), &pb.RepairRequest{Key: []byte("key")})
	require.NoError(t, err)
}

func TestFileService_RepairError(t *testing.T) {
	fSrv := newFileService(&fileClientStub{}, false)

	_, err := fSrv.Repair(context.Background(), &pb.RepairRequest{Key: nil})
	require.Equal(t, rpctypes.ErrGRPCNilKey, err)

	// client errors should propagate, iff those code paths hit
	fSrv = newFileService(fileErrorClient{}, false)
	_, err = fSrv.Repair(context.Background(), &pb.RepairRequest{Key: []byte("key")})
	require.Equal(t, errFooFileClient, err)
}

type fileClientStub struct{}

func (stub fileClientStub) Write(key []byte, r io.Reader) (*metatypes.Metadata, error) {
	return &metatypes.Metadata{}, nil
}
func (stub fileClientStub) Read(key []byte, w io.Writer) error {
	_, err := w.Write(key)
	return err
}
func (stub fileClientStub) ReadWithMeta(meta metatypes.Metadata, w io.Writer) error {
	_, err := w.Write(append([]byte("hello"), meta.Key...))
	return err
}
func (stub fileClientStub) Delete(key []byte) error                                  { return nil }
func (stub fileClientStub) DeleteWithMeta(meta metatypes.Metadata) error             { return nil }
func (stub fileClientStub) Check(key []byte, fast bool) (storage.CheckStatus, error) { return 0, nil }
func (stub fileClientStub) CheckWithMeta(meta metatypes.Metadata, fast bool) (storage.CheckStatus, error) {
	return 0, nil
}
func (stub fileClientStub) Repair(key []byte) (*metatypes.Metadata, error) {
	return &metatypes.Metadata{}, nil
}

var errFooFileClient = errors.New("fileErrorClient: foo")

type fileErrorClient struct{}

func (c fileErrorClient) Write(key []byte, r io.Reader) (*metatypes.Metadata, error) {
	return nil, errFooFileClient
}
func (c fileErrorClient) Read(key []byte, w io.Writer) error {
	return errFooFileClient
}
func (c fileErrorClient) ReadWithMeta(meta metatypes.Metadata, w io.Writer) error {
	return errFooFileClient
}
func (c fileErrorClient) Delete(key []byte) error                      { return errFooFileClient }
func (c fileErrorClient) DeleteWithMeta(meta metatypes.Metadata) error { return errFooFileClient }
func (c fileErrorClient) Check(key []byte, fast bool) (storage.CheckStatus, error) {
	return 0, errFooFileClient
}
func (c fileErrorClient) CheckWithMeta(meta metatypes.Metadata, fast bool) (storage.CheckStatus, error) {
	return 0, errFooFileClient
}
func (c fileErrorClient) Repair(key []byte) (*metatypes.Metadata, error) {
	return nil, errFooFileClient
}

var (
	_ fileClient = fileClientStub{}
	_ fileClient = fileErrorClient{}
)
