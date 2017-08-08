// Package meta  is WIP package for metadata.
// The spec need to be fixed first for further development.
package meta

import (
	"bytes"
	"io"

	"zombiezen.com/go/capnproto2"

	"github.com/zero-os/0-stor/client/meta/schema"
)

// Meta defines a metadata for 0-stor
type Meta struct {
	schema.Metadata
	msg *capnp.Message
}

// New creates new metadata
func New(key []byte, size uint64, shards []string) (*Meta, error) {
	msg, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
	if err != nil {
		return nil, err
	}

	md, err := schema.NewRootMetadata(seg)
	if err != nil {
		return nil, err
	}

	md.SetKey(key)
	md.SetSize(size)

	// set shards
	shardList, err := md.NewShard(int32(len(shards)))
	if err != nil {
		return nil, err
	}
	for i, shard := range shards {
		shardList.Set(i, shard)
	}
	md.SetShard(shardList)

	return &Meta{
		Metadata: md,
		msg:      msg,
	}, nil
}

// Encode encodes the meta to capnp format
func (m Meta) Encode(w io.Writer) error {
	return capnp.NewEncoder(w).Encode(m.msg)
}

// Bytes returns []byte representation of this meta
// in capnp format
func (m Meta) Bytes() ([]byte, error) {
	buf := new(bytes.Buffer)
	err := m.Encode(buf)
	return buf.Bytes(), err
}

// GetShardsSlice returns shards as go string slice
// instead of in canpnp format
func (m Meta) GetShardsSlice() ([]string, error) {
	var shards []string
	shardList, err := m.Shard()
	if err != nil {
		return nil, err
	}
	for i := 0; i < shardList.Len(); i++ {
		shard, err := shardList.At(i)
		if err != nil {
			return nil, err
		}
		shards = append(shards, shard)
	}
	return shards, nil
}

// DecodeReader decodes from given io.Reader and returns
// the parsed Meta
func DecodeReader(rd io.Reader) (*Meta, error) {
	msg, err := capnp.NewDecoder(rd).Decode()
	if err != nil {
		return nil, err
	}

	md, err := schema.ReadRootMetadata(msg)
	if err != nil {
		return nil, err
	}

	return &Meta{
		Metadata: md,
		msg:      msg,
	}, nil
}

// Decode decodes metadata
func Decode(p []byte) (*Meta, error) {
	return DecodeReader(bytes.NewReader(p))
}
