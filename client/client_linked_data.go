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

package client

import (
	"errors"
	"io"

	"github.com/threefoldtech/0-stor/client/datastor/pipeline"
	"github.com/threefoldtech/0-stor/client/metastor"
	"github.com/threefoldtech/0-stor/client/metastor/metatypes"

	log "github.com/sirupsen/logrus"
)

// NOTE:
// the functionality provided in this file is to be considered EXPERIMENTAL,
// and might be moved or changed in a future milestone.
// See https://github.com/threefoldtech/0-stor/issues/424 for more information.

var (
	// ErrInvalidTraverseIterator is an error returned when (meta)data
	// of an iterator is requested, while that iterator is in an invalid state.
	ErrInvalidTraverseIterator = errors.New(
		"TraverseIterator is invalid: did you call (TraverseIterator).Next?")
	// ErrInvalidEpochRange is an error returned when,
	// during the creation of a traverse iterator,
	// the given epoch range is invalid (e.g. start > end).
	ErrInvalidEpochRange = errors.New(
		"cannot create traverse iterator: epoch range is invalid")
)

// TraverseIterator defines the interface of an iterator,
// which is returned by a client traverse function.
type TraverseIterator interface {
	// Next moves the iterator one (valid) position forward,
	// returning false if the iterator has been exhausted.
	//
	// Next has to be called before any (meta)data can be fetched or read.
	Next() bool

	// PeekNextKey returns the next key in line.
	// Note that due to the specified epoch range it
	// might mean that the data of this key will never be available,
	// in case the creation time of the linked (meta)data is
	// not within the specified time range.
	//
	// False is returned in case the iterator has been exhausted,
	// and thus no next key is lined up.
	PeekNextKey() ([]byte, bool)

	// GetMetadata returns the current (and already fetched) metadata.
	//
	// An error is returned in case no metadata is available,
	// due to the iterator being in an invalid state.
	GetMetadata() (*metatypes.Metadata, error)

	// ReadData reads the data available for the current metadata,
	// and writes it to the specified writer.
	//
	// An error is returned in case the iterator is in an invalid state,
	// and thus no data is available to be read.
	ReadData(w io.Writer) error
}

// WriteLinked writes the data to a 0-stor cluster,
// storing the metadata using the internal metastor client,
// as well as linking the metadata created for this data,
// to the metadata linked to the given previous key.
//
// This method is to be considered EXPERIMENTAL,
// and might be moved or changed in a future milestone.
// See https://github.com/threefoldtech/0-stor/issues/424 for more information.
func (c *Client) WriteLinked(key, prevKey []byte, r io.Reader) (meta, prevMeta *metatypes.Metadata, err error) {
	if len(key) == 0 {
		return nil, nil, ErrNilKey // ensure a key is given
	}
	if len(prevKey) == 0 {
		// ensure a prevKey is given
		// this is not optional here,
		// if you don't want prevKey you should use the Write method
		return nil, nil, ErrNilKey
	}

	// ensure the prevmetadata exists, and lock it until we've updated it,
	// however we want to make sure that we only create+store the current metadata once
	prevMeta, err = c.metastorClient.UpdateMetadata(prevKey,
		func(prevMetadata metatypes.Metadata) (*metatypes.Metadata, error) {
			// create the current metadata, should it not be created yet
			if meta == nil {
				// process and write the data
				chunks, err := c.dataPipeline.Write(r)
				if err != nil {
					return nil, err
				}

				// create new metadata, as we'll overwrite either way
				now := EpochNow()
				meta = &metatypes.Metadata{
					Key:            key,
					CreationEpoch:  now,
					LastWriteEpoch: now,
					PreviousKey:    prevKey,
				}

				// set/update chunks and size in metadata
				meta.Chunks = chunks
				for _, chunk := range chunks {
					meta.Size += chunk.Size
				}

				// store current metadata
				err = c.metastorClient.SetMetadata(*meta)
				if err != nil {
					return nil, err
				}
			}

			// do the actual update of the prevMetadata
			prevMetadata.NextKey = key
			return &prevMetadata, nil
		})
	return meta, prevMeta, err
}

// Traverse traverses the stored (meta)data,
// which is chained together using the (*Client).WriteLinked method.
// It starts searching from a given startKey and will iterate through all (meta)data,
// which has a registered CreationEpoch in the given inclusive epoch range.
//
// Both the fromEpoch and toEpoch input parameters are optional and do not have to be given.
// Any value of 0 or less can be given as to not specify them. Not giving an epoch limit,
// simply means that this limit won't be used/enforced.
//
// An error will be returned in case no (valid) startKey is given,
// or in case the given epoch range is invalid (fromEpoch > toEpoch).
//
// The returned TraverseIterator is only valid,
// as long as the client which created and owns that iterator is valid (e.g. not closed).
// This traverse iterator is NOT /THREAD-SAFE/.
//
// This method is to be considered EXPERIMENTAL,
// and might be moved or changed in a future milestone.
// See https://github.com/threefoldtech/0-stor/issues/424 for more information.
func (c *Client) Traverse(startKey []byte, fromEpoch, toEpoch int64) (TraverseIterator, error) {
	if len(startKey) == 0 {
		return nil, ErrNilKey
	}
	if fromEpoch <= 0 {
		fromEpoch = -1
	}
	if toEpoch <= 0 {
		toEpoch = -1
	}
	if toEpoch != -1 && fromEpoch > toEpoch {
		return nil, ErrInvalidEpochRange
	}
	return &forwardTraverseIterator{
		traverseIteratorState: traverseIteratorState{
			dataPipeline: c.dataPipeline,
		},
		nextKey:    startKey,
		fromEpoch:  fromEpoch,
		toEpoch:    toEpoch,
		metaClient: c.metastorClient,
	}, nil
}

// TraversePostOrder traverses the stored (meta)data, backwards,
// which is chained together using the (*Client).WriteLinked method.
// It starts searching from a given startKey and will iterate through all (meta)data,
// which has a registered CreationEpoch in the given inclusive epoch range.
//
// Both the fromEpoch and toEpoch input parameters are optional and do not have to be given.
// Any value of 0 or less can be given as to not specify them. Not giving an epoch limit,
// simply means that this limit won't be used/enforced.
//
// As this method traverses backwards, the startKey is expected
// to be the newest data as the given fromEpoch should be the most recent time in this chain.
//
// An error will be returned in case no (valid) startKey is given,
// or in case the given epoch range is invalid (toEpoch > fromEpoch).
//
// The returned TraverseIterator is only valid,
// as long as the client which created and owns that iterator is valid (e.g. not closed).
// This traverse iterator is NOT /THREAD-SAFE/.
//
// This method is to be considered EXPERIMENTAL,
// and might be moved or changed in a future milestone.
// See https://github.com/threefoldtech/0-stor/issues/424 for more information.
func (c *Client) TraversePostOrder(startKey []byte, fromEpoch, toEpoch int64) (TraverseIterator, error) {
	if len(startKey) == 0 {
		return nil, ErrNilKey
	}
	if fromEpoch <= 0 {
		fromEpoch = -1
	}
	if toEpoch <= 0 {
		toEpoch = -1
	}
	if fromEpoch != -1 && toEpoch > fromEpoch {
		return nil, ErrInvalidEpochRange
	}
	return &backwardTraverseIterator{
		traverseIteratorState: traverseIteratorState{
			dataPipeline: c.dataPipeline,
		},
		previousKey: startKey,
		fromEpoch:   fromEpoch,
		toEpoch:     toEpoch,
		metaClient:  c.metastorClient,
	}, nil
}

// traverseIteratorState is the core of both iterators defined in this file.
// It defines the (meta)data fetcher methods, as implementations to become a TraverseIterator.
// The actual traverse iterator type will encapsulate this method, to provide the
// required Next method, to complete the implementation.
//
// The state contains a static dataPipeline, provided at construction time,
// and shared with the Client owner. After that client closes,
// this traverse iterator should no longer be used, as the dataPipeline will no longer function.
//
// It also contains a cached metadata structure pointer,
// which contains the current metadata state, the iterator is on.
// If this metadata is nil, the iterator is to be considered invalid
// (e.g. Next was never called (successfully)).
type traverseIteratorState struct {
	md *metatypes.Metadata

	dataPipeline pipeline.Pipeline
}

// Getmetadata implements TraverseIterator.GetMetadata
func (state *traverseIteratorState) GetMetadata() (*metatypes.Metadata, error) {
	if state.md == nil {
		return nil, ErrInvalidTraverseIterator
	}
	return state.md, nil
}

// ReadData implements TraverseIterator.ReadData
func (state *traverseIteratorState) ReadData(w io.Writer) error {
	if w == nil {
		panic("TraverseIterator: ReadData: required io.Writer is nil")
	}
	if state.md == nil {
		return ErrInvalidTraverseIterator
	}
	return state.dataPipeline.Read(state.md.Chunks, w)
}

// forwardTraverseIterator contains the logic and state
// to move a traverse iterator forward (in time).
type forwardTraverseIterator struct {
	traverseIteratorState

	nextKey            []byte
	fromEpoch, toEpoch int64
	metaClient         *metastor.Client
}

// Next implements TraverseIterator.Next
func (it *forwardTraverseIterator) Next() bool {
	for it.nextKey != nil {
		md, err := it.metaClient.GetMetadata(it.nextKey)
		if err != nil {
			log.Errorf("error while fetching metdata for key %s: %v",
				it.nextKey, err)
			return false
		}

		if it.toEpoch != -1 && md.CreationEpoch > it.toEpoch {
			// we've exhausted the iterator
			it.nextKey = nil
			return false
		}

		it.nextKey = md.NextKey

		if it.fromEpoch != -1 && md.CreationEpoch < it.fromEpoch {
			continue
		}

		it.traverseIteratorState.md = md
		return true
	}
	return false
}

// PeekNextKey implements TraverseIterator.PeekNextKey
func (it *forwardTraverseIterator) PeekNextKey() ([]byte, bool) {
	if it.nextKey == nil {
		return nil, false
	}
	return it.nextKey, true
}

// backwardTraverseIterator contains the logic and state
// to move a traverse iterator backward (in time).
type backwardTraverseIterator struct {
	traverseIteratorState

	previousKey        []byte
	fromEpoch, toEpoch int64
	metaClient         *metastor.Client
}

// Next implements TraverseIterator.Next
func (it *backwardTraverseIterator) Next() bool {
	for it.previousKey != nil {
		md, err := it.metaClient.GetMetadata(it.previousKey)
		if err != nil {
			log.Errorf("error while fetching metdata for key %s: %v",
				it.previousKey, err)
			return false
		}

		if it.toEpoch != -1 && md.CreationEpoch < it.toEpoch {
			// we've exhausted the iterator
			it.previousKey = nil
			return false
		}

		it.previousKey = md.PreviousKey

		if it.fromEpoch != -1 && md.CreationEpoch > it.fromEpoch {
			continue
		}

		it.traverseIteratorState.md = md
		return true
	}
	return false
}

// PeekNextKey implements TraverseIterator.PeekNextKey
func (it *backwardTraverseIterator) PeekNextKey() ([]byte, bool) {
	if it.previousKey == nil {
		return nil, false
	}
	return it.previousKey, true
}

// Ensure our iterators adhere to the TraverseIterator interface.
var (
	_ TraverseIterator = (*forwardTraverseIterator)(nil)
	_ TraverseIterator = (*backwardTraverseIterator)(nil)
)
