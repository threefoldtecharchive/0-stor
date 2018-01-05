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
	"io/ioutil"
	"net"
	"testing"

	"github.com/zero-os/0-stor/daemon/api/grpc/rpctypes"
	pb "github.com/zero-os/0-stor/daemon/api/grpc/schema"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

func TestFileAPI_Client_Key_ReadWriteDeleteCheck(t *testing.T) {
	require := require.New(t)

	daemon := newTestDaemon(t)
	require.NotNil(daemon)
	defer daemon.Close()

	lis, err := net.Listen("tcp", "localhost:0")
	require.NoError(err)
	go func() {
		err := daemon.Serve(lis)
		if err != nil {
			panic(err)
		}
	}()

	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
	require.NoError(err, "can't connect to the server")

	client := pb.NewFileServiceClient(conn)
	require.NotNil(client)

	ctx := context.Background()

	writeResp, err := client.Write(ctx, &pb.WriteRequest{Key: []byte("foo"), Data: []byte("bar")})
	require.NoError(err)
	require.NotNil(writeResp)
	metadata := writeResp.GetMetadata()
	require.NotNil(metadata)
	require.Equal([]byte("foo"), metadata.GetKey())

	readResp, err := client.Read(ctx, &pb.ReadRequest{Input: &pb.ReadRequest_Key{Key: []byte("foo")}})
	require.NoError(err)
	require.NotNil(readResp)
	data := readResp.GetData()
	require.Equal([]byte("bar"), data)

	checkResp, err := client.Check(ctx, &pb.CheckRequest{Input: &pb.CheckRequest_Key{Key: []byte("foo")}})
	require.NoError(err)
	require.NotNil(checkResp)
	status := checkResp.GetStatus()
	require.Equal(pb.CheckStatusOptimal, status)

	deleteResp, err := client.Delete(ctx, &pb.DeleteRequest{Input: &pb.DeleteRequest_Key{Key: []byte("foo")}})
	require.NoError(err)
	require.NotNil(deleteResp)

	_, err = client.Check(ctx, &pb.CheckRequest{Input: &pb.CheckRequest_Key{Key: []byte("foo")}})
	require.Equal(rpctypes.ErrGRPCKeyNotFound, err)

	_, err = client.Read(ctx, &pb.ReadRequest{Input: &pb.ReadRequest_Key{Key: []byte("foo")}})
	require.Equal(rpctypes.ErrGRPCKeyNotFound, err)
}

func TestFileAPI_Client_Meta_ReadWriteDeleteCheck(t *testing.T) {
	require := require.New(t)

	daemon := newTestDaemon(t)
	require.NotNil(daemon)
	defer daemon.Close()

	lis, err := net.Listen("tcp", "localhost:0")
	require.NoError(err)
	go func() {
		err := daemon.Serve(lis)
		if err != nil {
			panic(err)
		}
	}()

	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
	require.NoError(err, "can't connect to the server")

	client := pb.NewFileServiceClient(conn)
	require.NotNil(client)

	ctx := context.Background()

	writeResp, err := client.Write(ctx, &pb.WriteRequest{Key: []byte("foo"), Data: []byte("bar")})
	require.NoError(err)
	require.NotNil(writeResp)
	metadata := writeResp.GetMetadata()
	require.NotNil(metadata)
	require.Equal([]byte("foo"), metadata.GetKey())

	readResp, err := client.Read(ctx, &pb.ReadRequest{Input: &pb.ReadRequest_Metadata{Metadata: metadata}})
	require.NoError(err)
	require.NotNil(readResp)
	data := readResp.GetData()
	require.Equal([]byte("bar"), data)

	checkResp, err := client.Check(ctx, &pb.CheckRequest{Input: &pb.CheckRequest_Metadata{Metadata: metadata}})
	require.NoError(err)
	require.NotNil(checkResp)
	status := checkResp.GetStatus()
	require.Equal(pb.CheckStatusOptimal, status)

	deleteResp, err := client.Delete(ctx, &pb.DeleteRequest{Input: &pb.DeleteRequest_Metadata{Metadata: metadata}})
	require.NoError(err)
	require.NotNil(deleteResp)

	checkResp, err = client.Check(ctx, &pb.CheckRequest{Input: &pb.CheckRequest_Metadata{Metadata: metadata}})
	require.NoError(err)
	require.NotNil(checkResp)
	require.Equal(pb.CheckStatusInvalid, checkResp.GetStatus())

	_, err = client.Read(ctx, &pb.ReadRequest{Input: &pb.ReadRequest_Metadata{Metadata: metadata}})
	require.Equal(rpctypes.ErrGRPCKeyNotFound, err)
}

func TestFileAPI_Client_Key_ReadFileWriteFileCheck(t *testing.T) {
	require := require.New(t)

	daemon := newTestDaemon(t)
	require.NotNil(daemon)
	defer daemon.Close()

	lis, err := net.Listen("tcp", "localhost:0")
	require.NoError(err)
	go func() {
		err := daemon.Serve(lis)
		if err != nil {
			panic(err)
		}
	}()

	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
	require.NoError(err, "can't connect to the server")

	client := pb.NewFileServiceClient(conn)
	require.NotNil(client)

	ctx := context.Background()

	inputFile := newInMemoryFile()
	defer inputFile.Close()
	_, err = inputFile.Write([]byte("bar"))
	require.NoError(err)

	writeResp, err := client.WriteFile(ctx, &pb.WriteFileRequest{Key: []byte("foo"), FilePath: inputFile.Name()})
	require.NoError(err)
	require.NotNil(writeResp)
	metadata := writeResp.GetMetadata()
	require.NotNil(metadata)
	require.Equal([]byte("foo"), metadata.GetKey())

	outputFile := newInMemoryFile()
	defer outputFile.Close()

	readResp, err := client.ReadFile(ctx,
		&pb.ReadFileRequest{Input: &pb.ReadFileRequest_Key{Key: []byte("foo")},
			FilePath: outputFile.Name(), SynchronousIO: true})
	require.NoError(err)
	require.NotNil(readResp)

	data, err := ioutil.ReadAll(outputFile)
	require.NoError(err)
	require.Equal([]byte("bar"), data)

	checkResp, err := client.Check(ctx, &pb.CheckRequest{Input: &pb.CheckRequest_Key{Key: []byte("foo")}})
	require.NoError(err)
	require.NotNil(checkResp)
	status := checkResp.GetStatus()
	require.Equal(pb.CheckStatusOptimal, status)
}

func TestFileAPI_Client_Meta_ReadFileWriteFileCheck(t *testing.T) {
	require := require.New(t)

	daemon := newTestDaemon(t)
	require.NotNil(daemon)
	defer daemon.Close()

	lis, err := net.Listen("tcp", "localhost:0")
	require.NoError(err)
	go func() {
		err := daemon.Serve(lis)
		if err != nil {
			panic(err)
		}
	}()

	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
	require.NoError(err, "can't connect to the server")

	client := pb.NewFileServiceClient(conn)
	require.NotNil(client)

	ctx := context.Background()

	inputFile := newInMemoryFile()
	defer inputFile.Close()
	_, err = inputFile.Write([]byte("bar"))
	require.NoError(err)

	writeResp, err := client.WriteFile(ctx, &pb.WriteFileRequest{Key: []byte("foo"), FilePath: inputFile.Name()})
	require.NoError(err)
	require.NotNil(writeResp)
	metadata := writeResp.GetMetadata()
	require.NotNil(metadata)
	require.Equal([]byte("foo"), metadata.GetKey())

	outputFile := newInMemoryFile()
	defer outputFile.Close()

	readResp, err := client.ReadFile(ctx,
		&pb.ReadFileRequest{Input: &pb.ReadFileRequest_Metadata{Metadata: metadata},
			FilePath: outputFile.Name(), SynchronousIO: true})
	require.NoError(err)
	require.NotNil(readResp)

	data, err := ioutil.ReadAll(outputFile)
	require.NoError(err)
	require.Equal([]byte("bar"), data)

	checkResp, err := client.Check(ctx, &pb.CheckRequest{Input: &pb.CheckRequest_Metadata{Metadata: metadata}})
	require.NoError(err)
	require.NotNil(checkResp)
	status := checkResp.GetStatus()
	require.Equal(pb.CheckStatusOptimal, status)
}

func TestFileAPI_Client_Key_WriteReadStream(t *testing.T) {
	require := require.New(t)

	daemon := newTestDaemon(t)
	require.NotNil(daemon)
	defer daemon.Close()

	lis, err := net.Listen("tcp", "localhost:0")
	require.NoError(err)
	go func() {
		err := daemon.Serve(lis)
		if err != nil {
			panic(err)
		}
	}()

	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
	require.NoError(err, "can't connect to the server")

	client := pb.NewFileServiceClient(conn)
	require.NotNil(client)

	ctx := context.Background()

	writeResp, err := client.Write(ctx, &pb.WriteRequest{Key: []byte("foo"), Data: []byte("bar")})
	require.NoError(err)
	require.NotNil(writeResp)
	metadata := writeResp.GetMetadata()
	require.NotNil(metadata)
	require.Equal([]byte("foo"), metadata.GetKey())

	readResp, err := client.ReadStream(ctx, &pb.ReadStreamRequest{
		Input:     &pb.ReadStreamRequest_Key{Key: []byte("foo")},
		ChunkSize: 2,
	})
	require.NoError(err)
	require.NotNil(readResp)

	buf := bytes.NewBuffer(nil)
	for {
		resp, err := readResp.Recv()
		if err != nil {
			require.Equal(io.EOF, err)
			break
		}
		data := resp.GetDataChunk()
		length := len(data)
		require.True(length == 1 || length == 2)
		n, err := buf.Write(data)
		require.NoError(err)
		require.Equal(length, n)
	}
	data := buf.Bytes()
	require.Len(data, 3)
	require.Equal("bar", string(data))
}

func TestFileAPI_Client_Meta_WriteReadStream(t *testing.T) {
	require := require.New(t)

	daemon := newTestDaemon(t)
	require.NotNil(daemon)
	defer daemon.Close()

	lis, err := net.Listen("tcp", "localhost:0")
	require.NoError(err)
	go func() {
		err := daemon.Serve(lis)
		if err != nil {
			panic(err)
		}
	}()

	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
	require.NoError(err, "can't connect to the server")

	client := pb.NewFileServiceClient(conn)
	require.NotNil(client)

	ctx := context.Background()

	writeResp, err := client.Write(ctx, &pb.WriteRequest{Key: []byte("foo"), Data: []byte("bar")})
	require.NoError(err)
	require.NotNil(writeResp)
	metadata := writeResp.GetMetadata()
	require.NotNil(metadata)
	require.Equal([]byte("foo"), metadata.GetKey())

	readResp, err := client.ReadStream(ctx, &pb.ReadStreamRequest{
		Input:     &pb.ReadStreamRequest_Metadata{Metadata: metadata},
		ChunkSize: 2,
	})
	require.NoError(err)
	require.NotNil(readResp)

	buf := bytes.NewBuffer(nil)
	for {
		resp, err := readResp.Recv()
		if err != nil {
			require.Equal(io.EOF, err)
			break
		}
		data := resp.GetDataChunk()
		length := len(data)
		require.True(length == 1 || length == 2)
		n, err := buf.Write(data)
		require.NoError(err)
		require.Equal(length, n)
	}
	data := buf.Bytes()
	require.Len(data, 3)
	require.Equal("bar", string(data))
}

func TestFileAPI_Client_WriteStreamRead(t *testing.T) {
	require := require.New(t)

	daemon := newTestDaemon(t)
	require.NotNil(daemon)
	defer daemon.Close()

	lis, err := net.Listen("tcp", "localhost:0")
	require.NoError(err)
	go func() {
		err := daemon.Serve(lis)
		if err != nil {
			panic(err)
		}
	}()

	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
	require.NoError(err, "can't connect to the server")

	client := pb.NewFileServiceClient(conn)
	require.NotNil(client)

	ctx := context.Background()

	stream, err := client.WriteStream(ctx)
	require.NoError(err)
	require.NotNil(stream)

	// assemble request
	request := &pb.WriteStreamRequest{
		Input: &pb.WriteStreamRequest_Metadata_{
			Metadata: &pb.WriteStreamRequest_Metadata{Key: []byte("foo")}}}

	// first send out key
	err = stream.Send(request)
	require.NoError(err)

	// then send all our chunks

	requestData := &pb.WriteStreamRequest_Data_{Data: &pb.WriteStreamRequest_Data{}}
	request.Input = requestData

	requestData.Data.DataChunk = []byte{'a'}
	err = stream.Send(request)
	require.NoError(err)
	requestData.Data.DataChunk = []byte{'n', 's', 'w'}
	err = stream.Send(request)
	require.NoError(err)
	requestData.Data.DataChunk = []byte{'e', 'r'}
	err = stream.Send(request)
	require.NoError(err)
	resp, err := stream.CloseAndRecv()
	require.NoError(err)
	require.NotNil(resp)

	readResp, err := client.Read(ctx, &pb.ReadRequest{Input: &pb.ReadRequest_Key{Key: []byte("foo")}})
	require.NoError(err)
	require.NotNil(readResp)
	data := readResp.GetData()
	require.Equal([]byte("answer"), data)

	metadata := resp.GetMetadata()
	require.NotNil(metadata)

	readResp, err = client.Read(ctx, &pb.ReadRequest{Input: &pb.ReadRequest_Metadata{Metadata: metadata}})
	require.NoError(err)
	require.NotNil(readResp)
	data = readResp.GetData()
	require.Equal([]byte("answer"), data)
}
