// +build !go1.9

package client

import (
	pb "github.com/zero-os/0-stor/server/schema"
)

type CheckStatus pb.CheckResponse_Status

var (
	CheckStatusOk        = CheckStatus(pb.CheckResponse_ok)
	CheckStatusCorrupted = CheckStatus(pb.CheckResponse_corrupted)
	CheckStatusMissing   = CheckStatus(pb.CheckResponse_missing)
)
