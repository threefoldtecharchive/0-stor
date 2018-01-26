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
	"github.com/zero-os/0-stor/server"
	serverAPI "github.com/zero-os/0-stor/server/api"
	"github.com/zero-os/0-stor/server/api/grpc/rpctypes"
	pb "github.com/zero-os/0-stor/server/api/grpc/schema"
	"github.com/zero-os/0-stor/server/db"
	"github.com/zero-os/0-stor/server/encoding"

	log "github.com/Sirupsen/logrus"
	"golang.org/x/net/context"
)

var _ (pb.ObjectManagerServer) = (*ObjectAPI)(nil)

// ObjectAPI implements pb.ObjectManagerServer
type ObjectAPI struct {
	db       db.DB
	jobCount int
}

// NewObjectAPI returns a new ObjectAPI
func NewObjectAPI(db db.DB, jobs int) *ObjectAPI {
	if db == nil {
		panic("no database given to ObjectAPI")
	}
	if jobs <= 0 {
		jobs = DefaultJobCount
	}

	return &ObjectAPI{
		db:       db,
		jobCount: jobs,
	}
}

// CreateObject implements ObjectManagerServer.CreateObject
func (api *ObjectAPI) CreateObject(ctx context.Context, req *pb.CreateObjectRequest) (*pb.CreateObjectResponse, error) {
	label, err := extractStringFromContext(ctx, rpctypes.MetaLabelKey)
	if err != nil {
		log.Errorf("error while extracting label from GRPC metadata: %v", err)
		return nil, rpctypes.ErrGRPCNilLabel
	}

	// encode the data and store it
	data := req.GetData()
	if len(data) == 0 {
		return nil, rpctypes.ErrGRPCNilData
	}
	encodedData, err := encoding.EncodeObject(server.Object{Data: data})
	if err != nil {
		panic(err)
	}
	scopeKey := db.DataScopeKey([]byte(label))
	key, err := api.db.SetScoped(scopeKey, encodedData)
	if err != nil {
		return nil, rpctypes.ErrGRPCDatabase
	}

	// return the success reply
	return &pb.CreateObjectResponse{
		Key: key[len(scopeKey):],
	}, nil
}

// GetObject implements ObjectManagerServer.GetObject
func (api *ObjectAPI) GetObject(ctx context.Context, req *pb.GetObjectRequest) (*pb.GetObjectResponse, error) {
	label, err := extractStringFromContext(ctx, rpctypes.MetaLabelKey)
	if err != nil {
		log.Errorf("error while extracting label from GRPC metadata: %v", err)
		return nil, rpctypes.ErrGRPCNilLabel
	}

	// get key and ensure it's given
	key := req.GetKey()
	if len(key) == 0 {
		return nil, rpctypes.ErrGRPCNilKey
	}
	key = db.DataKey([]byte(label), key)

	// get data
	rawData, err := api.db.Get(key)
	if err != nil {
		if err == db.ErrNotFound {
			return nil, rpctypes.ErrGRPCKeyNotFound
		}
		log.Errorf("Database error for data (%v): %v", key, err)
		return nil, rpctypes.ErrGRPCDatabase
	}
	dataObject, err := encoding.DecodeObject(rawData)
	if err != nil {
		return nil, rpctypes.ErrGRPCObjectDataCorrupted
	}

	// return referenced object
	return &pb.GetObjectResponse{
		Data: dataObject.Data,
	}, nil
}

// DeleteObject implements ObjectManagerServer.DeleteObject
func (api *ObjectAPI) DeleteObject(ctx context.Context, req *pb.DeleteObjectRequest) (*pb.DeleteObjectResponse, error) {
	label, err := extractStringFromContext(ctx, rpctypes.MetaLabelKey)
	if err != nil {
		log.Errorf("error while extracting label from GRPC metadata: %v", err)
		return nil, rpctypes.ErrGRPCNilLabel
	}

	// get key and ensure it's given
	key := req.GetKey()
	if len(key) == 0 {
		return nil, rpctypes.ErrGRPCNilKey
	}
	key = db.DataKey([]byte(label), key)

	// delete object's data
	err = api.db.Delete(key)
	if err != nil {
		log.Errorf("Database error for data (%v): %v", key, err)
		return nil, rpctypes.ErrGRPCDatabase
	}

	// success, object is deleted
	return &pb.DeleteObjectResponse{}, nil
}

// GetObjectStatus implements ObjectManagerServer.GetObjectStatus
func (api *ObjectAPI) GetObjectStatus(ctx context.Context, req *pb.GetObjectStatusRequest) (*pb.GetObjectStatusResponse, error) {
	label, err := extractStringFromContext(ctx, rpctypes.MetaLabelKey)
	if err != nil {
		log.Errorf("error while extracting label from GRPC metadata: %v", err)
		return nil, rpctypes.ErrGRPCNilLabel
	}

	// get key and ensure it's given
	key := req.GetKey()
	if len(key) == 0 {
		return nil, rpctypes.ErrGRPCNilKey
	}
	key = db.DataKey([]byte(label), key)

	status, err := serverAPI.ObjectStatusForObject(key, api.db)
	if err != nil {
		log.Errorf("Database error for data (%v): %v", key, err)
		return nil, rpctypes.ErrGRPCDatabase
	}
	return &pb.GetObjectStatusResponse{Status: convertStatus(status)}, nil

}

// ListObjectKeys implements ObjectManagerServer.ListObjectKeys
func (api *ObjectAPI) ListObjectKeys(req *pb.ListObjectKeysRequest, stream pb.ObjectManager_ListObjectKeysServer) error {
	label, err := extractStringFromContext(stream.Context(), rpctypes.MetaLabelKey)
	if err != nil {
		log.Errorf("error while extracting label from GRPC metadata: %v", err)
		return rpctypes.ErrGRPCNilLabel
	}

	var (
		key            []byte
		scopeKey       = db.DataScopeKey([]byte(label))
		scopeKeyLength = len(scopeKey)
	)
	return api.db.ListItems(func(item db.Item) error {
		// copy key to take ownership over it
		key, err = item.Key()
		if n := len(key); n < scopeKeyLength {
			panic("invalid item key '" + string(key) +
				"' (filtered key is too short)")
		} else if n == scopeKeyLength {
			log.Warningf(
				"skipping listed key result, '%s', as it equals the given scopeKey",
				scopeKey)
			return nil
		}

		// send key over stream
		return stream.Send(&pb.ListObjectKeysResponse{
			Key: key[scopeKeyLength:],
		})
	}, scopeKey)
}

// convertStatus converts server.ObjectStatus to pb.ObjectStatus
func convertStatus(status server.ObjectStatus) pb.ObjectStatus {
	s, ok := _ProtoObjectStatusMapping[status]
	if !ok {
		panic("unknown ObjectStatus")
	}
	return s
}

var _ProtoObjectStatusMapping = map[server.ObjectStatus]pb.ObjectStatus{
	server.ObjectStatusOK:        pb.ObjectStatusOK,
	server.ObjectStatusMissing:   pb.ObjectStatusMissing,
	server.ObjectStatusCorrupted: pb.ObjectStatusCorrupted,
}
