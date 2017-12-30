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

package metastor

import (
	"errors"
)

var (
	// ErrNotFound is the error returned by metadata clients,
	// in case metadata requested using the GetMetadata,
	// couldn't be found.
	ErrNotFound = errors.New("key couldn't be found")

	// ErrNilKey is the error returned by metadata clients,
	// in case a nil key is given as part of a request.
	ErrNilKey = errors.New("nil key given")
)

// Client defines the client API of a metadata server.
// It is used to set, get and delete metadata.
// It is also used as an optional part of the the main 0-stor client,
// in order to fetch the metadata automatically for a given key.
//
// A Client can always be assumed to be thread safe.
type Client interface {
	// SetMetadata sets the metadata,
	// using the key defined as part of the given metadata.
	//
	// An error is returned in case the metadata couldn't be set.
	SetMetadata(md Metadata) error

	// UpdateMetadata updates already existing metadata,
	// returning an error in case there is no metadata to be found for the given key.
	// See `UpdateMetadataFunc` for more information about the required callback.
	//
	// UpdateMetadata panics when no callback is given.
	UpdateMetadata(key []byte, cb UpdateMetadataFunc) (*Metadata, error)

	// GetMetadata returns the metadata linked to the given key.
	//
	// An error is returned in case the linked data couldn't be found.
	// ErrNotFound is returned in case the key couldn't be found.
	// The returned data will always be non-nil in case no error was returned.
	GetMetadata(key []byte) (*Metadata, error)

	// DeleteMetadata deletes the metadata linked to the given key.
	// It is not considered an error if the metadata was already deleted.
	//
	// If an error is returned it should be assumed
	// that the data couldn't be deleted and might still exist.
	DeleteMetadata(key []byte) error

	// Close any open resources of this metadata client.
	Close() error
}

// UpdateMetadataFunc defines a function which receives an already stored metadata,
// and which can modify the metadate, safely, prior to returning it.
// In worst case it can return an error,
// and that error will be propagated back to the user.
type UpdateMetadataFunc func(md Metadata) (*Metadata, error)
