package fullreadwrite

import (
	"io"

	"github.com/zero-os/0-stor-lib/meta"
)

type WriteResponse struct {
	Written int
	Err     error
	Meta    *meta.Meta
}

type FullWriter interface {
	io.Writer
	WriteFull(data []byte) WriteResponse
}
