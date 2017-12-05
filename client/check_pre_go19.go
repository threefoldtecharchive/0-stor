// +build !go1.9

package client

import (
	pb "github.com/zero-os/0-stor/server/api/grpc/schema"
)

type CheckStatus pb.CheckResponse_Status

var (
	CheckStatusOk        = CheckStatus(pb.CheckStatusOK)
	CheckStatusCorrupted = CheckStatus(pb.CheckStatusCorrupted)
	CheckStatusMissing   = CheckStatus(pb.CheckStatusMissing)
)
