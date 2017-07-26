package rpc

import (
	"golang.org/x/net/context"

	pb "github.com/zero-os/0-stor/server/api/rpc/store"
	"github.com/zero-os/0-stor/server/db"
)

var _ (pb.ReservationManagerServer) = (*ReservationManager)(nil)

type ReservationManager struct {
	db db.DB
}

func NewReservationManaer(db db.DB) *ReservationManager {
	return &ReservationManager{db: db}
}

func (mgr *ReservationManager) Create(context.Context, *pb.CreateObjectRequest) (*pb.CreateReservationReply, error) {
	return nil, nil
}
func (mgr *ReservationManager) List(*pb.ListReservationRequest, pb.ReservationManager_ListServer) error {
	return nil
}
func (mgr *ReservationManager) Get(context.Context, *pb.GetReservationRequest) (*pb.GetReservationReply, error) {
	return nil, nil
}
func (mgr *ReservationManager) Renew(context.Context, *pb.RenewReservationRequest) (*pb.RenewReservationReply, error) {
	return nil, nil
}
