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

package etcd

import (
	"context"
	"errors"
	"fmt"
	"time"

	dbp "github.com/zero-os/0-stor/client/metastor/db"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/clientv3/concurrency"
)

// New creates new metastor database client, using an ETCD cluster as storage medium.
func New(endpoints []string) (*DB, error) {
	if len(endpoints) == 0 {
		return nil, errors.New("no endpoints given")
	}

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: metaOpTimeout,
	})
	if err != nil {
		return nil, err
	}
	return &DB{
		etcdClient: cli,
		ctx:        context.Background(),
	}, nil
}

// DB defines a metastor database,
// in the form of a ETCD (cluster) client,
// as to store and load processed metadata to/from an ETCD cluster.
type DB struct {
	etcdClient *clientv3.Client
	ctx        context.Context
}

// Set implements db.Set
func (db *DB) Set(key, metadata []byte) error {
	ctx, cancel := context.WithTimeout(db.ctx, metaOpTimeout)
	defer cancel()

	_, err := db.etcdClient.Put(ctx, string(key), string(metadata))
	return err
}

// Get implements db.Get
func (db *DB) Get(key []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(db.ctx, metaOpTimeout)
	defer cancel()

	resp, err := db.etcdClient.Get(ctx, string(key))
	if err != nil {
		return nil, err
	}
	if len(resp.Kvs) < 1 {
		return nil, dbp.ErrNotFound
	}
	return resp.Kvs[0].Value, nil
}

// Delete implements db.Delete
func (db *DB) Delete(key []byte) error {
	ctx, cancel := context.WithTimeout(db.ctx, metaOpTimeout)
	defer cancel()

	_, err := db.etcdClient.Delete(ctx, string(key))
	return err
}

// Update implements db.Update
func (db *DB) Update(key []byte, cb dbp.UpdateCallback) error {
	ctx, cancel := context.WithTimeout(db.ctx, metaOpTimeout)
	defer cancel()

	keyStr := string(key)
	resp, err := concurrency.NewSTM(db.etcdClient, func(stm concurrency.STM) error {
		// get the metadata
		metadataIn := stm.Get(keyStr)
		if len(metadataIn) == 0 {
			return dbp.ErrNotFound
		}

		// update the metadata
		metadataOut, err := cb([]byte(metadataIn))
		if err != nil {
			return err
		}
		// store the metadata
		stm.Put(keyStr, string(metadataOut))
		return nil
	}, concurrency.WithPrefetch(keyStr), concurrency.WithAbortContext(ctx))
	if err != nil {
		return err
	}
	if !resp.Succeeded {
		return fmt.Errorf("metadata update of '%s' didn't succeed", key)
	}
	return nil
}

// Close implements db.Close
func (db *DB) Close() error {
	return db.etcdClient.Close()
}

const (
	metaOpTimeout = 30 * time.Second
)

var (
	_ dbp.DB = (*DB)(nil)
)
