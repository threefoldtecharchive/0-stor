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

package test

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	dbp "github.com/zero-os/0-stor/client/metastor/db"
)

// New creates a new in-memory metadata DB,
// using an nothing but an in-memory map as its storage medium.
//
// This implementation is only meant for development and testing purposes,
// and shouldn't be used for anything serious,
// given that it will lose all data as soon as it goes out of scope.
func New() *DB {
	return &DB{
		md:       make(map[string]string),
		versions: make(map[string]uint64),
	}
}

// DB defines client to store metadata,
// storing the an in-memory metadata database,
// used to store encoded metadata directly in an in-memory map.
//
// This implementation is only meant for development and testing purposes,
// and shouldn't be used for anything serious,
// given that it will lose all data as soon as it goes out of scope.
type DB struct {
	md       map[string]string
	versions map[string]uint64
	mux      sync.RWMutex
}

// Set implements db.Set
func (db *DB) Set(namespace, key, metadata []byte) error {
	keyStr := inMemKey(namespace, key)
	db.mux.Lock()
	db.md[keyStr] = string(metadata)
	db.versions[keyStr]++
	db.mux.Unlock()
	return nil
}

// Get implements db.Get
func (db *DB) Get(namespace, key []byte) ([]byte, error) {
	db.mux.RLock()
	metadata, ok := db.md[inMemKey(namespace, key)]
	db.mux.RUnlock()
	if !ok {
		return nil, dbp.ErrNotFound
	}
	return []byte(metadata), nil
}

// Delete implements db.Delete
func (db *DB) Delete(namespace, key []byte) error {
	keyStr := inMemKey(namespace, key)
	db.mux.Lock()
	delete(db.md, keyStr)
	delete(db.versions, keyStr)
	db.mux.Unlock()
	return nil
}

// Update implements db.Update
func (db *DB) Update(namespace, key []byte, cb dbp.UpdateCallback) error {
	keyStr := inMemKey(namespace, key)
	for {
		db.mux.RLock()
		metadataIn, ok := db.md[keyStr]
		version := db.versions[keyStr]
		db.mux.RUnlock()
		if !ok {
			return dbp.ErrNotFound
		}

		metadataOut, err := cb([]byte(metadataIn))
		if err != nil {
			return err
		}

		db.mux.Lock()
		if db.versions[keyStr] != version {
			db.mux.Unlock()
			continue // retry once again
		}

		db.md[keyStr] = string(metadataOut)
		db.versions[keyStr]++
		db.mux.Unlock()
		break
	}

	// updated processed metadata successfully
	return nil
}

// ListKeys implements db.ListKeys
func (db *DB) ListKeys(namespace []byte, cb dbp.ListCallback) error {
	db.mux.Lock()
	defer db.mux.Unlock()

	var (
		keys   []string
		prefix = inMemPrefix(namespace)
	)

	for key := range db.md {
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, strings.TrimPrefix(key, prefix))
		}
	}

	// make it sorted in lexicographically order
	sort.Strings(keys)

	for _, key := range keys {
		if err := cb([]byte(key)); err != nil {
			return err
		}
	}
	return nil
}

// Close implements db.DB
func (db *DB) Close() error {
	db.mux.Lock()
	db.md = make(map[string]string)
	db.versions = make(map[string]uint64)
	db.mux.Unlock()

	return nil
}

func inMemPrefix(namespace []byte) string {
	return fmt.Sprintf("%s/", string(namespace))
}
func inMemKey(namespace, key []byte) string {
	return fmt.Sprintf("%s%s", inMemPrefix(namespace), string(key))
}

var (
	_ dbp.DB = (*DB)(nil)
)
