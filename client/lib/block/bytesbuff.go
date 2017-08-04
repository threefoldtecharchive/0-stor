package block

import (
	"bytes"
)

// BytesBuffer wraps bytes.Buffer to conform to block.Writer interface
type BytesBuffer struct {
	*bytes.Buffer
}

// NewBytesBuffer creates new BytesBuffer
func NewBytesBuffer() *BytesBuffer {
	return &BytesBuffer{
		Buffer: new(bytes.Buffer),
	}
}

// WriteBlock implements block.Writer interface
func (bb *BytesBuffer) WriteBlock(key, val []byte) (int, error) {
	return bb.Buffer.Write(val)
}
