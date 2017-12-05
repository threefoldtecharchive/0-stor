package proto

import (
	"github.com/zero-os/0-stor/client/meta"
	"github.com/zero-os/0-stor/client/meta/encoding"
)

// MarshalMetadata returns the gogo-proto encoding of the data parameter.
// It is important to use this function with the `UnmarshalMetadata` function of this package.
func MarshalMetadata(data meta.Data) ([]byte, error) {
	s := Metadata{
		Epoch:    data.Epoch,
		Key:      data.Key,
		Previous: data.Previous,
		Next:     data.Next,
	}

	if length := len(data.Chunks); length > 0 {
		s.Chunks = make([]Chunk, 0, length)
		for _, chunk := range data.Chunks {
			s.Chunks = append(s.Chunks, Chunk{
				Key:         chunk.Key,
				SizeInBytes: chunk.Size,
				Shards:      chunk.Shards,
			})
		}
	}

	return s.Marshal()
}

// UnmarshalMetadata parses the gogo-proto encoded data
// and stores the result in the value pointed to by the data parameter.
// It is important to use this function with a the `MashalMetadata` function of this package.
func UnmarshalMetadata(b []byte, data *meta.Data) error {
	if b == nil {
		panic("no bytes given to unmarshal to metadata")
	}
	if data == nil {
		panic("no metadata given to unmarshal to")
	}

	var s Metadata
	err := s.Unmarshal(b)
	if err != nil {
		return err
	}

	data.Epoch = s.Epoch
	data.Key = s.Key
	data.Next = s.Next
	data.Previous = s.Previous

	if length := len(s.Chunks); length > 0 {
		data.Chunks = make([]*meta.Chunk, 0, length)
		for _, chunk := range s.Chunks {
			data.Chunks = append(data.Chunks, &meta.Chunk{
				Size:   chunk.SizeInBytes,
				Key:    chunk.Key,
				Shards: chunk.Shards,
			})
		}
	}

	return nil
}

var (
	_ encoding.MarshalMetadata   = MarshalMetadata
	_ encoding.UnmarshalMetadata = UnmarshalMetadata
)
