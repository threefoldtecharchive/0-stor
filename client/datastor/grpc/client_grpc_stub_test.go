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
	"io"

	pb "github.com/zero-os/0-stor/server/api/grpc/schema"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type stubObjectService struct {
	key, data []byte
	status    pb.ObjectStatus
	err       error
	streamErr error
}

// CreateObject implements pb.ObjectService.CreateObject
func (os *stubObjectService) CreateObject(ctx context.Context, in *pb.CreateObjectRequest, opts ...grpc.CallOption) (*pb.CreateObjectResponse, error) {
	if os.err != nil {
		return nil, os.err
	}
	return &pb.CreateObjectResponse{}, nil
}

// GetObject implements pb.ObjectService.GetObject
func (os *stubObjectService) GetObject(ctx context.Context, in *pb.GetObjectRequest, opts ...grpc.CallOption) (*pb.GetObjectResponse, error) {
	if os.err != nil {
		return nil, os.err
	}
	return &pb.GetObjectResponse{
		Data: os.data,
	}, nil
}

// DeleteObject implements pb.ObjectService.DeleteObject
func (os *stubObjectService) DeleteObject(ctx context.Context, in *pb.DeleteObjectRequest, opts ...grpc.CallOption) (*pb.DeleteObjectResponse, error) {
	if os.err != nil {
		return nil, os.err
	}
	return &pb.DeleteObjectResponse{}, nil
}

// GetObjectStatus implements pb.ObjectService.GetObjectStatus
func (os *stubObjectService) GetObjectStatus(ctx context.Context, in *pb.GetObjectStatusRequest, opts ...grpc.CallOption) (*pb.GetObjectStatusResponse, error) {
	if os.err != nil {
		return nil, os.err
	}
	return &pb.GetObjectStatusResponse{
		Status: os.status,
	}, nil
}

// ListObjectKeys implements pb.ObjectService.ListObjectKeys
func (os *stubObjectService) ListObjectKeys(ctx context.Context, in *pb.ListObjectKeysRequest, opts ...grpc.CallOption) (pb.ObjectManager_ListObjectKeysClient, error) {
	if os.err != nil {
		return nil, os.err
	}
	return &stubListObjectKeysClient{
		ClientStream: nil,
		eof:          os.key != nil,
		key:          os.key,
		err:          os.streamErr,
	}, nil
}

type stubListObjectKeysClient struct {
	grpc.ClientStream
	key []byte
	err error
	eof bool
}

// Recv implements pb.ObjectManager_ListObjectKeysClient.Recv
func (stream *stubListObjectKeysClient) Recv() (*pb.ListObjectKeysResponse, error) {
	if stream.err != nil {
		return nil, stream.err
	}
	if stream.key == nil {
		if stream.eof {
			return nil, io.EOF
		}
		return &pb.ListObjectKeysResponse{}, nil
	}
	resp := &pb.ListObjectKeysResponse{Key: stream.key}
	stream.key = nil
	return resp, nil
}

type stubNamespaceService struct {
	label                          string
	readRPH, writeRPH, nrOfObjects int64
	err                            error
}

// GetNamespace implements pb.NamespaceService.GetNamespace
func (ns *stubNamespaceService) GetNamespace(ctx context.Context, in *pb.GetNamespaceRequest, opts ...grpc.CallOption) (*pb.GetNamespaceResponse, error) {
	if ns.err != nil {
		return nil, ns.err
	}
	return &pb.GetNamespaceResponse{
		Label:               ns.label,
		ReadRequestPerHour:  ns.readRPH,
		WriteRequestPerHour: ns.writeRPH,
		NrObjects:           ns.nrOfObjects,
	}, nil
}
