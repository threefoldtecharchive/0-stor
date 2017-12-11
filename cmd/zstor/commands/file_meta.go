package commands

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/zero-os/0-stor/client/metastor"
)

// writeMetaInHumanReadableFormat writes a metastor.Meta struct
// as a human readable format into the writer.
func writeMetaAsHumanReadableFormat(w io.Writer, m metastor.Data) error {
	w.Write([]byte(fmt.Sprintf("Key: %s\n", m.Key)))
	w.Write([]byte(fmt.Sprintf("Epoch: %d\n", m.Epoch)))

	w.Write([]byte("Chunks:\n"))
	for _, chunk := range m.Chunks {
		if chunk == nil {
			return errors.New("nil chunk")
		}

		w.Write([]byte(fmt.Sprintf("\tKey: %s\n", hex.EncodeToString(chunk.Key))))
		w.Write([]byte(fmt.Sprintf("\tSize: %d\n", chunk.Size)))
		w.Write([]byte("Shards:\n"))
		for _, shard := range chunk.Shards {
			w.Write([]byte(fmt.Sprintf("\t\t%s\n", shard)))
		}
		w.Write([]byte{'\n'})
	}

	if m.Previous != nil {
		w.Write([]byte(fmt.Sprintf("Previous: %s\n", m.Previous)))
	}
	if m.Next != nil {
		w.Write([]byte(fmt.Sprintf("Next: %s\n", m.Next)))
	}

	return nil
}

// writeMetaAsJSON writes a metastor.Meta struct
// as a (prettified) JSON.
func writeMetaAsJSON(w io.Writer, m metastor.Data, pretty bool) error {
	encoder := json.NewEncoder(w)
	if pretty {
		encoder.SetIndent("", "\t")
	}

	// turn the in-memory metadata structure,
	// into our custom JSON-friendly metadata structure
	metadata := _metaDataJSON{
		Size:     m.Size,
		Epoch:    m.Epoch,
		Key:      string(m.Key),
		Previous: string(m.Previous),
		Next:     string(m.Next),
	}
	for _, chunk := range m.Chunks {
		metadata.Chunks = append(metadata.Chunks, _metaDataChunkJSON{
			Size:   chunk.Size,
			Key:    string(chunk.Size),
			Shards: chunk.Shards,
		})
	}

	// encode our JSON-friendly metadata structure
	return encoder.Encode(metadata)
}

type _metaDataJSON struct {
	Size     int64                `json:"size"`
	Epoch    int64                `json:"epoch"`
	Key      string               `json:"key"`
	Chunks   []_metaDataChunkJSON `json:"chunks"`
	Previous string               `json:"previous,omitempty"`
	Next     string               `json:"next,omitempty"`
}

type _metaDataChunkJSON struct {
	Size   int64    `json:"size"`
	Key    string   `json:"string"`
	Shards []string `json:"shards"`
}
