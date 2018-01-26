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

package memory

import (
	"bytes"
	"sync"

	"github.com/zero-os/0-stor/server/db"
)

// DB implements the db.DB interface
type DB struct {
	m   map[string][]byte
	c   map[string]uint64
	mux sync.RWMutex
}

// New creates a new in-memory DB,
// useful for testing and development purposes only.
func New() *DB {
	return &DB{
		m: make(map[string][]byte),
		c: make(map[string]uint64),
	}
}

// Set implements DB.Set
func (mdb *DB) Set(key []byte, data []byte) error {
	if key == nil {
		return db.ErrNilKey
	}

	b := make([]byte, len(data))
	copy(b, data)
	mdb.mux.Lock()
	mdb.m[string(key)] = b
	mdb.mux.Unlock()
	return nil
}

// SetScoped implements DB.SetScoped
func (mdb *DB) SetScoped(scopeKey, data []byte) ([]byte, error) {
	if scopeKey == nil {
		return nil, db.ErrNilKey
	}

	b := make([]byte, len(data))
	copy(b, data)

	scopeKeyStr := string(scopeKey)

	mdb.mux.Lock()
	key := db.ScopedSequenceKey(scopeKey, mdb.c[scopeKeyStr])
	mdb.c[scopeKeyStr]++
	mdb.m[string(key)] = b
	mdb.mux.Unlock()

	return key, nil
}

// Get implements DB.Get
func (mdb *DB) Get(key []byte) ([]byte, error) {
	if key == nil {
		return nil, db.ErrNilKey
	}

	mdb.mux.RLock()
	val, exists := mdb.m[string(key)]
	if !exists {
		mdb.mux.RUnlock()
		return nil, db.ErrNotFound
	}

	b := make([]byte, len(val))
	copy(b, val)
	mdb.mux.RUnlock()

	return b, nil
}

// Exists implements DB.Exists
func (mdb *DB) Exists(key []byte) (bool, error) {
	if key == nil {
		return false, db.ErrNilKey
	}

	mdb.mux.RLock()
	_, exists := mdb.m[string(key)]
	mdb.mux.RUnlock()
	return exists, nil
}

// Delete implements DB.Delete
func (mdb *DB) Delete(key []byte) error {
	if key == nil {
		return db.ErrNilKey
	}

	mdb.mux.Lock()
	delete(mdb.m, string(key))
	mdb.mux.Unlock()
	return nil
}

// ListItems implements interface DB.ListItems
func (mdb *DB) ListItems(cb func(db.Item) error, prefix []byte) error {
	if cb == nil {
		panic("no callback given")
	}

	mdb.mux.RLock()
	defer mdb.mux.RUnlock()

	var (
		err error
		key []byte
	)

	for k, val := range mdb.m {
		key = []byte(k)
		if prefix != nil && !bytes.HasPrefix(key, prefix) {
			continue
		}

		item := &Item{
			key: key,
			val: val,
		}
		err = cb(item)
		if err != nil {
			return err
		}
	}

	return nil
}

// Close implements DB.Close
func (mdb *DB) Close() error {
	mdb.mux.Lock()
	mdb.m = make(map[string][]byte)
	mdb.mux.Unlock()
	return nil
}

// Item contains key and value of a in-memory item
type Item struct {
	key []byte
	val []byte
}

// Key implements interface Item.Key
func (item *Item) Key() ([]byte, error) {
	return item.key, nil
}

// Value implements interface Item.Value
func (item *Item) Value() ([]byte, error) {
	return item.val, nil
}

// ensure DB/Item interfaces are implemented
var (
	_ db.DB   = (*DB)(nil)
	_ db.Item = (*Item)(nil)
)
