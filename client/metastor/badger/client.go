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

	"github.com/zero-os/0-stor/client/metastor"
	"github.com/zero-os/0-stor/client/metastor/encoding"
	"github.com/zero-os/0-stor/client/metastor/encoding/proto"

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

// NewClient creates new Metadata client, using badger on the local FS as storage medium.
// The encoding pair of functions is optional, and if nil,
// Proto Marshal/Unmarshal funcs are used for the (un)marshaling of metadata values.
//
// Both the data and meta dir are required.
// If you want to be able to specify more options than just
// the required data and metadata directory,
// you can make use of the `NewClientWithOpts` function.
//
// Note that if a pair is defined,
// NewClient will panic if the Marshal and/or Unmarshal func(s) are undefined/nil.
func NewClient(data, meta string, pair *encoding.MarshalFuncPair) (*Client, error) {
	if len(data) == 0 {
		panic("no data directory defined")
	}
	if len(meta) == 0 {
		panic("no meta directory defined")
	}
	opts := badgerdb.DefaultOptions
	opts.SyncWrites = true
	opts.Dir, opts.ValueDir = meta, data
	return NewClientWithOpts(opts, pair)
}

// NewClientWithOpts creates new Metadata client, using badger on the local FS as storage medium.
// The encoding pair of functions is optional, and if nil,
// Proto Marshal/Unmarshal funcs are used for the (un)marshaling of metadata values.
//
// Both the data and meta dir, defined as properties of the given options, are required.
//
// Note that if a pair is defined,
// NewClient will panic if the Marshal and/or Unmarshal func(s) are undefined/nil.
func NewClientWithOpts(opts badgerdb.Options, pair *encoding.MarshalFuncPair) (*Client, error) {
	if pair != nil {
		if pair.Marshal == nil {
			panic("defined MarshalFuncPair is missing a marshal func")
		}
		if pair.Unmarshal == nil {
			panic("defined MarshalFuncPair is missing an unmarshal func")
		}
	} else {
		pair = &encoding.MarshalFuncPair{
			Marshal:   proto.MarshalMetadata,
			Unmarshal: proto.UnmarshalMetadata,
		}
	}

	if err := os.MkdirAll(opts.Dir, 0774); err != nil {
		log.Errorf("meta dir %q couldn't be created: %v", opts.Dir, err)
		return nil, err
	}

	if err := os.MkdirAll(opts.ValueDir, 0774); err != nil {
		log.Errorf("data dir %q couldn't be created: %v", opts.ValueDir, err)
		return nil, err
	}

	db, err := badgerdb.Open(opts)
	if err != nil {
		return nil, err
	}

	client := &Client{
		db:        db,
		marshal:   pair.Marshal,
		unmarshal: pair.Unmarshal,
	}
	client.ctx, client.cancelFunc = context.WithCancel(context.Background())
	go client.runGC()

	return client, nil
}

// Client defines client to store metadata,
// using badger on the local FS as its underlying storage medium.
type Client struct {
	db         *badgerdb.DB
	ctx        context.Context
	cancelFunc context.CancelFunc

	marshal   encoding.MarshalMetadata
	unmarshal encoding.UnmarshalMetadata
}

// SetMetadata implements metastor.Client.SetMetadata
func (c *Client) SetMetadata(data metastor.Metadata) error {
	if len(data.Key) == 0 {
		return metastor.ErrNilKey
	}

	bytes, err := c.marshal(data)
	if err != nil {
		return err
	}

	err = c.db.Update(func(txn *badgerdb.Txn) error {
		return txn.Set(data.Key, bytes)
	})
	if err != nil {
		return mapBadgerError(err)
	}
	return nil
}

// UpdateMetadata implements metastor.Client.UpdateMetadata
func (c *Client) UpdateMetadata(key []byte, cb metastor.UpdateMetadataFunc) (*metastor.Metadata, error) {
	if cb == nil {
		panic("Metastor (badger) Client: required UpdateMetadata CB is not given")
	}
	if len(key) == 0 {
		return nil, metastor.ErrNilKey
	}

	var (
		meta *metastor.Metadata
		err  = badgerdb.ErrConflict
	)
	for err == badgerdb.ErrConflict {
		err = c.db.Update(func(txn *badgerdb.Txn) error {
			// fetch and unmarshal the original stored metadata
			item, err := txn.Get(key)
			if err != nil {
				return err
			}
			bytes, err := item.Value()
			if err != nil {
				return err
			}
			var data metastor.Metadata
			err = c.unmarshal(bytes, &data)
			if err != nil {
				return err
			}

			// user-defined update of the metadata
			meta, err = cb(data)
			if err != nil {
				return err
			}

			// marshal and store the updated metadata
			bytes, err = c.marshal(*meta)
			if err != nil {
				return err
			}
			return txn.Set(key, bytes)
		})
	}
	if err != nil {
		return nil, mapBadgerError(err)
	}
	return meta, nil
}

// GetMetadata implements metastor.Client.GetMetadata
func (c *Client) GetMetadata(key []byte) (*metastor.Metadata, error) {
	if len(key) == 0 {
		return nil, metastor.ErrNilKey
	}

	var data metastor.Metadata
	err := c.db.View(func(txn *badgerdb.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		bytes, err := item.Value()
		if err != nil {
			return err
		}
		return c.unmarshal(bytes, &data)
	})
	if err != nil {
		return nil, mapBadgerError(err)
	}
	return &data, nil
}

// DeleteMetadata implements metastor.Client.DeleteMetadata
func (c *Client) DeleteMetadata(key []byte) error {
	if len(key) == 0 {
		return metastor.ErrNilKey
	}

	err := c.db.Update(func(txn *badgerdb.Txn) error {
		return txn.Delete(key)
	})
	if err != nil {
		return mapBadgerError(err)
	}
	return nil
}

// Close implements metastor.Client.Close
func (c *Client) Close() error {
	// cancel (db) context
	c.cancelFunc()

	// close db
	err := c.db.Close()
	if err != nil {
		return mapBadgerError(err)
	}

	return nil
}

// collectGarbage runs the garbage collection for Badger backend db
func (c *Client) collectGarbage() error {
	if err := c.db.PurgeOlderVersions(); err != nil {
		return err
	}
	return c.db.RunValueLogGC(discardRatio)
}

// runGC triggers the garbage collection for the Badger backend db.
// Should be run as a goroutine
func (c *Client) runGC() {
	ticker := time.NewTicker(cgInterval)
	for {
		select {
		case <-ticker.C:
			err := c.collectGarbage()
			if err != nil {
				// don't report error when gc didn't result in any cleanup
				if err == badgerdb.ErrNoRewrite {
					log.Debugf("Badger GC: %v", err)
				} else {
					log.Errorf("Badger GC failed: %v", err)
				}
			}
		case <-c.ctx.Done():
			return
		}
	}
}

// map badger errors, if we know about them
func mapBadgerError(err error) error {
	switch err {
	case badgerdb.ErrKeyNotFound:
		return metastor.ErrNotFound
	case badgerdb.ErrEmptyKey:
		return metastor.ErrNilKey
	default:
		return err
	}
}
