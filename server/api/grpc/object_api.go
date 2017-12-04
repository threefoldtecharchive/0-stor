package grpc

import (
	"fmt"

	"golang.org/x/net/context"

	"github.com/zero-os/0-stor/server/db"
	"github.com/zero-os/0-stor/server/manager"
	pb "github.com/zero-os/0-stor/server/schema"
)

var _ (pb.ObjectManagerServer) = (*ObjectAPI)(nil)

// ObjectAPI implements pb.ObjectManagerServer
type ObjectAPI struct {
	db db.DB
}

// NewObjectAPI returns a new ObjectAPI
func NewObjectAPI(db db.DB) *ObjectAPI {
	if db == nil {
		panic("no database given to ObjectAPI")
	}

	return &ObjectAPI{
		db: db,
	}
}

// Create implements ObjectManagerServer.Create
func (api *ObjectAPI) Create(ctx context.Context, req *pb.CreateObjectRequest) (*pb.CreateObjectReply, error) {
	label := req.GetLabel()

	obj := req.GetObject()

	mgr := manager.NewObjectManager(label, api.db)

	if err := mgr.Set([]byte(obj.GetKey()), obj.GetValue(), obj.GetReferenceList()); err != nil {
		return nil, err
	}

	return &pb.CreateObjectReply{}, nil
}

// List implements ObjectManagerServer.List
func (api *ObjectAPI) List(req *pb.ListObjectsRequest, stream pb.ObjectManager_ListServer) error {

	label := req.GetLabel()

	mgr := manager.NewObjectManager(label, api.db)

	keys, err := mgr.List(0, -1)
	if err != nil {
		return err
	}

	for _, key := range keys {
		obj, err := mgr.Get(key)
		if err != nil {
			return err
		}

		o, err := grpcObj(obj)
		if err != nil {
			return err
		}

		if err := stream.Send(o); err != nil {
			return nil
		}
	}

	return nil
}

// Get implements ObjectManagerServer.Get
func (api *ObjectAPI) Get(ctx context.Context, req *pb.GetObjectRequest) (*pb.GetObjectReply, error) {
	label, key := req.GetLabel(), req.GetKey()

	mgr := manager.NewObjectManager(label, api.db)

	obj, err := mgr.Get([]byte(key))
	if err != nil {
		return nil, err
	}

	o, err := grpcObj(obj)
	if err != nil {
		return nil, err
	}

	return &pb.GetObjectReply{
		Object: o,
	}, nil
}

// Exists implements ObjectManagerServer.Exists
func (api *ObjectAPI) Exists(ctx context.Context, req *pb.ExistsObjectRequest) (*pb.ExistsObjectReply, error) {
	label, key := req.GetLabel(), req.GetKey()

	mgr := manager.NewObjectManager(label, api.db)

	exists, err := mgr.Exists([]byte(key))
	if err != nil {
		return nil, err
	}

	return &pb.ExistsObjectReply{
		Exists: exists,
	}, nil
}

// Delete implements ObjectManagerServer.Delete
func (api *ObjectAPI) Delete(ctx context.Context, req *pb.DeleteObjectRequest) (*pb.DeleteObjectReply, error) {
	label, key := req.GetLabel(), req.GetKey()

	mgr := manager.NewObjectManager(label, api.db)

	if err := mgr.Delete([]byte(key)); err != nil {
		return nil, err
	}

	return &pb.DeleteObjectReply{}, nil
}

// SetReferenceList implements ObjectManagerServer.SetReferenceList
func (api *ObjectAPI) SetReferenceList(ctx context.Context, req *pb.UpdateReferenceListRequest) (*pb.UpdateReferenceListReply, error) {
	label, key, refList := req.GetLabel(), req.GetKey(), req.GetReferenceList()

	if len(refList) > db.RefIDCount {
		return nil, fmt.Errorf("too big reference list = %v", len(refList))
	}

	mgr := manager.NewObjectManager(label, api.db)
	err := mgr.UpdateReferenceList(key, refList, manager.RefListOpSet)

	return &pb.UpdateReferenceListReply{}, err
}

// AppendReferenceList implements ObjectManagerServer.AppendReferenceList
func (api *ObjectAPI) AppendReferenceList(ctx context.Context, req *pb.UpdateReferenceListRequest) (*pb.UpdateReferenceListReply, error) {
	label, key, refList := req.GetLabel(), req.GetKey(), req.GetReferenceList()

	if len(refList) > db.RefIDCount {
		return nil, fmt.Errorf("too big reference list = %v", len(refList))
	}

	mgr := manager.NewObjectManager(label, api.db)
	err := mgr.UpdateReferenceList(key, refList, manager.RefListOpAppend)

	return &pb.UpdateReferenceListReply{}, err
}

// RemoveReferenceList implements ObjectManagerServer.RemoveReferenceList
// Removes the items in the request reference list from the Object's reference list
func (api *ObjectAPI) RemoveReferenceList(ctx context.Context, req *pb.UpdateReferenceListRequest) (*pb.UpdateReferenceListReply, error) {
	label, key, refList := req.GetLabel(), req.GetKey(), req.GetReferenceList()

	if len(refList) > db.RefIDCount {
		return nil, fmt.Errorf("too big reference list = %v", len(refList))
	}

	mgr := manager.NewObjectManager(label, api.db)
	err := mgr.UpdateReferenceList(key, refList, manager.RefListOpRemove)

	return &pb.UpdateReferenceListReply{}, err
}

// Check implements ObjectManagerServer.Check
func (api *ObjectAPI) Check(req *pb.CheckRequest, stream pb.ObjectManager_CheckServer) error {
	label, ids := req.GetLabel(), req.GetIds()

	mgr := manager.NewObjectManager(label, api.db)

	for _, id := range ids {
		status, err := mgr.Check([]byte(id))
		if err != nil {
			return err
		}

		if err := stream.Send(&pb.CheckResponse{
			Id:     id,
			Status: convertStatus(status),
		}); err != nil {
			return nil
		}
	}

	return nil
}

// convertStatus convert manager.CheckStatus to pb.CheckResponse_Status
func convertStatus(status manager.CheckStatus) pb.CheckResponse_Status {
	switch status {
	case manager.CheckStatusOK:
		return pb.CheckResponse_ok
	case manager.CheckStatusMissing:
		return pb.CheckResponse_missing
	case manager.CheckStatusCorrupted:
		return pb.CheckResponse_corrupted
	default:
		panic("unknown CheckStatus")
	}
}

// grpcObj convert a db.Object to a pb.Object
func grpcObj(obj *db.Object) (*pb.Object, error) {
	data, err := obj.Data()
	if err != nil {
		return nil, err
	}
	refList, err := obj.GetreferenceListStr()
	if err != nil {
		return nil, err
	}
	return &pb.Object{
		Key:           obj.Key,
		Value:         data,
		ReferenceList: refList,
	}, nil
}
