// +build go1.9

package client

import (
	pb "github.com/zero-os/0-stor/grpc_store"
)

type CheckStatus = pb.CheckResponse_Status

var (
	CheckStatusOk        = pb.CheckResponse_ok
	CheckStatusCorrupted = pb.CheckResponse_corrupted
	CheckStatusMissing   = pb.CheckResponse_missing
)
