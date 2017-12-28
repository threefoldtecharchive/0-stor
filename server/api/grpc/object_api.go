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
	"golang.org/x/sync/errgroup"
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

	// we're dealing with multiple objects,
	// let's handle them asynchronously

	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()

	scopeKey := db.DataScopeKey([]byte(label))
	ch, err := api.db.ListItems(ctx, scopeKey)
	if err != nil {
		log.Errorf("Database error for data (%v): %v", label, err)
		return rpctypes.ErrGRPCDatabase
	}

	outputCh := make(chan pb.ListObjectKeysResponse, api.jobCount)

	// create an errgroup for all the worker routines,
	// including the input one
	group, ctx := errgroup.WithContext(ctx)

	// start the input goroutine,
	// so it can start fetching keys ASAP
	group.Go(func() error {
		// only this goroutine sends to outputCh,
		// so we can simply close it when we're done
		defer close(outputCh)

		// local variables reused for each iteration/item
		var (
			err  error
			key  []byte
			resp pb.ListObjectKeysResponse

			scopeKeyLength = len(scopeKey)
		)
		for item := range ch {
			// copy key to take ownership over it
			key = item.Key()
			if n := len(key); n < scopeKeyLength {
				panic("invalid item key '" + string(key) +
					"' (filtered key is too short)")
			} else if n == scopeKeyLength {
				log.Warningf(
					"skipping listed key result, '%s', as it equals the given scopeKey",
					scopeKey)
				continue
			}
			key = key[scopeKeyLength:]
			resp.Key = make([]byte, len(key))
			copy(resp.Key, key)

			// close current item
			err = item.Close()
			if err != nil {
				log.Errorf("Database error for data (%v): %v", label, err)
				return rpctypes.ErrGRPCDatabase
			}

			// send object over the channel, if possible
			select {
			case outputCh <- resp:
			case <-ctx.Done():
				return nil
			}
		}

		return nil
	})

	// start the output goroutine,
	// as we are only allowed to send to the stream on a single goroutine
	// (sending on multiple goroutines at once is not safe according to docs)
	group.Go(func() error {
		// local variables reused for each iteration/item
		var (
			resp pb.ListObjectKeysResponse
			open bool
		)

		// loop while we can receive responses,
		// or until the context is done
		for {
			select {
			case <-ctx.Done():
				return nil // early exist -> context is done
			case resp, open = <-outputCh:
				if !open {
					return nil // we're done!
				}
			}
			err := stream.Send(&resp)
			if err != nil {
				// TODO: should we check error?
				return err
			}
		}
	})

	// wait until all contexts are finished
	return group.Wait()
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
