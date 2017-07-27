package fullreadwrite

import (
	"io"

	"github.com/zero-os/0-stor/client/meta"
)

// WriteResponse defines response of WriteFull
type WriteResponse struct {
	Written int
	Err     error
	Meta    *meta.Meta
}

type Writer interface {
	io.Writer
	WriteFull(data []byte) WriteResponse
}

type Reader interface {
	ReadFull([]byte) ([]byte, error)
}
