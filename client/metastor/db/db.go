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

package db

import (
	"errors"
	"fmt"
)

var (
	// ErrNotFound is the error returned by a metastor KV database,
	// in case metadata requested couldn't be found.
	ErrNotFound = errors.New("metastor: key couldn't be found")

	// ErrTimeout is the error returned by a metastor KV database,
	// in case the database timed out.
	ErrTimeout = errors.New("metastor: database timed out")

	// ErrUnavailable is the error returned by a metastor KV database,
	// in case the database is unavailable.
	ErrUnavailable = errors.New("metastor: database is unavailable")
)

const (
	// TypeBadger is identifier to specify that we want to use Badger
	// as metadata db
	TypeBadger = "badger"

	// TypeETCD is identifier to specify that we want to use ETCD
	// as metadata db
	TypeETCD = "etcd"
)

// InternalError can be returned by a database as a generic internal error,
// retaining the actual internal error as part of the returned error.
type InternalError struct {
	Type string
	Err  error
}

// Error implements error.Error
func (ie *InternalError) Error() string {
	if ie.Err == nil {
		return fmt.Sprintf("metastor: internal %s database error", ie.Type)
	}
	return fmt.Sprintf("metastor: internal %s database error: %s", ie.Type, ie.Err.Error())
}

// DB interface is the interface defining how to interact with a key value store,
// as ued for metadata storage. ALl DB implements are assumed to be threadsafe.
type DB interface {
	// Set given key in the database equal to the processed metadata.
	Set(namespace, key, metadata []byte) error
	// Get the stored metadata from the database using the given key.
	Get(namespace, key []byte) (metadata []byte, err error)
	// Delete the metadata which is stored as the given key.
	Delete(namespace, key []byte) error
	// Update metadata stored as the given key,
	// as an in-memory-transaction, providing protection against data races.
	// When wishing to update metadata always use this method,
	// rather than a combination of Set+Get.
	Update(namespace, key []byte, cb UpdateCallback) error

	// ListKeys all keys in the given namespace.
	// The keys are sorted in lexicographically order.
	ListKeys(namespace []byte, cb ListCallback) error

	// Close any open (database) resources.
	Close() error
}

// UpdateCallback is the type of callback used to update the processed (encoded)
// metadata, which was already stored, previously.
type UpdateCallback func(orgMetadata []byte) (newMetadata []byte, err error)

// ListCallback is the type of callback used to process the listed keys
type ListCallback func(key []byte) error
