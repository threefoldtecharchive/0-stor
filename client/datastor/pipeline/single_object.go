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
	"bytes"
	"errors"
	"io"
	"io/ioutil"

	"github.com/threefoldtech/0-stor/client/datastor/pipeline/storage"
	"github.com/threefoldtech/0-stor/client/metastor/metatypes"
)

// NewSingleObjectPipeline creates single-threaded pipeline
// which writes all the content it can read, as a single object (no chunking),
// and processes and stores it all in sequence.
//
// NewSingleObjectPipeline requires a non-nil ChunkStorage and will panic if it is missing.
//
// If no ProcessorConstructor is given, a default constructor will be created for you,
// which will construct a processing.NopProcessor, effectively keeping the data unprocessed at all times.
// While the ProcessorConstructor is optional, it is recommended to define a valid constructor,
// as the storage of unprocessed data is both insecure and inefficient.
//
// If no HasherConstructor is given, a default constructor will be created for you,
// which will construct a 256-bit crypto hasher for you, producing checksums as keys.
// While the HasherConstructor is optional and the default one performs well,
// it is still recommended to define a valid constructor, as it will allow you
// to give a HasherConstructor which creates a crypto Hasher that
// produces signatures as keys, rather than checksums.
func NewSingleObjectPipeline(cs storage.ChunkStorage, pc ProcessorConstructor, hc HasherConstructor) *SingleObjectPipeline {
	if cs == nil {
		panic("no ChunkStorage given")
	}

	if pc == nil {
		pc = DefaultProcessorConstructor
	}
	if hc == nil {
		hc = DefaultHasherConstructor
	}

	return &SingleObjectPipeline{
		hasher:    hc,
		processor: pc,
		storage:   cs,
	}
}

// SingleObjectPipeline defines a single-threaded pipeline,
// which writes all the content it can read, at once,
// and write it as a single object (so no chunking).
// Optionally it can also process the data while writing and reading it.
type SingleObjectPipeline struct {
	hasher    HasherConstructor
	processor ProcessorConstructor
	storage   storage.ChunkStorage
}

// Write implements Pipeline.Write
//
// The following graph visualizes the logic of this pipeline's Write method:
//
// +-----------------------------------------------------------------------+
// | io.Reader+Hasher +-> Processor.Write +-> Storage.Write +-> meta.Meta  |
// +-----------------------------------------------------------------------+
//
// As you can see, it is all blocking, sequential and the input data is not split into chunks.
// Meaning this pipeline will always return single chunk, as long as the data was written successfully.
//
// When an error is returned by a sub-call, at any point,
// the function will return immediately with that error.
func (sop *SingleObjectPipeline) Write(r io.Reader) ([]metatypes.Chunk, error) {
	if r == nil {
		return nil, errors.New("no reader given to read from")
	}

	// create the hasher and processor
	hasher, err := sop.hasher()
	if err != nil {
		return nil, err
	}
	processor, err := sop.processor()
	if err != nil {
		return nil, err
	}

	input, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	hash := hasher.HashBytes(input)

	data, err := processor.WriteProcess(input)
	if err != nil {
		return nil, err
	}

	cfg, err := sop.storage.WriteChunk(data)
	if err != nil {
		return nil, err
	}

	return []metatypes.Chunk{
		{
			Size:    cfg.Size,
			Objects: cfg.Objects,
			Hash:    hash,
		},
	}, nil
}

// Read implements Pipeline.Read
//
// The following graph visualizes the logic of this pipeline's Read method:
//
//    +-------------------------------------------------------------+
//    |                                    +----------------------+ |
//    | metatypes.Chunk +-> storage.Read +--> Processor.Read +    | |
//    |                                    | Hash/Data Validation | |
//    |                                    +-----------+----------+ |
//    |                                                |            |
//    |                                io.Writer <-----+            |
//    +-------------------------------------------------------------+
//
// As you can see, it is all blocking, sequential and the input data is expected to be only 1 chunk.
// If less or more than one chunk is given, an error will be returned before the pipeline even starts reading.
//
// When an error is returned by a sub-call, at any point,
// the function will return immediately with that error.
func (sop *SingleObjectPipeline) Read(chunks []metatypes.Chunk, w io.Writer) error {
	if len(chunks) != 1 {
		return errUnexpectedChunkCount
	}
	if w == nil {
		return errors.New("nil writer")
	}

	// create the hasher and processor
	hasher, err := sop.hasher()
	if err != nil {
		return err
	}
	processor, err := sop.processor()
	if err != nil {
		return err
	}

	data, err := sop.storage.ReadChunk(storage.ChunkConfig{
		Size:    chunks[0].Size,
		Objects: chunks[0].Objects,
	})
	if err != nil {
		return err
	}

	data, err = processor.ReadProcess(data)
	if err != nil {
		return err
	}
	if bytes.Compare(chunks[0].Hash, hasher.HashBytes(data)) != 0 {
		return errors.New("object chunk's data and hash do not match")
	}

	_, err = w.Write(data)
	return err
}

// Check implements Pipeline.Check
func (sop *SingleObjectPipeline) Check(chunks []metatypes.Chunk, fast bool) (storage.CheckStatus, error) {
	if len(chunks) != 1 {
		return storage.CheckStatus(0), errUnexpectedChunkCount
	}
	return sop.storage.CheckChunk(storage.ChunkConfig{
		Size:    chunks[0].Size,
		Objects: chunks[0].Objects,
	}, fast)
}

// Repair implements Pipeline.Repair
func (sop *SingleObjectPipeline) Repair(chunks []metatypes.Chunk) ([]metatypes.Chunk, error) {
	if len(chunks) != 1 {
		return nil, errUnexpectedChunkCount
	}
	cfg, err := sop.storage.RepairChunk(storage.ChunkConfig{
		Size:    chunks[0].Size,
		Objects: chunks[0].Objects,
	})
	if err != nil {
		return nil, err
	}
	return []metatypes.Chunk{{
		Size:    cfg.Size,
		Objects: cfg.Objects,
		Hash:    chunks[0].Hash,
	}}, nil
}

// Delete implements Pipeline.Delete
func (sop *SingleObjectPipeline) Delete(chunks []metatypes.Chunk) error {
	if len(chunks) != 1 {
		return errUnexpectedChunkCount
	}
	return sop.storage.DeleteChunk(storage.ChunkConfig{
		Size:    chunks[0].Size,
		Objects: chunks[0].Objects,
	})
}

// ChunkSize implements Pipeline.ChunkSize
func (sop *SingleObjectPipeline) ChunkSize() int {
	return 0
}

// Close implements Pipeline.Close
func (sop *SingleObjectPipeline) Close() error {
	return sop.storage.Close()
}

var (
	errUnexpectedChunkCount = errors.New(
		"unexpected chunk count, SingleObjectPipeline requires one and only one chunk")
)

var (
	_ Pipeline = (*SingleObjectPipeline)(nil)
)
