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

package metatypes

// TODO:
// Investigate if we can move Previous/Next key to custom/extra
// metadata, that can be used by those users who desire it.
// It seems like a nice example of a feature that a user might want to provide
// for its file data, but that not everybody needs, and thus
// it might help us define a better format, that allows for additional features like this.

type (
	// Metadata represents the metadata of some data.
	// It is stored in some metadata server/cluster, and references
	// the data (see: Chunk and Object), which is stored in a zstordb cluster.
	Metadata struct {
		// Namespace defines namespace of the data,
		// and is chosen by the owner of this data.
		Namespace []byte

		// Key defines the key of the data,
		// and is chosen by the owner of this data.
		Key []byte

		// Size represent the total size of the data before any processing
		Size int64

		// StorageSize in bytes represents the total size of all chunks,
		// that make up the stored data, combined.
		StorageSize int64

		// CreationEpoch defines the time this data was initially created,
		// in the Unix epoch format, in nano seconds.
		CreationEpoch int64
		// LastWriteEpoch defines the time this data was last modified (e.g. repaired),
		// in the Unix epoch format, in nano seconds.
		LastWriteEpoch int64

		// Chunks is the metadata list of all chunks that make up the data, when combined.
		Chunks []Chunk

		// ChunkSize is the fixed chunk size, which is size used for all chunks,
		// except for the last chunk which might be less or equal to that chunk size.
		ChunkSize int32

		// PreviousKey is an optional key to the previous Metadata (node),
		// in case this Metadata (node) is used as part of a reversed/double linked list.
		PreviousKey []byte
		// NextKey is an optional key to the next Metadata (node),
		// in case this Metadata (node) is used as part of a (double) linked list.
		NextKey []byte

		// UserDefined is user defined metadata,
		// in case user want to store additional metadata.
		UserDefined map[string]string
	}

	// Chunk represents the metadata of a chunk of data.
	// It is stored in some metadata server as part of its Metadata owner.
	// A chunk is to be be stored in one or multiple zstordb servers,
	// where the chunk is referenced by its key and each shard addresses a zstordb server.
	Chunk struct {
		// Size in bytes represents the total size of all data (objects)
		// this chunk contains.
		Size int64

		// Objects defines the metadata of the objects
		// that make up this chunk.
		Objects []Object

		// Hash contains the checksum/signature of the entire chunk,
		// meaning the data of all objects (of this chunk) combined.
		Hash []byte
	}

	// Object represents the metadata of an object,
	// which makes up alone or together with other objects, a chunk.
	// An object is stored on a shard (see: zstordb server),
	// and is defined by a unique key, generated and defined by the server it is stored on.
	Object struct {
		// Key of the Object
		Key []byte
		// ShardID defines the ID of the shard the object is stored on
		ShardID string
	}
)
