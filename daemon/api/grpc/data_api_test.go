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

	"github.com/zero-os/0-stor/client/datastor/pipeline/storage"
	"github.com/zero-os/0-stor/client/metastor/metatypes"
	"github.com/zero-os/0-stor/daemon/api/grpc/rpctypes"
	pb "github.com/zero-os/0-stor/daemon/api/grpc/schema"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

func TestDataService_Write(t *testing.T) {
	dSrv := newDataService(&dataClientStub{}, false)

	_, err := dSrv.Write(context.Background(),
		&pb.DataWriteRequest{Data: []byte("data")})
	require.NoError(t, err)
}

func TestDataService_WriteError(t *testing.T) {
	dSrv := newDataService(&dataClientStub{}, false)

	_, err := dSrv.Write(context.Background(),
		&pb.DataWriteRequest{Data: nil})
	require.Equal(t, rpctypes.ErrGRPCNilData, err)

	// client errors should propagate, iff those code paths hit
	dSrv = newDataService(dataErrorClient{}, false)
	_, err = dSrv.Write(context.Background(),
		&pb.DataWriteRequest{Data: []byte("data")})
	require.Equal(t, errFooDataClient, err)
}

func TestDataService_WriteFile(t *testing.T) {
	dSrv := newDataService(&dataClientStub{}, false)

	_, err := dSrv.WriteFile(context.Background(),
		&pb.DataWriteFileRequest{FilePath: "foo"})
	require.NoError(t, err)
}

func TestDataService_WriteFileError(t *testing.T) {
	dSrv := newDataService(&dataClientStub{}, false)

	_, err := dSrv.WriteFile(context.Background(),
		&pb.DataWriteFileRequest{FilePath: ""})
	require.Equal(t, rpctypes.ErrGRPCNilFilePath, err)

	dSrv.disableLocalFSAccess = true
	_, err = dSrv.WriteFile(context.Background(),
		&pb.DataWriteFileRequest{FilePath: "foo"})
	require.Equal(t, rpctypes.ErrGRPCNoLocalFS, err)

	// client errors should propagate, iff those code paths hit
	dSrv = newDataService(dataErrorClient{}, false)
	_, err = dSrv.WriteFile(context.Background(),
		&pb.DataWriteFileRequest{FilePath: "foo"})
	require.Equal(t, errFooDataClient, err)
}

func TestDataService_Read(t *testing.T) {
	dSrv := newDataService(&dataClientStub{}, false)

	_, err := dSrv.Read(context.Background(),
		&pb.DataReadRequest{Chunks: []pb.Chunk{pb.Chunk{}}})
	require.NoError(t, err)
}

func TestDataService_ReadError(t *testing.T) {
	dSrv := newDataService(&dataClientStub{}, false)

	_, err := dSrv.Read(context.Background(),
		&pb.DataReadRequest{Chunks: nil})
	require.Equal(t, rpctypes.ErrGRPCNilChunks, err)

	// client errors should propagate, iff those code paths hit
	dSrv = newDataService(dataErrorClient{}, false)
	_, err = dSrv.Read(context.Background(),
		&pb.DataReadRequest{Chunks: []pb.Chunk{pb.Chunk{}}})
	require.Equal(t, errFooDataClient, err)
}

func TestDataService_ReadFile(t *testing.T) {
	dSrv := newDataService(&dataClientStub{}, false)

	_, err := dSrv.ReadFile(context.Background(),
		&pb.DataReadFileRequest{Chunks: []pb.Chunk{pb.Chunk{}}, FilePath: "foo"})
	require.NoError(t, err)
}

func TestDataService_ReadFileError(t *testing.T) {
	dSrv := newDataService(&dataClientStub{}, false)

	_, err := dSrv.ReadFile(context.Background(),
		&pb.DataReadFileRequest{Chunks: nil, FilePath: "foo"})
	require.Equal(t, rpctypes.ErrGRPCNilChunks, err)
	_, err = dSrv.ReadFile(context.Background(),
		&pb.DataReadFileRequest{Chunks: []pb.Chunk{pb.Chunk{}}, FilePath: ""})
	require.Equal(t, rpctypes.ErrGRPCNilFilePath, err)

	dSrv.disableLocalFSAccess = true

	_, err = dSrv.ReadFile(context.Background(),
		&pb.DataReadFileRequest{Chunks: []pb.Chunk{pb.Chunk{}}, FilePath: "foo"})
	require.Equal(t, rpctypes.ErrGRPCNoLocalFS, err)

	// client errors should propagate, iff those code paths hit
	dSrv = newDataService(dataErrorClient{}, false)
	_, err = dSrv.ReadFile(context.Background(),
		&pb.DataReadFileRequest{Chunks: []pb.Chunk{pb.Chunk{}}, FilePath: "foo"})
	require.Equal(t, errFooDataClient, err)
}

func TestDataService_Delete(t *testing.T) {
	dSrv := newDataService(&dataClientStub{}, false)

	_, err := dSrv.Delete(context.Background(),
		&pb.DataDeleteRequest{Chunks: []pb.Chunk{pb.Chunk{}}})
	require.NoError(t, err)
}

func TestDataService_DeleteError(t *testing.T) {
	dSrv := newDataService(&dataClientStub{}, false)

	_, err := dSrv.Delete(context.Background(),
		&pb.DataDeleteRequest{Chunks: nil})
	require.Equal(t, rpctypes.ErrGRPCNilChunks, err)

	// client errors should propagate, iff those code paths hit
	dSrv = newDataService(dataErrorClient{}, false)
	_, err = dSrv.Delete(context.Background(),
		&pb.DataDeleteRequest{Chunks: []pb.Chunk{pb.Chunk{}}})
	require.Equal(t, errFooDataClient, err)
}

func TestDataService_Check(t *testing.T) {
	dSrv := newDataService(&dataClientStub{}, false)

	_, err := dSrv.Check(context.Background(),
		&pb.DataCheckRequest{Chunks: []pb.Chunk{pb.Chunk{}}})
	require.NoError(t, err)
}

func TestDataService_CheckError(t *testing.T) {
	dSrv := newDataService(&dataClientStub{}, false)

	_, err := dSrv.Check(context.Background(),
		&pb.DataCheckRequest{Chunks: nil})
	require.Equal(t, rpctypes.ErrGRPCNilChunks, err)

	// client errors should propagate, iff those code paths hit
	dSrv = newDataService(dataErrorClient{}, false)
	_, err = dSrv.Check(context.Background(),
		&pb.DataCheckRequest{Chunks: []pb.Chunk{pb.Chunk{}}})
	require.Equal(t, errFooDataClient, err)
}

func TestDataService_Repair(t *testing.T) {
	dSrv := newDataService(&dataClientStub{}, false)

	_, err := dSrv.Repair(context.Background(), &pb.DataRepairRequest{Chunks: []pb.Chunk{pb.Chunk{}}})
	require.NoError(t, err)
}

func TestDataService_RepairError(t *testing.T) {
	dSrv := newDataService(&dataClientStub{}, false)

	_, err := dSrv.Repair(context.Background(), &pb.DataRepairRequest{Chunks: nil})
	require.Equal(t, rpctypes.ErrGRPCNilChunks, err)

	// client errors should propagate, iff those code paths hit
	dSrv = newDataService(dataErrorClient{}, false)
	_, err = dSrv.Repair(context.Background(), &pb.DataRepairRequest{Chunks: []pb.Chunk{pb.Chunk{}}})
	require.Equal(t, errFooDataClient, err)
}

type dataClientStub struct{}

func (stub dataClientStub) Write(r io.Reader) ([]metatypes.Chunk, error) {
	return nil, nil
}
func (stub dataClientStub) Read(chunks []metatypes.Chunk, w io.Writer) error {
	_, err := w.Write([]byte("hello"))
	return err
}
func (stub dataClientStub) Delete(chunks []metatypes.Chunk) error {
	return nil
}
func (stub dataClientStub) Check(chunks []metatypes.Chunk, fast bool) (storage.CheckStatus, error) {
	return 0, nil
}
func (stub dataClientStub) Repair(chunks []metatypes.Chunk) ([]metatypes.Chunk, error) {
	return nil, nil
}

var errFooDataClient = errors.New("dataErrorClient: foo")

type dataErrorClient struct{}

func (stub dataErrorClient) Write(r io.Reader) ([]metatypes.Chunk, error) {
	return nil, errFooDataClient
}
func (stub dataErrorClient) Read(chunks []metatypes.Chunk, w io.Writer) error {
	return errFooDataClient
}
func (stub dataErrorClient) Delete(chunks []metatypes.Chunk) error {
	return errFooDataClient
}
func (stub dataErrorClient) Check(chunks []metatypes.Chunk, fast bool) (storage.CheckStatus, error) {
	return 0, errFooDataClient
}
func (stub dataErrorClient) Repair(chunks []metatypes.Chunk) ([]metatypes.Chunk, error) {
	return nil, errFooDataClient
}

var (
	_ dataClient = dataClientStub{}
	_ dataClient = dataErrorClient{}
)
