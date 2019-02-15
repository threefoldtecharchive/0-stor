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

package storage

import (
	"errors"

	"github.com/threefoldtech/0-stor/client/datastor"
	"github.com/threefoldtech/0-stor/client/metastor/metatypes"

	log "github.com/sirupsen/logrus"
)

// NewRandomChunkStorage creates a new RandomChunkStorage.
// See `RandomChunkStorage` for more information.
func NewRandomChunkStorage(cluster datastor.Cluster) (*RandomChunkStorage, error) {
	if cluster == nil {
		panic("no cluster given")
	}
	if cluster.ListedShardCount() < 1 {
		return nil, errors.New("RandomChunkStorage: at least one listed shard is required")
	}
	return &RandomChunkStorage{cluster: cluster}, nil
}

// RandomChunkStorage is the most simplest Storage implementation.
// For writing it only writes to one shard, randomly chosen.
// For reading it expects just, and only, one shard, to read from.
// Repairing is not supported for this storage for obvious reasons.
type RandomChunkStorage struct {
	cluster datastor.Cluster
}

// WriteChunk implements storage.ChunkStorage.WriteChunk
func (rs *RandomChunkStorage) WriteChunk(data []byte) (*ChunkConfig, error) {
	var (
		key   []byte
		err   error
		shard datastor.Shard
	)

	// go through all shards, in pseudo-random fashion,
	// until the data could be written to one of them.
	it := rs.cluster.GetRandomShardIterator(nil)
	for it.Next() {
		shard = it.Shard()
		key, err = shard.CreateObject(data)
		if err == nil {
			return &ChunkConfig{
				Size: int64(len(data)),
				Objects: []metatypes.Object{
					{
						Key:     key,
						ShardID: shard.Identifier(),
					},
				},
			}, nil
		}

		// check if the error is because the namespace if full
		// if it is, we don't log the error.
		if err == datastor.ErrNamespaceFull {
			log.WithField("shard", shard.Identifier()).Warningf("%v", err)
		} else {
			// if this is another error, we casually log the shard-write error,
			// and continue trying with another shard...
			log.WithField("shard", shard.Identifier()).WithError(err).Errorf("failed to write data to random shard")
		}
	}
	return nil, ErrShardsUnavailable
}

// ReadChunk implements storage.ChunkStorage.ReadChunk
func (rs *RandomChunkStorage) ReadChunk(cfg ChunkConfig) ([]byte, error) {
	if len(cfg.Objects) != 1 {
		return nil, ErrUnexpectedObjectCount
	}
	obj := &cfg.Objects[0]

	shard, err := rs.cluster.GetShard(obj.ShardID)
	if err != nil {
		return nil, err
	}

	object, err := shard.GetObject(obj.Key)
	if err != nil {
		return nil, err
	}

	if int64(len(object.Data)) != cfg.Size {
		return object.Data, ErrInvalidDataSize
	}
	return object.Data, nil
}

// CheckChunk implements storage.ChunkStorage.CheckChunk
func (rs *RandomChunkStorage) CheckChunk(cfg ChunkConfig, fast bool) (CheckStatus, error) {
	if len(cfg.Objects) != 1 {
		return CheckStatusInvalid, ErrUnexpectedObjectCount
	}
	obj := &cfg.Objects[0]

	shard, err := rs.cluster.GetShard(obj.ShardID)
	if err != nil {
		return CheckStatusInvalid, nil
	}

	status, err := shard.GetObjectStatus(obj.Key)
	if err != nil || status != datastor.ObjectStatusOK {
		return CheckStatusInvalid, nil
	}

	return CheckStatusOptimal, nil
}

// RepairChunk implements storage.ChunkStorage.RepairChunk
func (rs *RandomChunkStorage) RepairChunk(cfg ChunkConfig) (*ChunkConfig, error) {
	return nil, ErrNotSupported
}

// DeleteChunk implements storage.ChunkStorage.DeleteChunk
func (rs *RandomChunkStorage) DeleteChunk(cfg ChunkConfig) error {
	if len(cfg.Objects) != 1 {
		return ErrUnexpectedObjectCount
	}
	obj := &cfg.Objects[0]
	shard, err := rs.cluster.GetShard(obj.ShardID)
	if err != nil {
		return err
	}
	return shard.DeleteObject(obj.Key)
}

// Close implements ChunkStorage.Close
func (rs *RandomChunkStorage) Close() error {
	return rs.cluster.Close()
}

var (
	_ ChunkStorage = (*RandomChunkStorage)(nil)
)
