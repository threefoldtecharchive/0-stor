package block

import (
	"github.com/zero-os/0-stor/client/metastor"
)

// Writer defines Writer that work on block level
type Writer interface {
	// WriteBlock write the value to the underlying writer
	WriteBlock(key, value []byte, md *metastor.Data) (*metastor.Data, error)
}
