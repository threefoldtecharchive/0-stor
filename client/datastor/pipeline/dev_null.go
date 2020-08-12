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
	"crypto/md5"
	"errors"
	"io"

	"github.com/threefoldtech/0-stor/client/datastor/pipeline/storage"
	"github.com/threefoldtech/0-stor/client/metastor/metatypes"
)

var (
	errOperationNotSupported = errors.New("operation not supported")
)

type devNull struct {
	chunkSize int
}

// NewDevNull create a new /dev/null pipeline where all data is discared
func NewDevNull(chunkSize int) Pipeline {
	return &devNull{chunkSize}
}

func (d *devNull) Write(r io.Reader) ([]metatypes.Chunk, error) {
	var chunks []metatypes.Chunk
	buf := make([]byte, d.chunkSize)
	for {
		reader := io.LimitReader(r, int64(d.chunkSize))
		hasher := md5.New()
		n, err := io.CopyBuffer(hasher, reader, buf)
		if err != nil {
			return chunks, err
		} else if n == 0 {
			break
		}

		chunk := metatypes.Chunk{
			Size: n,
			Hash: hasher.Sum(nil),
		}

		chunks = append(chunks, chunk)
	}

	return chunks, nil
}

// Read content from a zstordb cluster,
// the details depend upon the specific implementation.
func (d *devNull) Read(chunks []metatypes.Chunk, w io.Writer) error {
	return errOperationNotSupported
}

// Check if content stored on a zstordb cluster is (still) valid,
// the details depend upon the specific implementation.
func (d *devNull) Check(chunks []metatypes.Chunk, fast bool) (storage.CheckStatus, error) {
	return storage.CheckStatusInvalid, errOperationNotSupported
}

// Repair content stored on a zstordb cluster,
// the details depend upon the specific implementation.
func (d *devNull) Repair(chunks []metatypes.Chunk) ([]metatypes.Chunk, error) {
	return nil, errOperationNotSupported
}

// Delete content stored on a zstordb cluster,
// the details depend upon the specific implementation.
func (d *devNull) Delete(chunks []metatypes.Chunk) error {
	return errOperationNotSupported
}

// ChunkSize returns the fixed chunk size, which is size used for all chunks,
// except for the last chunk which might be less or equal to that chunk size.
func (d *devNull) ChunkSize() int {
	return d.chunkSize
}

// Close any open resources.
func (d *devNull) Close() error {
	return nil
}
