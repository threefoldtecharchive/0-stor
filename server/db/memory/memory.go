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
	"context"
	"sync"

	"github.com/zero-os/0-stor/server/db"

	log "github.com/Sirupsen/logrus"
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
func (mdb *DB) ListItems(ctx context.Context, prefix []byte) (<-chan db.Item, error) {
	if ctx == nil {
		panic("(*DB).ListItems expects a non-nil context.Context")
	}

	// already read-lock, in the main routine,
	// such that for sure the channel has read-access as soon as it starts,
	// and this way nobody can be writing after this call returned (unless there are no keys found)
	mdb.mux.RLock()

	ch := make(chan db.Item, 1)
	go func() {
		defer mdb.mux.RUnlock() // unlock (our) read-lock at the end of this goroutine)

		// close channel when we return
		// (early or because all pairs have been fetched)
		// this is important so that the channel can be used as a range iterator.
		defer close(ch)

		// because of the fact that we want to read-lock until we're finished,
		// we only want to stop a range iteration of a pair,
		// until the returned item has actually been closed.
		closeCh := make(chan struct{}, 1)

		// go through all items
		for k, val := range mdb.m {
			select {
			case <-ctx.Done():
				return

			default:
				key := []byte(k)
				if prefix != nil && !bytes.HasPrefix(key, prefix) {
					continue
				}

				item := &Item{
					key:  key,
					val:  val,
					done: func() { closeCh <- struct{}{} },
				}

				// send item,
				// because `ch` is (unary-)buffered, this line can never be dead-locked
				ch <- item

				// wait until either the item is closed,
				// or until the context finished somehow early (due to an error?!)
				select {
				case <-closeCh:
				case <-ctx.Done():
					// ensure we close item, so it becomes unusable
					err := item.Close()
					if err != nil && err != db.ErrClosedItem {
						log.Warningf("context closed before item is closed, force-closing item (err: %v)", err)
					} else {
						log.Warning("context closed before item is closed, force-closing item")
					}
					return // return early
				}
			}
		}
	}()

	return ch, nil
}

// Close implements DB.Close
func (mdb *DB) Close() error {
	mdb.mux.Lock()
	mdb.m = make(map[string][]byte)
	mdb.mux.Unlock()
	return nil
}

// Item contains key and value of a badger item
type Item struct {
	key  []byte
	val  []byte
	done func()
}

// Key implements interface Item.Key
func (item *Item) Key() []byte {
	if item.done == nil {
		return nil
	}
	return item.key
}

// Value implements interface Item.Value
func (item *Item) Value() ([]byte, error) {
	if item.done == nil {
		return nil, db.ErrClosedItem
	}
	return item.val, nil
}

// Error implements interface Item.Error
func (item *Item) Error() error { return nil }

// Close implements interface Item.Close
func (item *Item) Close() error {
	if item.done == nil {
		return db.ErrClosedItem
	}

	// notifies the DB (item owner),
	// that the item is no longer needed by the user,
	// and we can move on to the next item or finish the update.
	item.done()
	item.done = nil

	return nil
}

// ensure DB/Item interfaces are implemented
var (
	_ db.DB   = (*DB)(nil)
	_ db.Item = (*Item)(nil)
)
