package grpc

import (
	"golang.org/x/sync/errgroup"

	"golang.org/x/net/context"

	log "github.com/Sirupsen/logrus"
	"github.com/zero-os/0-stor/server"
	serverAPI "github.com/zero-os/0-stor/server/api"
	"github.com/zero-os/0-stor/server/api/grpc/rpctypes"
	pb "github.com/zero-os/0-stor/server/api/grpc/schema"
	"github.com/zero-os/0-stor/server/db"
	"github.com/zero-os/0-stor/server/encoding"
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

// SetObject implements ObjectManagerServer.SetObject
func (api *ObjectAPI) SetObject(ctx context.Context, req *pb.SetObjectRequest) (*pb.SetObjectResponse, error) {
	label, err := extractStringFromContext(ctx, rpctypes.MetaLabelKey)
	if err != nil {
		log.Errorf("error while extracting label from GRPC metadata: %v", err)
		return nil, rpctypes.ErrGRPCNilLabel
	}

	key := req.GetKey()
	if len(key) == 0 {
		return nil, rpctypes.ErrGRPCNilKey
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
	valueKey := db.DataKey([]byte(label), key)
	err = api.db.Set(valueKey, encodedData)
	if err != nil {
		return nil, rpctypes.ErrGRPCDatabase
	}

	// either delete the reference list, or set it.
	refList := req.GetReferenceList()
	if refList != nil {
		data, err = encoding.EncodeReferenceList(server.ReferenceList(refList))
		if err != nil {
			panic(err)
		}

		refListkey := db.ReferenceListKey([]byte(label), key)
		err = api.db.Set(refListkey, data)
		if err != nil {
			return nil, rpctypes.ErrGRPCDatabase
		}
	}

	// return the success reply
	return &pb.SetObjectResponse{}, nil
}

// GetObject implements ObjectManagerServer.GetObject
func (api *ObjectAPI) GetObject(ctx context.Context, req *pb.GetObjectRequest) (*pb.GetObjectResponse, error) {
	label, err := extractStringFromContext(ctx, rpctypes.MetaLabelKey)
	if err != nil {
		log.Errorf("error while extracting label from GRPC metadata: %v", err)
		return nil, rpctypes.ErrGRPCNilLabel
	}

	key := req.GetKey()
	if len(key) == 0 {
		return nil, rpctypes.ErrGRPCNilKey
	}

	// get data
	dataKey := db.DataKey([]byte(label), key)
	rawData, err := api.db.Get(dataKey)
	if err != nil {
		if err == db.ErrNotFound {
			return nil, rpctypes.ErrGRPCKeyNotFound
		}
		log.Errorf("Database error for data (%v): %v", dataKey, err)
		return nil, rpctypes.ErrGRPCDatabase
	}
	dataObject, err := encoding.DecodeObject(rawData)
	if err != nil {
		return nil, rpctypes.ErrGRPCObjectDataCorrupted
	}

	// get reference list (if it exists)
	refListKey := db.ReferenceListKey([]byte(label), key)
	refListData, err := api.db.Get(refListKey)
	if err != nil {
		if err == db.ErrNotFound {
			// return non referenced object
			return &pb.GetObjectResponse{
				Data: dataObject.Data,
			}, nil
		}
		log.Errorf("Database error for refList (%v): %v", refListKey, err)
		return nil, rpctypes.ErrGRPCDatabase
	}

	// decode existing reference list
	refList, err := encoding.DecodeReferenceList(refListData)
	if err != nil {
		return nil, rpctypes.ErrGRPCObjectRefListCorrupted
	}

	// return referenced object
	return &pb.GetObjectResponse{
		Data:          dataObject.Data,
		ReferenceList: refList,
	}, nil
}

// DeleteObject implements ObjectManagerServer.DeleteObject
func (api *ObjectAPI) DeleteObject(ctx context.Context, req *pb.DeleteObjectRequest) (*pb.DeleteObjectResponse, error) {
	label, err := extractStringFromContext(ctx, rpctypes.MetaLabelKey)
	if err != nil {
		log.Errorf("error while extracting label from GRPC metadata: %v", err)
		return nil, rpctypes.ErrGRPCNilLabel
	}

	key := req.GetKey()
	if len(key) == 0 {
		return nil, rpctypes.ErrGRPCNilKey
	}

	// delete object's data
	dataKey := db.DataKey([]byte(label), key)
	err = api.db.Delete(dataKey)
	if err != nil {
		log.Errorf("Database error for data (%v): %v", dataKey, err)
		return nil, rpctypes.ErrGRPCDatabase
	}

	// delete object's reference list
	refListKey := db.ReferenceListKey([]byte(label), key)
	err = api.db.Delete(refListKey)
	if err != nil {
		log.Errorf("Database error for refList (%v): %v", refListKey, err)
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

	key := req.GetKey()
	if len(key) == 0 {
		return nil, rpctypes.ErrGRPCNilKey
	}

	status, err := serverAPI.ObjectStatusForObject([]byte(label), key, api.db)
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

	ch, err := api.db.ListItems(ctx, db.DataPrefix([]byte(label)))
	if err != nil {
		log.Errorf("Database error for data (%v): %v", label, err)
		return rpctypes.ErrGRPCDatabase
	}

	prefixLength := db.DataKeyPrefixLength([]byte(label))
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
		)
		for item := range ch {
			// copy key to take ownership over it
			key = item.Key()
			if len(key) <= prefixLength {
				panic("invalid item key (filtered key is too short)")
			}
			key = key[prefixLength:]
			resp.Key = make([]byte, len(key))
			copy(resp.Key, key)

			// send object over the channel, if possible
			select {
			case outputCh <- resp:
			case <-ctx.Done():
				return nil
			}

			// close current item
			err = item.Close()
			if err != nil {
				log.Errorf("Database error for data (%v): %v", label, err)
				return rpctypes.ErrGRPCDatabase
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

// SetReferenceList implements ObjectManagerServer.SetReferenceList
func (api *ObjectAPI) SetReferenceList(ctx context.Context, req *pb.SetReferenceListRequest) (*pb.SetReferenceListResponse, error) {
	label, err := extractStringFromContext(ctx, rpctypes.MetaLabelKey)
	if err != nil {
		log.Errorf("error while extracting label from GRPC metadata: %v", err)
		return nil, rpctypes.ErrGRPCNilLabel
	}

	// get parameters and ensure they're given
	key := req.GetKey()
	if len(key) == 0 {
		return nil, rpctypes.ErrGRPCNilKey
	}
	refList := req.GetReferenceList()
	if len(refList) == 0 {
		return nil, rpctypes.ErrGRPCNilRefList
	}

	// encode reference list
	data, err := encoding.EncodeReferenceList(refList)
	if err != nil {
		panic(err)
	}

	// store reference list if possible
	refListKey := db.ReferenceListKey([]byte(label), key)
	err = api.db.Set(refListKey, data)
	if err != nil {
		return nil, rpctypes.ErrGRPCDatabase
	}

	return &pb.SetReferenceListResponse{}, nil
}

// GetReferenceList implements ObjectManagerServer.GetReferenceList
func (api *ObjectAPI) GetReferenceList(ctx context.Context, req *pb.GetReferenceListRequest) (*pb.GetReferenceListResponse, error) {
	label, err := extractStringFromContext(ctx, rpctypes.MetaLabelKey)
	if err != nil {
		log.Errorf("error while extracting label from GRPC metadata: %v", err)
		return nil, rpctypes.ErrGRPCNilLabel
	}

	key := req.GetKey()
	if len(key) == 0 {
		return nil, rpctypes.ErrGRPCNilKey
	}

	refListKey := db.ReferenceListKey([]byte(label), key)
	refListData, err := api.db.Get(refListKey)
	if err != nil {
		if err == db.ErrNotFound {
			return nil, rpctypes.ErrGRPCKeyNotFound
		}

		log.Errorf("Database error for refList (%v): %v", refListKey, err)
		return nil, rpctypes.ErrGRPCDatabase
	}

	refList, err := encoding.DecodeReferenceList(refListData)
	if err != nil {
		return nil, rpctypes.ErrGRPCObjectRefListCorrupted
	}

	return &pb.GetReferenceListResponse{
		ReferenceList: refList,
	}, nil
}

// GetReferenceCount implements ObjectManagerServer.GetReferenceCount
func (api *ObjectAPI) GetReferenceCount(ctx context.Context, req *pb.GetReferenceCountRequest) (*pb.GetReferenceCountResponse, error) {
	label, err := extractStringFromContext(ctx, rpctypes.MetaLabelKey)
	if err != nil {
		log.Errorf("error while extracting label from GRPC metadata: %v", err)
		return nil, rpctypes.ErrGRPCNilLabel
	}

	key := req.GetKey()
	if len(key) == 0 {
		return nil, rpctypes.ErrGRPCNilKey
	}

	refListKey := db.ReferenceListKey([]byte(label), key)
	refListData, err := api.db.Get(refListKey)
	if err != nil {
		if err == db.ErrNotFound {
			// no reference list == no references
			return &pb.GetReferenceCountResponse{
				Count: 0,
			}, nil
		}

		log.Errorf("Database error for refList (%v): %v", refListKey, err)
		return nil, rpctypes.ErrGRPCDatabase
	}

	refList, err := encoding.DecodeReferenceList(refListData)
	if err != nil {
		return nil, rpctypes.ErrGRPCObjectRefListCorrupted
	}

	return &pb.GetReferenceCountResponse{
		Count: int64(len(refList)),
	}, nil
}

// AppendToReferenceList implements ObjectManagerServer.AppendToReferenceList
func (api *ObjectAPI) AppendToReferenceList(ctx context.Context, req *pb.AppendToReferenceListRequest) (*pb.AppendToReferenceListResponse, error) {
	label, err := extractStringFromContext(ctx, rpctypes.MetaLabelKey)
	if err != nil {
		log.Errorf("error while extracting label from GRPC metadata: %v", err)
		return nil, rpctypes.ErrGRPCNilLabel
	}

	// get parameters and ensure they're given
	key := req.GetKey()
	if len(key) == 0 {
		return nil, rpctypes.ErrGRPCNilKey
	}
	refList := req.GetReferenceList()
	if len(refList) == 0 {
		return nil, rpctypes.ErrGRPCNilRefList
	}

	// define update callback
	cb := func(refListData []byte) ([]byte, error) {
		if len(refListData) == 0 {
			// if input of update callback is nil, the data didn't exist yet,
			// in which case we can simply encode the target ref list as it is
			return encoding.EncodeReferenceList(refList)
		}
		// append new list to current list,
		// without decoding the current list
		return encoding.AppendToEncodedReferenceList(refListData, refList)
	}

	refListKey := db.ReferenceListKey([]byte(label), key)
	// loop-update until we have no conflict
	err = api.db.Update(refListKey, cb)
	for err == db.ErrConflict {
		err = api.db.Update(refListKey, cb)
	}
	if err != nil {
		if err == encoding.ErrInvalidChecksum || err == encoding.ErrInvalidData {
			return nil, rpctypes.ErrGRPCObjectRefListCorrupted
		}
		log.Errorf("Database error for refList (%v): %v", refListKey, err)
		return nil, rpctypes.ErrGRPCDatabase
	}

	return &pb.AppendToReferenceListResponse{}, nil
}

// DeleteFromReferenceList implements ObjectManagerServer.DeleteFromReferenceList
func (api *ObjectAPI) DeleteFromReferenceList(ctx context.Context, req *pb.DeleteFromReferenceListRequest) (*pb.DeleteFromReferenceListResponse, error) {
	label, err := extractStringFromContext(ctx, rpctypes.MetaLabelKey)
	if err != nil {
		log.Errorf("error while extracting label from GRPC metadata: %v", err)
		return nil, rpctypes.ErrGRPCNilLabel
	}

	// get parameters and ensure they're given
	key := req.GetKey()
	if len(key) == 0 {
		return nil, rpctypes.ErrGRPCNilKey
	}
	refList := req.GetReferenceList()
	if len(refList) == 0 {
		return nil, rpctypes.ErrGRPCNilRefList
	}

	var count int
	// define update callback
	cb := func(refListData []byte) ([]byte, error) {
		if len(refListData) == 0 {
			// if input of update callback is nil, the data didn't exist yet,
			// in which case we can simply return nil, as we don't need to do anything
			return nil, nil
		}
		var data []byte
		// remove new list from current list
		data, count, err = encoding.RemoveFromEncodedReferenceList(refListData, refList)
		return data, err
	}

	// get current reference list data if possible
	refListKey := db.ReferenceListKey([]byte(label), key)
	// loop-update until we have no conflict
	err = api.db.Update(refListKey, cb)
	for err == db.ErrConflict {
		err = api.db.Update(refListKey, cb)
	}
	if err != nil {
		if err == encoding.ErrInvalidChecksum || err == encoding.ErrInvalidData {
			return nil, rpctypes.ErrGRPCObjectRefListCorrupted
		}
		log.Errorf("Database error for refList (%v): %v", refListKey, err)
		return nil, rpctypes.ErrGRPCDatabase
	}

	return &pb.DeleteFromReferenceListResponse{
		Count: int64(count),
	}, nil
}

// DeleteReferenceList implements ObjectManagerServer.DeleteReferenceList
func (api *ObjectAPI) DeleteReferenceList(ctx context.Context, req *pb.DeleteReferenceListRequest) (*pb.DeleteReferenceListResponse, error) {
	label, err := extractStringFromContext(ctx, rpctypes.MetaLabelKey)
	if err != nil {
		log.Errorf("error while extracting label from GRPC metadata: %v", err)
		return nil, rpctypes.ErrGRPCNilLabel
	}

	// get key parameter and ensure it's given
	key := req.GetKey()
	if len(key) == 0 {
		return nil, rpctypes.ErrGRPCNilKey
	}

	// delete ref list
	refListKey := db.ReferenceListKey([]byte(label), key)
	err = api.db.Delete(refListKey)
	if err != nil {
		log.Errorf("Database error for refList (%v): %v", refListKey, err)
		return nil, rpctypes.ErrGRPCDatabase
	}

	// success, reference list is deleted
	return &pb.DeleteReferenceListResponse{}, nil
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
