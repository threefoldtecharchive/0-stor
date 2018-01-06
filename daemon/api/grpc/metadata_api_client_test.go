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
	"net"
	"testing"

	"github.com/zero-os/0-stor/daemon/api/grpc/rpctypes"
	pb "github.com/zero-os/0-stor/daemon/api/grpc/schema"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"golang.org/x/net/context"
)

func TestMetadataAPI_Client_SetGetDelete(t *testing.T) {
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

	client := pb.NewMetadataServiceClient(conn)
	require.NotNil(client)

	ctx := context.Background()

	_, err = client.GetMetadata(ctx, &pb.GetMetadataRequest{Key: []byte("foo")})
	require.Equal(rpctypes.ErrGRPCKeyNotFound, err)

	_, err = client.DeleteMetadata(ctx, &pb.DeleteMetadataRequest{Key: []byte("foo")})
	require.NoError(err, "it should be ok to delete non-existing data")

	_, err = client.SetMetadata(ctx, &pb.SetMetadataRequest{Metadata: &pb.Metadata{
		Key:            []byte("foo"),
		SizeInBytes:    1,
		CreationEpoch:  2,
		LastWriteEpoch: 4,
	}})
	require.NoError(err)

	getResp, err := client.GetMetadata(ctx, &pb.GetMetadataRequest{Key: []byte("foo")})
	require.NoError(err)
	require.NotNil(getResp)
	metadata := getResp.GetMetadata()
	require.NotNil(metadata)
	require.Equal(metadata.GetKey(), []byte("foo"))
	require.Equal(int64(1), metadata.GetSizeInBytes())
	require.Equal(int64(2), metadata.GetCreationEpoch())
	require.Equal(int64(4), metadata.GetLastWriteEpoch())

	_, err = client.DeleteMetadata(ctx, &pb.DeleteMetadataRequest{Key: []byte("foo")})
	require.NoError(err, "it should be ok to delete existing data")

	_, err = client.GetMetadata(ctx, &pb.GetMetadataRequest{Key: []byte("foo")})
	require.Equal(rpctypes.ErrGRPCKeyNotFound, err)
}
