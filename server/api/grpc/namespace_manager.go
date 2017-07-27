package grpc

import (
	"golang.org/x/net/context"

	pb "github.com/zero-os/0-stor/server/api/grpc/store"
	"github.com/zero-os/0-stor/server/db"
)

var _ (pb.NamespaceManagerServer) = (*NamespaceManager)(nil)

type NamespaceManager struct {
	db db.DB
}

func NewNamespaceManager(db db.DB) *NamespaceManager {
	return &NamespaceManager{db: db}
}

func (mgr *NamespaceManager) List(*pb.ListReservationRequest, pb.NamespaceManager_ListServer) error {
	return nil
}
func (mgr *NamespaceManager) Create(context.Context, *pb.CreateReservationRequest) (*pb.CreateNamespaceReply, error) {
	return nil, nil
}
func (mgr *NamespaceManager) Get(context.Context, *pb.GetNamespaceStatRequest) (*pb.GetNamespaceReply, error) {
	return nil, nil
}
func (mgr *NamespaceManager) Delete(context.Context, *pb.DeleteNamespaceRequest) (*pb.DeleteNamespaceReply, error) {
	return nil, nil
}
func (mgr *NamespaceManager) Stat(context.Context, *pb.GetNamespaceStatRequest) (*pb.GetNamespaceStatReply, error) {
	return nil, nil
}
