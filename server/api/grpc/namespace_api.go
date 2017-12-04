package grpc

import (
	"golang.org/x/net/context"

	"github.com/zero-os/0-stor/server/db"
	pb "github.com/zero-os/0-stor/server/schema"
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

// Get implements NamespaceManagerServer.Get
func (api *NamespaceAPI) Get(ctx context.Context, req *pb.GetNamespaceRequest) (*pb.GetNamespaceReply, error) {
	label := req.GetLabel()

	count, err := db.CountKeys(api.db, []byte(label))
	if err != nil {
		return nil, err
	}
	read, write := stats.Rate(label)

	resp := &pb.GetNamespaceReply{
		Namespace: &pb.Namespace{
			Label:               label,
			ReadRequestPerHour:  read,
			WriteRequestPerHour: write,
			// there are no more seperate namespace objects,
			// thus we do no longer need to subtract (for now)
			NrObjects: int64(count), // - 1,
		},
	}

	return resp, nil
}
