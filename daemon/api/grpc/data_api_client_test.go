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
	"google.golang.org/grpc/metadata"
)

func TestDataAPI_Client_ReadWriteDeleteCheck(t *testing.T) {
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

	client := pb.NewDataServiceClient(conn)
	require.NotNil(client)

	ctx := context.Background()

	writeResp, err := client.Write(ctx, &pb.DataWriteRequest{Data: []byte("bar")})
	require.NoError(err)
	require.NotNil(writeResp)
	chunks := writeResp.GetChunks()
	require.Len(chunks, 1)
	require.Len(chunks[0].GetObjects(), 1)

	readResp, err := client.Read(ctx, &pb.DataReadRequest{Chunks: chunks})
	require.NoError(err)
	require.NotNil(readResp)
	data := readResp.GetData()
	require.Equal([]byte("bar"), data)

	checkResp, err := client.Check(ctx, &pb.DataCheckRequest{Chunks: chunks})
	require.NoError(err)
	require.NotNil(checkResp)
	status := checkResp.GetStatus()
	require.Equal(pb.CheckStatusOptimal, status)

	deleteResp, err := client.Delete(ctx, &pb.DataDeleteRequest{Chunks: chunks})
	require.NoError(err)
	require.NotNil(deleteResp)

	checkResp, err = client.Check(ctx, &pb.DataCheckRequest{Chunks: chunks})
	require.NoError(err)
	require.NotNil(checkResp)
	status = checkResp.GetStatus()
	require.Equal(pb.CheckStatusInvalid, status)

	_, err = client.Read(ctx, &pb.DataReadRequest{Chunks: chunks})
	require.Equal(rpctypes.ErrGRPCKeyNotFound, err)
}

func TestDataAPI_Client_ReadFileWriteFileCheck(t *testing.T) {
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

	client := pb.NewDataServiceClient(conn)
	require.NotNil(client)

	ctx := context.Background()

	inputFile := newInMemoryFile()
	defer inputFile.Close()
	_, err = inputFile.Write([]byte("bar"))
	require.NoError(err)

	writeResp, err := client.WriteFile(ctx, &pb.DataWriteFileRequest{FilePath: inputFile.Name()})
	require.NoError(err)
	require.NotNil(writeResp)
	chunks := writeResp.GetChunks()
	require.Len(chunks, 1)
	require.Len(chunks[0].GetObjects(), 1)

	outputFile := newInMemoryFile()
	defer outputFile.Close()

	readResp, err := client.ReadFile(ctx,
		&pb.DataReadFileRequest{Chunks: chunks,
			FilePath: outputFile.Name(), SynchronousIO: true})
	require.NoError(err)
	require.NotNil(readResp)

	data, err := ioutil.ReadAll(outputFile)
	require.NoError(err)
	require.Equal([]byte("bar"), data)

	checkResp, err := client.Check(ctx, &pb.DataCheckRequest{Chunks: chunks})
	require.NoError(err)
	require.NotNil(checkResp)
	status := checkResp.GetStatus()
	require.Equal(pb.CheckStatusOptimal, status)
}

func TestDataAPI_Client_WriteReadStream(t *testing.T) {
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

	client := pb.NewDataServiceClient(conn)
	require.NotNil(client)

	ctx := context.Background()

	writeResp, err := client.Write(ctx, &pb.DataWriteRequest{Data: []byte("bar")})
	require.NoError(err)
	require.NotNil(writeResp)
	chunks := writeResp.GetChunks()
	require.Len(chunks, 1)

	readResp, err := client.ReadStream(ctx, &pb.DataReadStreamRequest{
		Chunks:    chunks,
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

func TestDataAPI_Client_WriteStreamRead(t *testing.T) {
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

	client := pb.NewDataServiceClient(conn)
	require.NotNil(client)

	ctx := context.Background()

	md := metadata.Pairs(rpctypes.MetaKeyTag, "foo")
	writeCtx := metadata.NewOutgoingContext(ctx, md)
	stream, err := client.WriteStream(writeCtx)
	require.NoError(err)
	require.NotNil(stream)

	err = stream.Send(&pb.DataWriteStreamRequest{DataChunk: []byte{'a'}})
	require.NoError(err)
	err = stream.Send(&pb.DataWriteStreamRequest{DataChunk: []byte{'n', 's', 'w'}})
	require.NoError(err)
	err = stream.Send(&pb.DataWriteStreamRequest{DataChunk: []byte{'e', 'r'}})
	require.NoError(err)
	resp, err := stream.CloseAndRecv()
	require.NoError(err)
	require.NotNil(resp)
	chunks := resp.GetChunks()
	require.Len(chunks, 1)

	readResp, err := client.Read(ctx, &pb.DataReadRequest{Chunks: chunks})
	require.NoError(err)
	require.NotNil(readResp)
	data := readResp.GetData()
	require.Equal([]byte("answer"), data)
}
