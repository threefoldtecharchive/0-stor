package grpc

import (
	"golang.org/x/net/context"

	pb "github.com/zero-os/0-stor/server/api/grpc/store"
	"github.com/zero-os/0-stor/server/db"
)

var _ (pb.ObjectManagerServer) = (*ObjectManager)(nil)

type ObjectManager struct {
	db db.DB
}

func NewObjectManager(db db.DB) *ObjectManager {
	return &ObjectManager{db: db}
}

func (mgr *ObjectManager) Create(context.Context, *pb.CreateObjectRequest) (*pb.CreateObjectReply, error) {
	return nil, nil
}
func (mgr *ObjectManager) List(*pb.ListObjectsRequest, pb.ObjectManager_ListServer) error {
	return nil
}
func (mgr *ObjectManager) Get(context.Context, *pb.GetObjectRequest) (*pb.GetObjectReply, error) {
	return nil, nil
}
func (mgr *ObjectManager) Exists(context.Context, *pb.ExistsObjectRequest) (*pb.ExistsObjectReply, error) {
	return nil, nil
}
func (mgr *ObjectManager) Delete(context.Context, *pb.DeleteObjectRequest) (*pb.DeleteObjectReply, error) {
	return nil, nil
}
