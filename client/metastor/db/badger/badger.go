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
	"errors"
	"os"
	"time"

	dbp "github.com/zero-os/0-stor/client/metastor/db"

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

// New creates a new metastor database implementation,
// using badger on the local FS as storage medium.
//
// Both the data and meta dir are required.
// If you want to be able to specify more options than just
// the required data and metadata directory,
// you can make use of the `NewWithOpts` function,
// as this function will use default options for all other badger options.
func New(data, meta string) (*DB, error) {
	if len(data) == 0 {
		return nil, errors.New("no data directory defined")
	}
	if len(meta) == 0 {
		return nil, errors.New("no meta directory defined")
	}
	opts := badgerdb.DefaultOptions
	opts.SyncWrites = true
	opts.Dir, opts.ValueDir = meta, data
	return NewWithOpts(opts)
}

// NewWithOpts creates a new metastor database implementation,
// using badger on the local FS as storage medium.
//
// Both the data and meta dir, defined as properties of the given options, are required.
func NewWithOpts(opts badgerdb.Options) (*DB, error) {
	if err := os.MkdirAll(opts.Dir, 0774); err != nil {
		log.Errorf("meta dir %q couldn't be created: %v", opts.Dir, err)
		return nil, err
	}

	if err := os.MkdirAll(opts.ValueDir, 0774); err != nil {
		log.Errorf("data dir %q couldn't be created: %v", opts.ValueDir, err)
		return nil, err
	}

	badgerDB, err := badgerdb.Open(opts)
	if err != nil {
		return nil, err
	}

	db := &DB{
		badger: badgerDB,
	}
	db.ctx, db.cancelFunc = context.WithCancel(context.Background())
	go db.runGC()

	return db, nil
}

// DB defines a metastor database implementation,
// using badger on the local FS as its underlying storage medium.
type DB struct {
	badger     *badgerdb.DB
	ctx        context.Context
	cancelFunc context.CancelFunc
}

// Set implements db.Set
func (db *DB) Set(key, metadata []byte) error {
	err := db.badger.Update(func(txn *badgerdb.Txn) error {
		return txn.Set(key, metadata)
	})
	if err != nil {
		return mapBadgerError(err)
	}
	return nil
}

// Get implements db.Get
func (db *DB) Get(key []byte) (metadata []byte, err error) {
	err = db.badger.View(func(txn *badgerdb.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		value, err := item.Value()
		if err != nil {
			return err
		}
		metadata = make([]byte, len(value))
		copy(metadata, value)
		return nil
	})
	if err != nil {
		return nil, mapBadgerError(err)
	}
	return metadata, nil
}

// Delete implements db.Delete
func (db *DB) Delete(key []byte) error {
	err := db.badger.Update(func(txn *badgerdb.Txn) error {
		return txn.Delete(key)
	})
	if err != nil {
		return mapBadgerError(err)
	}
	return nil
}

// Update implements db.Update
func (db *DB) Update(key []byte, cb dbp.UpdateCallback) error {
	err := badgerdb.ErrConflict
	for err == badgerdb.ErrConflict {
		err = db.badger.Update(func(txn *badgerdb.Txn) error {
			// fetch the original stored and encoded metadata
			item, err := txn.Get(key)
			if err != nil {
				return err
			}
			metadata, err := item.Value()
			if err != nil {
				return err
			}

			// user-defined update of the metadata
			metadata, err = cb(metadata)
			if err != nil {
				return err
			}

			// store the updated metadata
			return txn.Set(key, metadata)
		})
	}
	if err != nil {
		return mapBadgerError(err)
	}
	return nil
}

// Close implements metastor.Client.Close
func (db *DB) Close() error {
	// cancel (db) context
	db.cancelFunc()
	// close db
	return db.badger.Close()
}

// collectGarbage runs the garbage collection for Badger backend db
func (db *DB) collectGarbage() error {
	if err := db.badger.PurgeOlderVersions(); err != nil {
		return err
	}
	return db.badger.RunValueLogGC(discardRatio)
}

// runGC triggers the garbage collection for the Badger backend db.
// Should be run as a goroutine
func (db *DB) runGC() {
	ticker := time.NewTicker(cgInterval)
	for {
		select {
		case <-ticker.C:
			err := db.collectGarbage()
			if err != nil {
				// don't report error when gc didn't result in any cleanup
				if err == badgerdb.ErrNoRewrite {
					log.Debugf("Badger GC: %v", err)
				} else {
					log.Errorf("Badger GC failed: %v", err)
				}
			}
		case <-db.ctx.Done():
			return
		}
	}
}

// map badger errors, if we know about them
func mapBadgerError(err error) error {
	switch err {
	case badgerdb.ErrKeyNotFound:
		return dbp.ErrNotFound
	default:
		return err
	}
}
