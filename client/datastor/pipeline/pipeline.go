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

package pipeline

import (
	"io"

	"github.com/threefoldtech/0-stor/client/datastor/pipeline/crypto"
	"github.com/threefoldtech/0-stor/client/datastor/pipeline/storage"
	"github.com/threefoldtech/0-stor/client/metastor/metatypes"
	"github.com/threefoldtech/0-stor/client/processing"
)

// Pipeline defines the interface to write and read content
// to/from a zstordb cluster.
//
// Prior to storage content can be
// processed (compressed and/or encrypted), as well as split
// (into smaller chunks) and distributed in terms of replication
// or erasure coding.
//
// Content written in one way,
// has to be read in a way that is compatible. Meaning that if
// content was compressed and encrypted using a certain configuration,
// it will have to be decrypted and decompressed using that same configuration,
// or else the content will not be able to be read.
type Pipeline interface {
	// Write content to a zstordb cluster,
	// the details depend upon the specific implementation.
	Write(r io.Reader) ([]metatypes.Chunk, error)
	// Read content from a zstordb cluster,
	// the details depend upon the specific implementation.
	Read(chunks []metatypes.Chunk, w io.Writer) error

	// Check if content stored on a zstordb cluster is (still) valid,
	// the details depend upon the specific implementation.
	Check(chunks []metatypes.Chunk, fast bool) (storage.CheckStatus, error)
	// Repair content stored on a zstordb cluster,
	// the details depend upon the specific implementation.
	Repair(chunks []metatypes.Chunk) ([]metatypes.Chunk, error)

	// Delete content stored on a zstordb cluster,
	// the details depend upon the specific implementation.
	Delete(chunks []metatypes.Chunk) error

	// ChunkSize returns the fixed chunk size, which is size used for all chunks,
	// except for the last chunk which might be less or equal to that chunk size.
	ChunkSize() int

	// Close any open resources.
	Close() error
}

// Constructor types which are used to create unique instances of the types involved,
// for each branch (goroutine) of a pipeline.
type (
	// HasherConstructor is a constructor type which is used to create a unique
	// Hasher for each goroutine where the Hasher is needed within a pipeline.
	// This is required as a (crypto) Hasher is not thread-safe.
	HasherConstructor func() (crypto.Hasher, error)
	// ProcessorConstructor is a constructor type which is used to create a unique
	// Processor for each goroutine where the Processor is needed within a pipeline.
	// This is required as a Processor is not thread-safe.
	ProcessorConstructor func() (processing.Processor, error)
)

// DefaultHasherConstructor is an implementation of a HasherConstructor,
// which can be used as a safe default HasherConstructor, by pipeline implementations,
// should such a constructor not be given by the user.
func DefaultHasherConstructor() (crypto.Hasher, error) {
	return crypto.NewDefaultHasher256(nil)
}

// DefaultProcessorConstructor is an implementation of a ProcessorConstructor,
// which can be used as a safe default ProcessorConstructor, by pipeline implementations,
// should such a constructor not be given by the user.
func DefaultProcessorConstructor() (processing.Processor, error) {
	return processing.NopProcessor{}, nil
}
