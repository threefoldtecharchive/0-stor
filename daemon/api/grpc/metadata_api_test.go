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
	"fmt"
	"io"
	"net"
	"testing"

	"github.com/threefoldtech/0-stor/client/metastor/db"
	"github.com/threefoldtech/0-stor/client/metastor/metatypes"
	"github.com/threefoldtech/0-stor/daemon/api/grpc/rpctypes"
	pb "github.com/threefoldtech/0-stor/daemon/api/grpc/schema"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

func TestSetMetadata(t *testing.T) {
	service := newMetadataService(metadataClientStub{})

	_, err := service.SetMetadata(context.Background(), &pb.SetMetadataRequest{
		Metadata: &pb.Metadata{},
	})
	require.NoError(t, err)
}

func TestSetMetadataError(t *testing.T) {
	service := newMetadataService(metadataClientStub{})

	_, err := service.SetMetadata(context.Background(), nil)
	require.Equal(t, rpctypes.ErrGRPCNilMetadata, err)

	service.client = metadataErrorClient{}
	_, err = service.SetMetadata(context.Background(), &pb.SetMetadataRequest{
		Metadata: &pb.Metadata{},
	})
	require.Equal(t, errFooMetadataClient, err)
}

func TestGetMetadata(t *testing.T) {
	service := newMetadataService(metadataClientStub{})

	_, err := service.GetMetadata(context.Background(), &pb.GetMetadataRequest{
		Key: []byte("foo"),
	})
	require.NoError(t, err)
}

func TestGetMetadataError(t *testing.T) {
	service := newMetadataService(metadataClientStub{})

	_, err := service.GetMetadata(context.Background(), nil)
	require.Equal(t, rpctypes.ErrGRPCNilKey, err)

	service.client = metadataErrorClient{}
	_, err = service.GetMetadata(context.Background(), &pb.GetMetadataRequest{
		Key: []byte("foo"),
	})
	require.Equal(t, errFooMetadataClient, err)
}

func TestDeleteMetadata(t *testing.T) {
	service := newMetadataService(metadataClientStub{})

	_, err := service.DeleteMetadata(context.Background(), &pb.DeleteMetadataRequest{
		Key: []byte("foo"),
	})
	require.NoError(t, err)
}

func TestDeleteMetadataError(t *testing.T) {
	service := newMetadataService(metadataClientStub{})

	_, err := service.DeleteMetadata(context.Background(), nil)
	require.Equal(t, rpctypes.ErrGRPCNilKey, err)

	service.client = metadataErrorClient{}
	_, err = service.DeleteMetadata(context.Background(), &pb.DeleteMetadataRequest{
		Key: []byte("foo"),
	})
	require.Equal(t, errFooMetadataClient, err)
}

func TestListKeys(t *testing.T) {
	require := require.New(t)

	// creates test daemon
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

	// creates grpc client
	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
	require.NoError(err)

	client := pb.NewMetadataServiceClient(conn)
	require.NotNil(client)

	ctx := context.Background()

	// populates the data
	var (
		keys       [][]byte
		listedKeys [][]byte
	)
	for i := 0; i < 10; i++ {
		key := []byte(fmt.Sprintf("key_%d", i))
		_, err := client.SetMetadata(ctx, &pb.SetMetadataRequest{
			Metadata: &pb.Metadata{Key: key},
		})
		require.NoError(err)

		keys = append(keys, key)
	}

	// list it
	stream, err := client.ListKeys(ctx, &pb.ListMetadataKeysRequest{})
	for {
		resp, err := stream.Recv()
		if err != nil {
			require.Equal(io.EOF, err)
			break
		}
		listedKeys = append(listedKeys, resp.Key)
	}
	require.Equal(keys, listedKeys)
}

type metadataClientStub struct{}

func (stub metadataClientStub) SetMetadata(metadata metatypes.Metadata) error {
	return nil
}
func (stub metadataClientStub) GetMetadata(key []byte) (*metatypes.Metadata, error) {
	return &metatypes.Metadata{Key: key}, nil
}
func (stub metadataClientStub) DeleteMetadata(key []byte) error {
	return nil
}

func (stub metadataClientStub) ListKeys(cb db.ListCallback) error {
	return nil
}

var errFooMetadataClient = errors.New("metadataErrorClient: foo")

type metadataErrorClient struct{}

func (stub metadataErrorClient) SetMetadata(metadata metatypes.Metadata) error {
	return errFooMetadataClient
}
func (stub metadataErrorClient) GetMetadata(key []byte) (*metatypes.Metadata, error) {
	return nil, errFooMetadataClient
}
func (stub metadataErrorClient) DeleteMetadata(key []byte) error {
	return errFooMetadataClient
}

func (stub metadataErrorClient) ListKeys(cb db.ListCallback) error {
	return errFooMetadataClient
}

var (
	_ metadataClient = metadataClientStub{}
)
