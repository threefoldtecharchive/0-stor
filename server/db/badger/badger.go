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

package badger

import (
	"context"
	"os"
	"time"

	"github.com/zero-os/0-stor/server/db"

	log "github.com/Sirupsen/logrus"
	badgerdb "github.com/dgraph-io/badger"
)

const (
	// discardRatio represents the discard ratio for the badger GC
	// https://godoc.org/github.com/dgraph-io/badger#DB.RunValueLogGC
	discardRatio = 0.5

	// GC interval
	cgInterval = 10 * time.Minute
)

// DB implements the db.DB interface
type DB struct {
	db         *badgerdb.DB
	ctx        context.Context
	cancelFunc context.CancelFunc
	seqCache   *sequenceCache
}

// New creates new badger DB with default options
func New(data, meta string) (*DB, error) {
	if len(data) == 0 {
		panic("no data directory defined")
	}
	if len(meta) == 0 {
		panic("no meta directory defined")
	}
	opts := badgerdb.DefaultOptions
	opts.SyncWrites = true
	opts.Dir, opts.ValueDir = meta, data
	return NewWithOpts(opts)
}

// NewWithOpts creates new badger DB with own options
func NewWithOpts(opts badgerdb.Options) (*DB, error) {
	if err := os.MkdirAll(opts.Dir, 0774); err != nil {
		log.Errorf("Meta dir %q couldn't be created: %v", opts.Dir, err)
		return nil, err
	}

	if err := os.MkdirAll(opts.ValueDir, 0774); err != nil {
		log.Errorf("Data dir %q couldn't be created: %v", opts.ValueDir, err)
		return nil, err
	}

	db, err := badgerdb.Open(opts)
	if err != nil {
		return nil, err
	}
	seqCache := newSequenceCache(db)

	ctx, cancel := context.WithCancel(context.Background())

	badger := &DB{
		db:         db,
		ctx:        ctx,
		cancelFunc: cancel,
		seqCache:   seqCache,
	}

	go badger.runGC()

	return badger, err
}

// Set implements DB.Set
func (bdb *DB) Set(key []byte, data []byte) error {
	if key == nil {
		return db.ErrNilKey
	}

	err := bdb.db.Update(func(tx *badgerdb.Txn) error {
		return tx.Set(key, data)
	})
	if err != nil {
		return mapBadgerError(err)
	}
	return nil
}

// SetScoped implements DB.SetScoped
func (bdb *DB) SetScoped(scopeKey, data []byte) ([]byte, error) {
	if scopeKey == nil {
		return nil, db.ErrNilKey
	}

	key, err := bdb.seqCache.IncrementKey(scopeKey)
	if err != nil {
		return nil, err
	}
	err = bdb.db.Update(func(tx *badgerdb.Txn) error {
		return tx.Set(key, data)
	})
	if err != nil {
		return nil, mapBadgerError(err)
	}
	return key, nil
}

// Get implements DB.Get
func (bdb *DB) Get(key []byte) ([]byte, error) {
	if key == nil {
		return nil, db.ErrNilKey
	}

	var val []byte
	err := bdb.db.View(func(tx *badgerdb.Txn) error {
		item, err := tx.Get(key)
		if err != nil {
			return err
		}
		v, err := item.Value()
		if err != nil {
			return err
		}

		val = make([]byte, len(v))
		copy(val, v)
		return nil
	})
	if err != nil {
		return nil, mapBadgerError(err)
	}
	return val, nil
}

// Exists implements DB.Exists
func (bdb *DB) Exists(key []byte) (bool, error) {
	if key == nil {
		return false, db.ErrNilKey
	}

	var exists bool
	err := bdb.db.View(func(tx *badgerdb.Txn) error {
		_, err := tx.Get(key)
		if err != nil {
			if err == badgerdb.ErrKeyNotFound {
				return nil
			}
			return err
		}

		exists = true
		return nil
	})
	if err != nil {
		return false, mapBadgerError(err)
	}
	return exists, nil
}

// Delete implements DB.Delete
func (bdb *DB) Delete(key []byte) error {
	if key == nil {
		return db.ErrNilKey
	}

	err := bdb.db.Update(func(tx *badgerdb.Txn) error {
		err := tx.Delete(key)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return mapBadgerError(err)
	}
	return nil
}

// ListItems implements interface DB.ListItems
func (bdb *DB) ListItems(ctx context.Context, prefix []byte) (<-chan db.Item, error) {
	if ctx == nil {
		panic("(*DB).ListItems expects a non-nil context.Context")
	}

	ch := make(chan db.Item, 1)

	go func() {
		defer close(ch)

		// used to wait until item is closed
		closeCh := make(chan struct{}, 1)

		err := bdb.db.View(
			func(tx *badgerdb.Txn) error {
				opt := badgerdb.DefaultIteratorOptions
				opt.PrefetchValues = false
				it := tx.NewIterator(opt)
				defer it.Close()

				// define predicate to use for the for loop
				var pred func() bool
				if prefix != nil {
					it.Seek(prefix)
					pred = func() bool { return it.ValidForPrefix(prefix) }
				} else {
					it.Rewind()
					pred = it.Valid
				}

				for ; pred(); it.Next() {
					item := &Item{
						item:  it.Item(),
						close: func() { closeCh <- struct{}{} },
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
						return nil // return early
					}
				}

				return nil
			},
		)
		if err != nil {
			err = mapBadgerError(err)
			ch <- &db.ErrorItem{Err: err}
		}
	}()

	return ch, nil
}

// Close implements DB.Close
func (bdb *DB) Close() error {
	// purge all cached sequences
	bdb.seqCache.Purge()

	// cancel (db) context
	bdb.cancelFunc()

	// close db
	err := bdb.db.Close()
	if err != nil {
		return mapBadgerError(err)
	}

	return nil
}

// collectGarbage runs the garbage collection for Badger backend db
func (bdb *DB) collectGarbage() error {
	if err := bdb.db.PurgeOlderVersions(); err != nil {
		return err
	}

	return bdb.db.RunValueLogGC(discardRatio)
}

// runGC triggers the garbage collection for the Badger backend db.
// Should be run as a goroutine
func (bdb *DB) runGC() {
	ticker := time.NewTicker(cgInterval)
	for {
		select {
		case <-ticker.C:
			err := bdb.collectGarbage()
			if err != nil {
				// don't report error when gc didn't result in any cleanup
				if err == badgerdb.ErrNoRewrite {
					log.Debugf("Badger GC: %v", err)
				} else {
					log.Errorf("Badger GC failed: %v", err)
				}
			}
		case <-bdb.ctx.Done():
			return
		}
	}
}

// map badger errors, if we know about them
func mapBadgerError(err error) error {
	switch err {
	case badgerdb.ErrKeyNotFound:
		return db.ErrNotFound
	case badgerdb.ErrConflict:
		return db.ErrConflict
	case badgerdb.ErrEmptyKey:
		return db.ErrNilKey
	default:
		return err
	}
}

// Item contains key and value of a badger item
type Item struct {
	item  *badgerdb.Item
	close func()
}

// Key implements interface Item.Key
func (item *Item) Key() []byte {
	if item.close == nil {
		return nil
	}

	return item.item.Key()
}

// Value implements interface Item.Value
func (item *Item) Value() ([]byte, error) {
	if item.close == nil {
		return nil, db.ErrClosedItem
	}
	return item.item.Value()
}

// Error implements interface Item.Error
func (item *Item) Error() error { return nil }

// Close implements interface Item.Close
func (item *Item) Close() error {
	if item.close == nil {
		return db.ErrClosedItem
	}

	// notify master iterator it can continue
	item.close()

	// make item useless
	item.close = nil
	return nil
}

// ensure DB/Item interfaces are implemented
var (
	_ db.DB   = (*DB)(nil)
	_ db.Item = (*Item)(nil)
)
