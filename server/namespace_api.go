package server

import (
	"golang.org/x/net/context"

	"github.com/zero-os/0-stor/server/db"
	"github.com/zero-os/0-stor/server/jwt"
	"github.com/zero-os/0-stor/server/manager"
	pb "github.com/zero-os/0-stor/server/schema"
	"github.com/zero-os/0-stor/server/stats"
)

var _ (pb.NamespaceManagerServer) = (*NamespaceAPI)(nil)

type NamespaceAPI struct {
	db          db.DB
	jwtVerifier jwt.TokenVerifier
}

func NewNamespaceAPI(db db.DB) *NamespaceAPI {
	if db == nil {
		panic("no database given to NamespaceAPI")
	}

	return &NamespaceAPI{
		db: db,
	}
}

func (api *NamespaceAPI) Get(ctx context.Context, req *pb.GetNamespaceRequest) (*pb.GetNamespaceReply, error) {
	label := req.GetLabel()

	mgr := manager.NewNamespaceManager(api.db)

	count, err := mgr.Count(label)
	if err != nil {
		return nil, err
	}
	read, write := stats.Rate(label)

	resp := &pb.GetNamespaceReply{
		Namespace: &pb.Namespace{
			Label: label,
			// SpaceAvailale: ,
			// SpaceUsed: ,
			ReadRequestPerHour:  read,
			WriteRequestPerHour: write,
			NrObjects:           int64(count) - 1,
		},
	}

	return resp, nil
}
