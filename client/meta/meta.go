// Package meta  is WIP package for metadata.
// The spec need to be fixed first for further development.
package meta

import (
	"bytes"
	"encoding/gob"
	"io"
)

// Meta defines a metadata
type Meta struct {
	Size   uint64
	Key    []byte
	Shards []string
}

// Encode encodes the meta to `gob` format
func (m Meta) Encode(w io.Writer) error {
	return gob.NewEncoder(w).Encode(m)

}

// Bytes returns []byte representation of this meta
// in gob format
func (m Meta) Bytes() ([]byte, error) {
	buf := new(bytes.Buffer)
	err := m.Encode(buf)
	return buf.Bytes(), err
}

// Decode decodes metadata
func Decode(p []byte) (*Meta, error) {
	var meta Meta
	return &meta, gob.NewDecoder(bytes.NewReader(p)).Decode(&meta)
}
