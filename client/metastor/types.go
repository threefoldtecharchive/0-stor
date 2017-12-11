package metastor

import (
	"bytes"
)

type (
	// Data represents the metadata of some data.
	// It is stored in some metadata server, and references
	// to data (see: Chunk), which is stored in a zstordb server.
	// This structure can be used as a node as part of a linked list.
	// It can be a (reversed) linked list as well as a double linked list.
	Data struct {
		// Size in bytes represents the total size of all referenced chunks combined.
		Size int64
		// Epoch defines the nano epoch time of when this data was stored.
		Epoch int64
		// Key defines the key of the data, and is chosen by the owner of this data.
		Key []byte
		// Chunks is the list of all chunk (references) that make up the data, when combined.
		Chunks []*Chunk
		// Previous is an optional key to the previous Data (node),
		// in case this Data (node) is used as part of a reversed/double linked list.
		Previous []byte
		// Next is an optional key to the next Data (node),
		// in case this Data (node) is used as part of a (double) linked list.
		Next []byte
	}

	// Chunk represents the metadata of a chunk of data.
	// It is stored in some metadata server as part of its Metadata owner.
	// A chunk is to be be stored in one or multiple zstordb servers,
	// where the chunk is referenced by its key and each shard addresses a zstordb server.
	Chunk struct {
		// Size in bytes represents the size of all the data this chunk references.
		Size int64
		// Key defines the key of the data, and is chosen automatically by the system.
		// When using the 0-stor client, this will be the crypto hash of the input (plain) chunk data.
		Key []byte
		// Shards contains the list of addresses,
		// where each address references a zstordb server where this chunk has stored some data.
		// A chunk can be stored on one or multiple shards.
		Shards []string
	}
)

// GetChunk return a chunk by it's key.
//
// If the key is not found, a new chunk is added
// to the metadata object and returned
func (md *Data) GetChunk(key []byte) *Chunk {
	if md == nil {
		panic("no metadata given to get a chunk from")
	}

	for _, chunk := range md.Chunks {
		if bytes.Compare(chunk.Key, key) == 0 {
			return chunk
		}
	}
	chunk := &Chunk{Key: key}
	md.Chunks = append(md.Chunks, chunk)
	return chunk
}
