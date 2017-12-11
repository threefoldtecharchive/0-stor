package block

import (
	"github.com/zero-os/0-stor/client/metastor"
)

// NilWriter is a block.wWriter that do nothing
type NilWriter struct {
}

// NewNilWriter creates new NilWriter
func NewNilWriter() *NilWriter {
	return &NilWriter{}
}

// WriteBlock implements block.Writer
// it returns the length of the given value
func (nw NilWriter) WriteBlock(key, value []byte, md *metastor.Data) (*metastor.Data, error) {
	return md, nil
}
