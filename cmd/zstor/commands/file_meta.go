package commands

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/zero-os/0-stor/client/meta"
)

// writeMetaInHumanReadableFormat writes a meta.Meta struct
// as a human readable format into the writer.
func writeMetaAsHumanReadableFormat(w io.Writer, m *meta.Meta) error {
	if m == nil {
		return errors.New("no metadata given")
	}

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
	if m.ConfigPtr != nil {
		w.Write([]byte(fmt.Sprintf("Config pointer: %s\n", m.ConfigPtr)))
	}

	return nil
}

// writeMetaAsJSON writes a meta.Meta struct
// as a (prettified) JSON.
func writeMetaAsJSON(w io.Writer, m *meta.Meta, pretty bool) error {
	if m == nil {
		return errors.New("no metadata given")
	}

	encoder := json.NewEncoder(w)
	if pretty {
		encoder.SetIndent("", "\t")
	}

	return encoder.Encode(m)
}
