package fullreadwrite

import (
	"bytes"
)

type BytesBuffer struct {
	*bytes.Buffer
}

func NewBytesBuffer() *BytesBuffer {
	return &BytesBuffer{
		Buffer: new(bytes.Buffer),
	}
}

func (bb *BytesBuffer) WriteFull(p []byte) WriteResponse {
	n, err := bb.Buffer.Write(p)
	return WriteResponse{
		Written: n,
		Err:     err,
	}
}
