package grpc

import (
	"golang.org/x/net/context"

	log "github.com/Sirupsen/logrus"
	"github.com/zero-os/0-stor/server/api/grpc/rpctypes"
	pb "github.com/zero-os/0-stor/server/api/grpc/schema"
	"github.com/zero-os/0-stor/server/db"
	"github.com/zero-os/0-stor/server/stats"
)

var _ (pb.NamespaceManagerServer) = (*NamespaceAPI)(nil)

// NamespaceAPI implements pb.NamespaceManagerServer
type NamespaceAPI struct {
	db db.DB
}

// NewNamespaceAPI returns a NamespaceAPI
func NewNamespaceAPI(db db.DB) *NamespaceAPI {
	if db == nil {
		panic("no database given to NamespaceAPI")
	}

	return &NamespaceAPI{
		db: db,
	}
}

// GetNamespace implements NamespaceManagerServer.GetNamespace
func (api *NamespaceAPI) GetNamespace(ctx context.Context, req *pb.GetNamespaceRequest) (*pb.GetNamespaceResponse, error) {
	label, err := extractStringFromContext(ctx, rpctypes.MetaLabelKey)
	if err != nil {
		return nil, rpctypes.ErrGRPCNilLabel
	}

	count, err := db.CountKeys(api.db, []byte(label))
	if err != nil {
		log.Errorf("Database error for key %v: %v", label, err)
		return nil, rpctypes.ErrGRPCDatabase
	}
	read, write := stats.Rate(label)

	resp := &pb.GetNamespaceResponse{
		Label:               label,
		ReadRequestPerHour:  read,
		WriteRequestPerHour: write,
		NrObjects:           int64(count),
	}

	return resp, nil
}
