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
	"context"
	"errors"
	"sync"

	"github.com/threefoldtech/0-stor/client/datastor"
	"github.com/threefoldtech/0-stor/client/metastor/metatypes"

	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

// NewReplicatedChunkStorage creates a new ReplicatedChunkStorage.
// See `ReplicatedChunkStorage` for more information.
//
// jobCount is optional and can be `<= 0` in order to use DefaultJobCount.
func NewReplicatedChunkStorage(cluster datastor.Cluster, dataShardCount, jobCount int) (*ReplicatedChunkStorage, error) {
	if cluster == nil {
		panic("ReplicatedChunkStorage: no datastor cluster given")
	}
	if dataShardCount < 1 {
		panic("ReplicatedChunkStorage: dataShardCount has to be at least 1")
	}

	if cluster.ListedShardCount() < dataShardCount {
		return nil, errors.New("ReplicatedChunkStorage requires " +
			"at least dataShardCount amount of listed datastor shards")
	}

	if jobCount < 1 {
		jobCount = DefaultJobCount
	}
	writeJobCount := jobCount
	if writeJobCount < dataShardCount {
		writeJobCount = dataShardCount
	}

	return &ReplicatedChunkStorage{
		cluster:        cluster,
		dataShardCount: dataShardCount,
		jobCount:       jobCount,
		writeJobCount:  writeJobCount,
	}, nil
}

// ReplicatedChunkStorage defines a storage implementation,
// which writes an object to multiple shards at once,
// the amount of shards which is defined by the used dataShardCount.
//
// For reading it will try to a multitude of the possible shards at once,
// and return the object that it received first. As it is expected that all
// shards return the same object for this key, when making use of this storage,
// there is no need to read from all shards and wait for all of those results as well.
//
// Repairing is done by first assembling a list of corrupt, OK and dead shards.
// Once that's done, the corrupt shards will be simply tried to be written to again,
// while the dead shards will be attempted to be replaced, if possible.
type ReplicatedChunkStorage struct {
	cluster                 datastor.Cluster
	dataShardCount          int
	jobCount, writeJobCount int
}

// WriteChunk implements storage.ChunkStorage.WriteChunk
func (rs *ReplicatedChunkStorage) WriteChunk(data []byte) (*ChunkConfig, error) {
	return rs.write(nil, rs.dataShardCount, data)
}

// ReadChunk implements storage.ChunkStorage.ReadChunk
func (rs *ReplicatedChunkStorage) ReadChunk(cfg ChunkConfig) ([]byte, error) {
	// ensure that at least 1 shard is given
	if len(cfg.Objects) == 0 {
		return nil, ErrUnexpectedObjectCount
	}

	var (
		err    error
		object *datastor.Object
		shard  datastor.Shard
	)
	// simply try to read sequentially until one could be read,
	// as we should in most scenarios only ever have to read from 1 (and 2 or 3 in bad situations),
	// it would be bad for performance to try to read from multiple goroutines and shards for all calls.
	for _, obj := range cfg.Objects {
		shard, err = rs.cluster.GetShard(obj.ShardID)
		if err != nil {
			log.Errorf("failed to get shard %q for object %q: %v",
				obj.Key, obj.ShardID, err)
			continue
		}

		object, err = shard.GetObject(obj.Key)
		if err != nil {
			log.Errorf("failed to read %q from replicated shard %q: %v",
				obj.Key, obj.ShardID, err)
			continue
		}

		if int64(len(object.Data)) == cfg.Size {
			return object.Data, nil
		}
		log.Errorf("failed to read %q from replicated shard %q: invalid data size",
			obj.Key, obj.ShardID)
	}

	// sadly, no shard was available
	log.Error("failed replicate-read chunk from any of the configured shards")
	return nil, ErrShardsUnavailable
}

// CheckChunk implements storage.ChunkStorage.CheckChunk
func (rs *ReplicatedChunkStorage) CheckChunk(cfg ChunkConfig, fast bool) (CheckStatus, error) {
	objectCount := len(cfg.Objects)
	if objectCount == 0 {
		return CheckStatusInvalid, ErrUnexpectedObjectCount
	}

	// define the jobCount
	jobCount := rs.jobCount
	if jobCount > objectCount {
		jobCount = objectCount
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// create a channel-based fetcher to get all the objects
	objectIndexCh := make(chan int, jobCount)
	go func() {
		defer close(objectIndexCh)
		for i := range cfg.Objects {
			select {
			case objectIndexCh <- i:
			case <-ctx.Done():
				return
			}
		}
	}()

	// each worker will help us get through all shards,
	// until we found the desired amount of valid shards,
	// the maximum which is helped guarantee by the requestCh iterator,
	// while the minimum is defined by that same channel or by exhausting the shardCh.
	resultCh := make(chan struct{}, jobCount)

	// create our goroutine,
	// to close our resultCh in case we have exhausted our worker goroutines
	var wg sync.WaitGroup
	wg.Add(jobCount)
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// create all the actual workers
	for i := 0; i < jobCount; i++ {
		go func() {
			defer wg.Done()

			var (
				open   bool
				err    error
				status datastor.ObjectStatus
				index  int
				object metatypes.Object
				shard  datastor.Shard
			)

			for {
				// fetch a next available object
				select {
				case index, open = <-objectIndexCh:
					if !open {
						return
					}
				case <-ctx.Done():
					return
				}
				object = cfg.Objects[index]

				// first get the shard for that object, if possible
				shard, err = rs.cluster.GetShard(object.ShardID)
				if err != nil {
					log.Errorf("error while fetching shard %q for object %q: %v",
						object.ShardID, object.Key, err)
					continue
				}

				// validate if the object's status for this shard is OK
				status, err = shard.GetObjectStatus(object.Key)
				if err != nil {
					log.Errorf("error while validating %q stored on shard %q: %v",
						object.Key, object.ShardID, err)
					continue
				}
				if status != datastor.ObjectStatusOK {
					log.Debugf("object %q stored on shard %q is not valid: %s",
						object.Key, object.ShardID, status)
					continue
				}

				// shard is valid for this object,
				// notify the result collector about it
				select {
				case resultCh <- struct{}{}:
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	// if we want a fast result,
	// we simply want to know that at least one is available
	if fast {
		select {
		case _, open := <-resultCh:
			if !open {
				return CheckStatusInvalid, nil
			}
			if rs.dataShardCount == 1 {
				return CheckStatusOptimal, nil
			}
			return CheckStatusValid, nil
		case <-ctx.Done():
			return CheckStatusInvalid, nil
		}
	}

	// otherwise we'll go through all of them,
	// until we have a max of nrReplication results
	var validObjectCount int
	for range resultCh {
		validObjectCount++
		if validObjectCount == rs.dataShardCount {
			return CheckStatusOptimal, nil
		}
	}

	if validObjectCount > 0 {
		return CheckStatusValid, nil
	}
	return CheckStatusInvalid, nil
}

// RepairChunk implements storage.ChunkStorage.RepairChunk
func (rs *ReplicatedChunkStorage) RepairChunk(cfg ChunkConfig) (*ChunkConfig, error) {
	objectCount := len(cfg.Objects)
	if objectCount == 0 {
		// we can't do anything if no shards are given
		return nil, ErrUnexpectedObjectCount
	}
	if objectCount == 1 {
		// we can't repair, but if that only shard is valid,
		// we can at least return the same config
		shard, err := rs.cluster.GetShard(cfg.Objects[0].ShardID)
		if err != nil {
			return nil, ErrShardsUnavailable
		}
		status, err := shard.GetObjectStatus(cfg.Objects[0].Key)
		if err != nil || status != datastor.ObjectStatusOK {
			return nil, ErrShardsUnavailable
		}
		// simply return the same config
		return &cfg, nil
	}

	// first, let's collect all valid and invalid shards in 2 separate slices
	validObjects, invalidObjects := rs.splitObjects(cfg.Objects)
	// NOTE: len(validObjects)+len(invalidObjects) < len(cfg.Objects)
	//       is valid, and is the scenario possible when some shards
	//       returned an error and thus indicated they were actually non functional
	objectCount = len(validObjects)
	if objectCount == 0 {
		return nil, ErrShardsUnavailable
	}
	if objectCount >= rs.dataShardCount {
		// if our validShard count is already good enough, we can quit
		return &ChunkConfig{
			Size:    cfg.Size,
			Objects: validObjects,
		}, nil
	}

	// read the object from the first available shard
	var (
		err    error
		object *datastor.Object
		shard  datastor.Shard
	)
	// simply try to read sequentially until one could be read,
	// as we should in most scenarios only ever have to read from 1 (and 2 or 3 in bad situations),
	// it would be bad for performance to try to read from multiple goroutines and shards for all calls.
	for _, obj := range validObjects {
		shard, err = rs.cluster.GetShard(obj.ShardID)
		if err != nil {
			log.Errorf("failed to get shard %q for object %q: %v",
				obj.Key, obj.ShardID, err)
			validObjects = validObjects[1:]
			continue
		}

		object, err = shard.GetObject(obj.Key)
		if err != nil {
			log.Errorf("failed to read %q from replicated shard %q: %v",
				obj.Key, obj.ShardID, err)
			validObjects = validObjects[1:]
			continue
		}
		if int64(len(object.Data)) != cfg.Size {
			log.Errorf("failed to read %q from replicated shard %q: invalid data size",
				obj.Key, obj.ShardID)
			validObjects = validObjects[1:]
			continue
		}

		// object is considered valid
		break
	}
	if object == nil {
		log.Error("no valid object could be found to replicate-repair from")
		return nil, ErrShardsUnavailable
	}
	objectCount = len(validObjects)

	// write to our non-used shards
	exceptShards := make([]string, 0, len(invalidObjects)+objectCount)
	for _, obj := range invalidObjects {
		exceptShards = append(exceptShards, obj.ShardID)
	}
	for _, obj := range validObjects {
		exceptShards = append(exceptShards, obj.ShardID)
	}
	outputCfg, err := rs.write(exceptShards, rs.dataShardCount-objectCount, object.Data)
	if err != nil {
		return outputCfg, err
	}

	// add our shards to our output cfg
	// and return it if we have at least dataShardCount amount of shards
	outputCfg.Objects = append(validObjects, outputCfg.Objects...)
	if len(cfg.Objects) < rs.dataShardCount {
		return outputCfg, ErrShardsUnavailable
	}
	return outputCfg, nil
}

// splitObjects is a private utility method,
// to help us split the given objects into valid and invalid ones.
func (rs *ReplicatedChunkStorage) splitObjects(allObjects []metatypes.Object) (validObjects []metatypes.Object, invalidObjects []metatypes.Object) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// create a channel-based fetcher to get all the objects
	objectIndexCh := make(chan int, rs.jobCount)
	go func() {
		defer close(objectIndexCh)
		for i := range allObjects {
			select {
			case objectIndexCh <- i:
			case <-ctx.Done():
				return
			}
		}
	}()

	// each worker will continue to fetch objects from shards, while there are objects left to be checked,
	// and for each fetched object it will check and indicate whether or not
	// the object is valid
	type checkResult struct {
		Index int
		Valid bool // valid true means the shard is valid
		// !Valid && Index >= 0
		// means neither invalid nor valid,
		// we simply should give it another try
	}
	resultCh := make(chan checkResult, rs.jobCount)

	// create our goroutine,
	// to close our resultCh in case we have exhausted our worker goroutines
	var wg sync.WaitGroup
	wg.Add(rs.jobCount)
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// create all the actual workers
	for i := 0; i < rs.jobCount; i++ {
		go func() {
			defer wg.Done()

			var (
				open   bool
				err    error
				status datastor.ObjectStatus
				index  int
				object metatypes.Object
				shard  datastor.Shard
			)

			for {
				index, open = <-objectIndexCh
				if !open {
					return
				}
				object = allObjects[index]

				// first get the shard for that object, if possible
				shard, err = rs.cluster.GetShard(object.ShardID)
				if err != nil {
					log.Errorf("error while fetching shard %q for object %q: %v",
						object.ShardID, object.Key, err)
					continue
				}

				var result checkResult

				status, err = shard.GetObjectStatus(object.Key)
				if err == nil {
					if status == datastor.ObjectStatusOK {
						result.Valid = true
						result.Index = index
					} else {
						result.Index = -1
					}
				} else {
					log.Errorf("error while validating %q stored on shard %q: %v",
						object.Key, object.ShardID, err)
					result.Index = index
				}

				// shard is valid for this object,
				// notify the result collector about it
				resultCh <- result
			}
		}()
	}

	// collect all results
	for result := range resultCh {
		if result.Valid {
			validObjects = append(validObjects, allObjects[result.Index])
		} else if result.Index >= 0 {
			invalidObjects = append(invalidObjects, allObjects[result.Index])
		}
	}
	return
}

func (rs *ReplicatedChunkStorage) write(exceptShards []string, dataShardCount int, data []byte) (*ChunkConfig, error) {
	group, ctx := errgroup.WithContext(context.Background())

	jobCount := rs.jobCount
	if jobCount > dataShardCount {
		jobCount = dataShardCount
	}

	// request the worker goroutines,
	// to get exactly dataShardCount amount of replications.
	requestCh := make(chan struct{}, jobCount)
	go func() {
		defer close(requestCh) // closes itself
		for i := dataShardCount; i > 0; i-- {
			select {
			case requestCh <- struct{}{}:
			case <-ctx.Done():
				return
			}
		}
	}()

	// create a channel-based iterator, to fetch the shards,
	// randomly and thread-save
	shardCh := datastor.ShardIteratorChannel(ctx,
		rs.cluster.GetRandomShardIterator(exceptShards), jobCount)

	// write to dataShardCount amount of shards,
	// and return their identifiers over the resultCh,
	// collection all the successful shards' identifiers for the final output
	resultCh := make(chan metatypes.Object, jobCount)
	// create all the actual workers
	for i := 0; i < jobCount; i++ {
		group.Go(func() error {
			var (
				open   bool
				err    error
				shard  datastor.Shard
				object metatypes.Object
			)
			for {
				// wait for a request
				select {
				case _, open = <-requestCh:
					if !open {
						// fake request: channel is closed -> return
						return nil
					}
				case <-ctx.Done():
					return nil
				}

				// loop here, until we either have an error,
				// or until we have written to a shard
			writeLoop:
				for {
					// fetch a random shard,
					// it's an error if this is not possible,
					// as a shard is expected to be still available at this stage
					select {
					case shard, open = <-shardCh:
						if !open {
							// not enough shards are available,
							// we know this because the iterator ch has already been closed
							return ErrShardsUnavailable
						}
					case <-ctx.Done():
						return errors.New("context was unexpectedly cancelled, " +
							"while fetching shard for a replicate-write request")
					}

					// do the actual storage
					object.Key, err = shard.CreateObject(data)
					if err == nil {
						object.ShardID = shard.Identifier()
						select {
						case resultCh <- object:
							break writeLoop
						case <-ctx.Done():
							return errors.New("context was unexpectedly cancelled, " +
								"while returning the identifier of a shard for a replicate-write request")
						}
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
			}
		})
	}

	// close the result channel,
	// when all grouped goroutines are finished, so it can be used as an iterator
	go func() {
		err := group.Wait()
		if err != nil {
			log.Errorf("replicate-writinghas failed due to an error: %v", err)
		}
		close(resultCh)
	}()

	cfg := &ChunkConfig{Size: int64(len(data))}
	// collect the identifiers of all shards, we could write our object to
	cfg.Objects = make([]metatypes.Object, 0, rs.dataShardCount)
	// fetch all results
	for object := range resultCh {
		cfg.Objects = append(cfg.Objects, object)
	}

	// check if we have sufficient replications
	if len(cfg.Objects) < dataShardCount {
		return cfg, ErrShardsUnavailable
	}
	return cfg, nil
}

// DeleteChunk implements storage.ChunkStorage.DeleteChunk
func (rs *ReplicatedChunkStorage) DeleteChunk(cfg ChunkConfig) error {
	objectLength := len(cfg.Objects)
	if objectLength == 0 {
		// if no objects are given, something is wrong
		return ErrUnexpectedObjectCount
	}

	if objectLength == 1 {
		// it will be weird if only 1 object is given,
		// but if so, we don't really want to spin any goroutines
		obj := &cfg.Objects[0]
		shard, err := rs.cluster.GetShard(obj.ShardID)
		if err != nil {
			return err
		}
		return shard.DeleteObject(obj.Key)
	}

	// limit our job count,
	// in case we don't have that many objects to delete
	jobCount := rs.jobCount
	if jobCount > objectLength {
		jobCount = objectLength
	}

	// create an errgroup for all our delete jobs
	group, ctx := errgroup.WithContext(context.Background())

	// spawn our object fetcher
	indexCh := make(chan int, jobCount)
	go func() {
		defer close(indexCh)
		for i := range cfg.Objects {
			select {
			case indexCh <- i:
			case <-ctx.Done():
				return
			}
		}
	}()

	// spawn all our delete jobs
	for i := 0; i < jobCount; i++ {
		group.Go(func() error {
			var (
				err   error
				obj   *metatypes.Object
				shard datastor.Shard
			)
			for index := range indexCh {
				obj = &cfg.Objects[index]
				shard, err = rs.cluster.GetShard(obj.ShardID)
				if err != nil {
					return err
				}
				err = shard.DeleteObject(obj.Key)
				if err != nil {
					return err
				}
			}
			return nil
		})
	}

	// simply wait for all jobs to finish,
	// and return its (nil) error
	return group.Wait()
}

// Close implements ChunkStorage.Close
func (rs *ReplicatedChunkStorage) Close() error {
	return rs.cluster.Close()
}

var (
	_ ChunkStorage = (*ReplicatedChunkStorage)(nil)
)
