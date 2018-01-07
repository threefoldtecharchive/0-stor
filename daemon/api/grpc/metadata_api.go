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
	"github.com/zero-os/0-stor/client/metastor"
	"github.com/zero-os/0-stor/client/metastor/metatypes"
	"github.com/zero-os/0-stor/daemon/api/grpc/rpctypes"
	pb "github.com/zero-os/0-stor/daemon/api/grpc/schema"

	"golang.org/x/net/context"
)

func newMetadataService(client metadataClient) *metadataService {
	return &metadataService{client: client}
}

// metadataService is used to set, get and delete metadata directly
// using the metadata storage client.
type metadataService struct {
	client metadataClient
}

// SetMetadata implements MetadataServiceServer.SetMetadata
func (service *metadataService) SetMetadata(ctx context.Context, req *pb.SetMetadataRequest) (*pb.SetMetadataResponse, error) {
	metadata := req.GetMetadata()
	if metadata == nil {
		return nil, rpctypes.ErrGRPCNilMetadata
	}

	// convert pb.Metadata into a metatypes.Metadata structure
	input := convertProtoToInMemoryMetadata(metadata)

	err := service.client.SetMetadata(input)
	if err != nil {
		return nil, mapMetaStorError(err)
	}
	return &pb.SetMetadataResponse{}, nil
}

// GetMetadata implements MetadataServiceServer.GetMetadata
func (service *metadataService) GetMetadata(ctx context.Context, req *pb.GetMetadataRequest) (*pb.GetMetadataResponse, error) {
	key := req.GetKey()
	if len(key) == 0 {
		return nil, rpctypes.ErrGRPCNilKey
	}

	metadata, err := service.client.GetMetadata(key)
	if err != nil {
		return nil, mapMetaStorError(err)
	}

	// convert metatypes.Metadata into a pb.Metadata structure
	output := convertInMemoryToProtoMetadata(*metadata)
	return &pb.GetMetadataResponse{Metadata: output}, nil
}

// DeleteMetadata implements MetadataServiceServer.DeleteMetadata
func (service *metadataService) DeleteMetadata(ctx context.Context, req *pb.DeleteMetadataRequest) (*pb.DeleteMetadataResponse, error) {
	key := req.GetKey()
	if len(key) == 0 {
		return nil, rpctypes.ErrGRPCNilKey
	}

	err := service.client.DeleteMetadata(key)
	if err != nil {
		return nil, mapMetaStorError(err)
	}
	return &pb.DeleteMetadataResponse{}, nil
}

// metadataClient is used by the metadataService,
// to run the actual business logic of the service.
type metadataClient interface {
	SetMetadata(metadata metatypes.Metadata) error
	GetMetadata(key []byte) (*metatypes.Metadata, error)
	DeleteMetadata(key []byte) error
}

var (
	_ pb.MetadataServiceServer = (*metadataService)(nil)
	_ metadataClient           = (*metastor.Client)(nil)
)
