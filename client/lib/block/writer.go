package block

import (
	"github.com/zero-os/0-stor/client/meta"
)

// WriteResponse defines response of WriteBlock
type WriteResponse struct {
	Written int
	Err     error
	Meta    *meta.Meta
}

// Writer defines Writer that work on block level
type Writer interface {
	WriteBlock(data []byte) WriteResponse
}
