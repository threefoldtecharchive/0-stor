/*
 * Copyright (C) 2017-2018 GIG Technology NV and Contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package commands

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/threefoldtech/0-stor/client/metastor/metatypes"
)

// writeMetaInHumanReadableFormat writes a metastor.Meta struct
// as a human readable format into the writer.
func writeMetaAsHumanReadableFormat(w io.Writer, m metatypes.Metadata) error {
	w.Write([]byte(fmt.Sprintf("Key: %s\n", m.Key)))
	w.Write([]byte(fmt.Sprintf("CreationEpoch: %d\n", m.CreationEpoch)))
	w.Write([]byte(fmt.Sprintf("LastWriteEpoch: %d\n", m.LastWriteEpoch)))

	w.Write([]byte("Chunks:\n"))
	for _, chunk := range m.Chunks {
		w.Write([]byte(fmt.Sprintf("\tSize: %d\n", chunk.Size)))
		w.Write([]byte("Objects:\n"))
		for _, object := range chunk.Objects {
			w.Write([]byte(fmt.Sprintf("\t\tKey: %s\n", object.Key)))
			w.Write([]byte(fmt.Sprintf("\t\tShardID: %s\n", object.ShardID)))
		}
		w.Write([]byte(fmt.Sprintf("\tHash: %s\n", chunk.Hash)))
		w.Write([]byte{'\n'})
	}

	if m.PreviousKey != nil {
		w.Write([]byte(fmt.Sprintf("PreviousKey: %s\n", m.PreviousKey)))
	}
	if m.NextKey != nil {
		w.Write([]byte(fmt.Sprintf("NextKey: %s\n", m.NextKey)))
	}

	return nil
}

// writeMetaAsJSON writes a metastor.Meta struct
// as a (prettified) JSON.
func writeMetaAsJSON(w io.Writer, m metatypes.Metadata, pretty bool) error {
	encoder := json.NewEncoder(w)
	if pretty {
		encoder.SetIndent("", "\t")
	}

	// turn the in-memory metadata structure,
	// into our custom JSON-friendly metadata structure
	metadata := _MetaDataJSON{
		Key:            string(m.Key),
		Size:           m.Size,
		CreationEpoch:  m.CreationEpoch,
		LastWriteEpoch: m.LastWriteEpoch,
		PreviousKey:    string(m.PreviousKey),
		NextKey:        string(m.NextKey),
	}
	for _, chunk := range m.Chunks {
		c := _MetaDataChunkJSON{
			Size: chunk.Size,
			Hash: string(chunk.Hash),
		}
		for _, object := range chunk.Objects {
			c.Objects = append(c.Objects, _MetaDataObjectJSON{
				Key:   string(object.Key),
				Shard: object.ShardID,
			})
		}
		metadata.Chunks = append(metadata.Chunks, c)
	}

	// encode our JSON-friendly metadata structure
	return encoder.Encode(metadata)
}

type _MetaDataJSON struct {
	Key            string               `json:"key"`
	Size           int64                `json:"size"`
	CreationEpoch  int64                `json:"creation_epoch"`
	LastWriteEpoch int64                `json:"last_write_epoch"`
	Chunks         []_MetaDataChunkJSON `json:"chunks"`
	PreviousKey    string               `json:"previous_key,omitempty"`
	NextKey        string               `json:"next_key,omitempty"`
}

type _MetaDataChunkJSON struct {
	Size    int64                 `json:"size"`
	Objects []_MetaDataObjectJSON `json:"objects"`
	Hash    string                `json:"hash"`
}

type _MetaDataObjectJSON struct {
	Key   string `json:"key"`
	Shard string `json:"shard"`
}
