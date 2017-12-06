package grpc

import (
	"errors"

	"golang.org/x/sync/errgroup"

	"golang.org/x/net/context"

	"github.com/zero-os/0-stor/server"
	serverAPI "github.com/zero-os/0-stor/server/api"
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

// Create implements ObjectManagerServer.Create
func (api *ObjectAPI) Create(ctx context.Context, req *pb.CreateObjectRequest) (*pb.CreateObjectReply, error) {
	label := []byte(req.GetLabel())

	obj := req.GetObject()
	key := obj.GetKey()

	// encode the value and store it
	value := obj.GetValue()
	data, err := encoding.EncodeObject(server.Object{Data: value})
	if err != nil {
		return nil, err
	}
	valueKey := db.DataKey(label, key)
	err = api.db.Set(valueKey, data)
	if err != nil {
		return nil, err
	}

	// either delete the reference list, or set it.
	refListkey := db.ReferenceListKey(label, key)
	refList := obj.GetReferenceList()
	if len(refList) == 0 {
		err = api.db.Delete(refListkey)
		if err != nil {
			return nil, err
		}
	} else {
		data, err = encoding.EncodeReferenceList(server.ReferenceList(refList))
		if err != nil {
			return nil, err
		}
		err = api.db.Set(refListkey, data)
		if err != nil {
			return nil, err
		}
	}

	// return the success reply
	return &pb.CreateObjectReply{}, nil
}

// List implements ObjectManagerServer.List
func (api *ObjectAPI) List(req *pb.ListObjectsRequest, stream pb.ObjectManager_ListServer) error {
	label := []byte(req.GetLabel())

	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()

	prefix := db.DataPrefix(label)
	ch, err := api.db.ListItems(ctx, prefix)
	if err != nil {
		return err
	}

	prefixLength := len(prefix)

	type intermediateObject struct {
		Key   []byte
		Value []byte
	}
	workerCh := make(chan intermediateObject, api.jobCount)
	outputCh := make(chan pb.Object, api.jobCount)

	// create an errgroup for all the worker routines,
	// including the input one
	group, ctx := errgroup.WithContext(ctx)

	// start the input goroutine,
	// so it can start fetching keys and values ASAP
	group.Go(func() error {
		// close worker channel when this channel is closed
		// (either because of an error or because all items have been received)
		defer close(workerCh)

		// local variables reused for each iteration/item
		var (
			err      error
			value    []byte
			object   server.Object
			imObject intermediateObject
		)
		for item := range ch {
			// decode the value
			value, err = item.Value()
			if err != nil {
				return err
			}
			object, err = encoding.DecodeObject(value)
			if err != nil {
				return err
			}

			// copy value, to take ownership over it
			imObject.Value = make([]byte, len(object.Data))
			copy(imObject.Value, object.Data)

			// copy key to take ownership over it
			key := item.Key()
			if len(key) <= prefixLength {
				return errors.New("invalid item key")
			}
			key = key[prefixLength+1:]
			imObject.Key = make([]byte, len(key))
			copy(imObject.Key, key)

			// send object over the channel, if possible
			select {
			case workerCh <- imObject:
			case <-ctx.Done():
				return nil
			}

			// close current item
			err = item.Close()
			if err != nil {
				return err
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
			object          pb.Object
			workerStopCount int
		)

		// loop while we can receive intermediate objects,
		// or until the context is done
		for {
			select {
			case <-ctx.Done():
				return nil // early exist -> context is done
			case object = <-outputCh:
				if object.GetKey() == nil {
					workerStopCount++
					if workerStopCount == api.jobCount {
						return nil // we're done!
					}
					continue
				}
			}
			err := stream.Send(&object)
			if err != nil {
				return err
			}
		}
	})

	// start all the workers
	for i := 0; i < api.jobCount; i++ {
		group.Go(func() error {
			// local variables reused for each iteration/item
			var (
				imObject intermediateObject
				object   pb.Object
				open     bool
			)

			// loop while we can receive intermediate objects,
			// or until the context is done
			for {
				select {
				case <-ctx.Done():
					return nil // early exist -> context is done
				case imObject, open = <-workerCh:
					if !open {
						// send a nil object to indicate a worker is finished
						select {
						case outputCh <- pb.Object{}:
						case <-ctx.Done():
							return nil
						}
						return nil // early exit -> worker channel closed
					}
				}

				// set value
				object.Key = imObject.Key
				object.Value = imObject.Value

				// get reference list (if it exists)
				refListKey := db.ReferenceListKey(label, imObject.Key)
				refListData, err := api.db.Get(refListKey)
				if err == db.ErrNotFound {
					object.ReferenceList = nil
				} else {
					if err != nil {
						return err
					}
					object.ReferenceList, err = encoding.DecodeReferenceList(refListData)
					if err != nil {
						return err
					}
				}

				// send the object ready for output
				select {
				case outputCh <- object:
				case <-ctx.Done():
					return nil
				}
			}
		})
	}

	// wait until all contexts are finished
	return group.Wait()
}

// Get implements ObjectManagerServer.Get
func (api *ObjectAPI) Get(ctx context.Context, req *pb.GetObjectRequest) (*pb.GetObjectReply, error) {
	label := []byte(req.GetLabel())
	var object pb.Object
	object.Key = req.GetKey()

	dataKey := db.DataKey(label, object.Key)
	// fetch data
	rawData, err := api.db.Get(dataKey)
	if err != nil {
		return nil, err
	}
	// decode and validate data
	dataObject, err := encoding.DecodeObject(rawData)
	if err != nil {
		return nil, err
	}
	object.Value = dataObject.Data

	// get reference list (if it exists)
	refListKey := db.ReferenceListKey(label, object.Key)
	refListData, err := api.db.Get(refListKey)
	if err == db.ErrNotFound {
		object.ReferenceList = nil
	} else {
		if err != nil {
			return nil, err
		}
		object.ReferenceList, err = encoding.DecodeReferenceList(refListData)
		if err != nil {
			return nil, err
		}
	}

	return &pb.GetObjectReply{
		Object: &object,
	}, nil
}

// Exists implements ObjectManagerServer.Exists
func (api *ObjectAPI) Exists(ctx context.Context, req *pb.ExistsObjectRequest) (*pb.ExistsObjectReply, error) {
	label := []byte(req.GetLabel())
	key := req.GetKey()
	dataKey := db.DataKey(label, key)

	exists, err := api.db.Exists(dataKey)
	if err != nil {
		return nil, err
	}

	return &pb.ExistsObjectReply{
		Exists: exists,
	}, nil
}

// Delete implements ObjectManagerServer.Delete
func (api *ObjectAPI) Delete(ctx context.Context, req *pb.DeleteObjectRequest) (*pb.DeleteObjectReply, error) {
	label := []byte(req.GetLabel())
	key := req.GetKey()
	dataKey := db.DataKey(label, key)

	err := api.db.Delete(dataKey)
	if err != nil {
		return nil, err
	}
	return &pb.DeleteObjectReply{}, nil
}

// SetReferenceList implements ObjectManagerServer.SetReferenceList
func (api *ObjectAPI) SetReferenceList(ctx context.Context, req *pb.UpdateReferenceListRequest) (*pb.UpdateReferenceListReply, error) {
	refList := req.GetReferenceList()
	// encode reference list
	data, err := encoding.EncodeReferenceList(refList)
	if err != nil {
		return nil, err
	}

	// store reference list if possible
	refListKey := db.ReferenceListKey([]byte(req.GetLabel()), req.GetKey())
	err = api.db.Set(refListKey, data)
	if err != nil {
		return nil, err
	}

	return &pb.UpdateReferenceListReply{}, nil
}

// AppendReferenceList implements ObjectManagerServer.AppendReferenceList
func (api *ObjectAPI) AppendReferenceList(ctx context.Context, req *pb.UpdateReferenceListRequest) (*pb.UpdateReferenceListReply, error) {
	// get current reference list data if possible
	refListKey := db.ReferenceListKey([]byte(req.GetLabel()), req.GetKey())

	// define update callback
	cb := func(refListData []byte) ([]byte, error) {
		if refListData == nil {
			// if input of update callback is nil, the data didn't exist yet,
			// in which case we can simply encode the target ref list as it is
			return encoding.EncodeReferenceList(req.GetReferenceList())
		}
		// append new list to current list,
		// without decoding the current list
		return encoding.AppendToEncodedReferenceList(refListData, req.GetReferenceList())
	}

	// loop-update until we have no conflict
	err := api.db.Update(refListKey, cb)
	for err == db.ErrConflict {
		err = api.db.Update(refListKey, cb)
	}
	if err != nil {
		return nil, err
	}

	return &pb.UpdateReferenceListReply{}, nil
}

// RemoveReferenceList implements ObjectManagerServer.RemoveReferenceList
// Removes the items in the request reference list from the Object's reference list
func (api *ObjectAPI) RemoveReferenceList(ctx context.Context, req *pb.UpdateReferenceListRequest) (*pb.UpdateReferenceListReply, error) {
	// get current reference list data if possible
	refListKey := db.ReferenceListKey([]byte(req.GetLabel()), req.GetKey())

	// define update callback
	cb := func(refListData []byte) ([]byte, error) {
		if refListData == nil {
			// if input of update callback is nil, the data didn't exist yet,
			// in which case we can simply return nil, as we don't need to do anything
			return nil, nil
		}
		// remove new list from current list
		return encoding.RemoveFromEncodedReferenceList(refListData, req.GetReferenceList())
	}

	// loop-update until we have no conflict
	err := api.db.Update(refListKey, cb)
	for err == db.ErrConflict {
		err = api.db.Update(refListKey, cb)
	}
	if err != nil {
		return nil, err
	}

	return &pb.UpdateReferenceListReply{}, nil
}

// Check implements ObjectManagerServer.Check
func (api *ObjectAPI) Check(req *pb.CheckRequest, stream pb.ObjectManager_CheckServer) error {
	label := []byte(req.GetLabel())

	// if no ids are given, return early
	Ids := req.GetIds()
	length := len(Ids)
	if length == 0 {
		return nil
	}
	// if only one id is given, simply check that object
	if length == 1 {
		// check status
		status, err := serverAPI.CheckStatusForObject(label, []byte(Ids[0]), api.db)
		if err != nil {
			return err
		}
		protoStatus := convertStatus(status)
		// send the status for that single object given to the callee
		return stream.Send(&pb.CheckResponse{Id: Ids[0], Status: protoStatus})
	}

	// multiple ids are requested, let's start goroutines
	jobCount := api.jobCount
	if length < jobCount {
		jobCount = length
	}

	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()

	idCh := make(chan string, jobCount)
	outCh := make(chan pb.CheckResponse, jobCount)

	// create an errgroup for all the worker routines,
	// including the input one
	group, ctx := errgroup.WithContext(ctx)

	// start the input goroutine,
	// so it can start fetching keys and values ASAP
	group.Go(func() error {
		// close worker channel when this channel is closed
		// (either because of an error or because all items have been received)
		defer close(idCh)
		for i := range Ids {
			select {
			case idCh <- Ids[i]:
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
			resp            pb.CheckResponse
			workerStopCount int
		)

		// loop while we can receive intermediate objects,
		// or until the context is done
		for {
			select {
			case <-ctx.Done():
				return nil // early exist -> context is done
			case resp = <-outCh:
				if resp.GetId() == "" {
					workerStopCount++
					if workerStopCount == jobCount {
						return nil // we're done!
					}
					continue
				}
			}
			err := stream.Send(&resp)
			if err != nil {
				return err
			}
		}
	})

	// start all the workers
	for i := 0; i < jobCount; i++ {
		group.Go(func() error {
			// local variables reused for each iteration/item
			var (
				response pb.CheckResponse
				open     bool
			)

			// loop while we can receive intermediate objects,
			// or until the context is done
			for {
				select {
				case <-ctx.Done():
					return nil // early exist -> context is done
				case response.Id, open = <-idCh:
					if !open {
						// send a nil response to indicate a worker is finished
						select {
						case outCh <- pb.CheckResponse{}:
						case <-ctx.Done():
							return nil
						}
						return nil // early exit -> worker channel closed
					}
				}

				// check status
				status, err := serverAPI.CheckStatusForObject(label, []byte(response.Id), api.db)
				if err != nil {
					return err
				}
				response.Status = convertStatus(status)

				// send the object ready for output
				select {
				case outCh <- response:
				case <-ctx.Done():
					return nil
				}
			}
		})
	}

	// wait until all contexts are finished
	return group.Wait()
}

// convertStatus convert manager.CheckStatus to pb.CheckResponse_Status
func convertStatus(status server.CheckStatus) pb.CheckResponse_Status {
	s, ok := _ProtoCheckStatusMapping[status]
	if !ok {
		panic("unknown CheckStatus")
	}
	return s
}

var _ProtoCheckStatusMapping = map[server.CheckStatus]pb.CheckResponse_Status{
	server.CheckStatusOK:        pb.CheckStatusOK,
	server.CheckStatusMissing:   pb.CheckStatusMissing,
	server.CheckStatusCorrupted: pb.CheckStatusCorrupted,
}
