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
	"fmt"

	"github.com/threefoldtech/0-stor/client/datastor"
	"github.com/threefoldtech/0-stor/client/metastor/metatypes"

	log "github.com/sirupsen/logrus"
	"github.com/templexxx/reedsolomon"
	"golang.org/x/sync/errgroup"
)

// NewDistributedChunkStorage creates a new DistributedChunkStorage,
// using the given Cluster and default ReedSolomonEncoderDecoder as internal DistributedEncoderDecoder.
// See `DistributedChunkStorage` `DistributedEncoderDecoder` for more information.
func NewDistributedChunkStorage(cluster datastor.Cluster, dataShardCount, parityShardCount, jobCount int) (*DistributedChunkStorage, error) {
	if cluster.ListedShardCount() < dataShardCount+parityShardCount {
		return nil, errors.New("DistributedChunkStorage requires " +
			"at least dataShardCount+parityShardCount amount of listed datastor shards")
	}
	dec, err := NewReedSolomonEncoderDecoder(dataShardCount, parityShardCount)
	if err != nil {
		return nil, fmt.Errorf("failed to create DistributedChunkStorage: %v", err)
	}
	return NewDistributedChunkStorageWithEncoderDecoder(cluster, dec, jobCount), nil
}

// NewDistributedChunkStorageWithEncoderDecoder creates a new DistributedChunkStorage,
// using the given Cluster and DistributedEncoderDecoder.
// See `DistributedChunkStorage` `DistributedEncoderDecoder` for more information.
func NewDistributedChunkStorageWithEncoderDecoder(cluster datastor.Cluster, dec DistributedEncoderDecoder, jobCount int) *DistributedChunkStorage {
	if cluster == nil {
		panic("DistributedChunkStorage: no datastor cluster given")
	}
	if dec == nil {
		panic("DistributedChunkStorage: no DistributedEncoderDecoder given")
	}

	if jobCount < 1 {
		jobCount = DefaultJobCount
	}

	return &DistributedChunkStorage{
		cluster:  cluster,
		dec:      dec,
		jobCount: jobCount,
	}
}

// DistributedChunkStorage defines a storage implementation,
// which splits and distributes data over a secure amount of shards,
// rather than just writing it to a single shard as it is.
// This to provide protection against data loss when one of the used shards drops.
//
// By default the erasure code algorithms as implemented in
// the github.com/templexxx/reedsolomon library are used,
// and wrapped by the default ReedSolomonEncoderDecoder type.
// When using this default distributed encoder-decoder,
// you need to provide at least 2 shards (1 data- and 1 parity- shard).
//
// When creating a DistributedChunkStorage you can also pass in your
// own DistributedEncoderDecoder should you not be satisfied with the default implementation.
type DistributedChunkStorage struct {
	cluster  datastor.Cluster
	dec      DistributedEncoderDecoder
	jobCount int
}

// WriteChunk implements storage.ChunkStorage.WriteChunk
func (ds *DistributedChunkStorage) WriteChunk(data []byte) (*ChunkConfig, error) {
	parts, err := ds.dec.Encode(data)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	group, ctx := errgroup.WithContext(ctx)

	jobCount := ds.jobCount
	partsCount := len(parts)
	if jobCount > partsCount {
		jobCount = partsCount
	}

	// sends each part to an available worker goroutine,
	// which tries to store it in a random shard.
	// however make sure that we store the shard list,
	// in the same order as how we received the different parts,
	// otherwise we might not be able to decode it once again.
	type indexedPart struct {
		Index int
		Data  []byte
	}
	inputCh := make(chan indexedPart, jobCount)
	go func() {
		defer close(inputCh) // closes itself
		for index, part := range parts {
			select {
			case inputCh <- indexedPart{index, part}:
			case <-ctx.Done():
				return
			}
		}
	}()

	// create a channel-based iterator, to fetch the shards,
	// randomly and thread-save
	shardCh := datastor.ShardIteratorChannel(ctx,
		ds.cluster.GetRandomShardIterator(nil), jobCount)

	// write all the different parts to their own separate shard,
	// and return the written object information over the resultCh,
	// which will be used to collect all the successful shards' identifiers for the final output
	type indexedObject struct {
		Index  int
		Object metatypes.Object
	}
	resultCh := make(chan indexedObject, jobCount)
	// create all the actual workers
	for i := 0; i < jobCount; i++ {
		group.Go(func() error {
			var (
				key   []byte
				part  indexedPart
				open  bool
				err   error
				shard datastor.Shard
			)
			for {
				// wait for a part to write
				select {
				case part, open = <-inputCh:
					if !open {
						// channel is closed -> return
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
							"while fetching shard for a distribute-write request")
					}

					// do the actual storage
					key, err = shard.CreateObject(part.Data)
					if err == nil {
						object := metatypes.Object{Key: key, ShardID: shard.Identifier()}
						select {
						case resultCh <- indexedObject{part.Index, object}:
							break writeLoop
						case <-ctx.Done():
							return errors.New("context was unexpectedly cancelled, " +
								"while returning the identifier of a shard for a distribute-write request")
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
			log.WithError(err).Errorf("duplicate-writing has failed due to an error")
		}
		close(resultCh)
	}()

	// collect the identifiers of all shards we could write our object to,
	// and store+send them in the same order as how we received the parts
	var (
		resultCount int
		objects     = make([]metatypes.Object, partsCount)
	)
	// fetch all results
	for result := range resultCh {
		objects[result.Index] = result.Object
		resultCount++
	}

	cfg := ChunkConfig{Size: int64(len(data)), Objects: objects}
	// check if we have sufficient distributions
	if resultCount < partsCount {
		return &cfg, ErrShardsUnavailable
	}
	return &cfg, nil
}

// ReadChunk implements storage.ChunkStorage.ReadChunk
func (ds *DistributedChunkStorage) ReadChunk(cfg ChunkConfig) ([]byte, error) {
	return ds.readChunk(cfg, false)
}

func (ds *DistributedChunkStorage) readChunk(cfg ChunkConfig, checkStatus bool) ([]byte, error) {
	// validate the input object count
	objectCount := len(cfg.Objects)

	requiredObjectCount := ds.dec.RequiredShardCount()
	if requiredObjectCount != objectCount {
		return nil, ErrUnexpectedObjectCount
	}
	minimumShardCount := ds.dec.MinimumValidShardCount()

	// define the jobCount
	jobCount := ds.jobCount
	if jobCount > objectCount {
		jobCount = objectCount
	}

	// create our sync-purpose variables
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	group, ctx := errgroup.WithContext(ctx)

	// create a channel-based iterator, to get the objects,
	// in sequence as given, and thread-save.
	objectIndexCh := make(chan int, jobCount)
	go func() {
		defer close(objectIndexCh)
		for index := range cfg.Objects {
			select {
			case objectIndexCh <- index:
			case <-ctx.Done():
				return
			}
		}
	}()

	type readResult struct {
		Index int
		Data  []byte
	}

	// read all the needed parts,
	// from the available datashards
	resultCh := make(chan readResult, jobCount)
	// create all the actual workers
	for i := 0; i < jobCount; i++ {
		group.Go(func() error {
			var (
				open        bool
				object      *datastor.Object
				err         error
				shard       datastor.Shard
				index       int
				inputObject metatypes.Object
			)
			for {
				// fetch a random shard
				select {
				case index, open = <-objectIndexCh:
					if !open {
						return nil
					}
				case <-ctx.Done():
					return nil
				}
				inputObject = cfg.Objects[index]
				shard, err = ds.cluster.GetShard(inputObject.ShardID)
				if err != nil {
					// casually log the shard-get error,
					// and continue trying with another object...
					log.WithFields(log.Fields{
						"shard":  inputObject.ShardID,
						"object": inputObject.Key,
					}).WithError(err).Errorf("failed to get object")
					continue
				}

				if checkStatus {
					// check chunk status. Used for repair
					// we need to know if we can use this shard to reconstruct
					//  the file or not
					status, err := shard.GetObjectStatus(inputObject.Key)
					if err != nil {
						log.WithFields(log.Fields{
							"shard":  inputObject.ShardID,
							"object": inputObject.Key,
						}).WithError(err).Errorf("error while checking status of object")
						continue
					}
					if status != datastor.ObjectStatusOK {
						log.Debugf("object %q stored on shard %q is not valid: %s",
							inputObject.Key, inputObject.Key, status)
						continue
					}
				}

				// fetch the data part
				object, err = shard.GetObject(inputObject.Key)
				if err != nil {
					// casually log the shard-read error,
					// and continue trying with another shard...
					log.WithFields(log.Fields{
						"shard":  inputObject.ShardID,
						"object": inputObject.Key,
					}).WithError(err).Errorf("failed to read object")
					continue // try another shard
				}
				result := readResult{
					Index: index,
					Data:  object.Data,
				}
				select {
				case resultCh <- result:
				case <-ctx.Done():
					// this can be expected in case we reached the minimum shards needed
					return nil
				}
			}
		})
	}

	// close the result channel,
	// when all grouped goroutines are finished, so it can be used as an iterator
	go func() {
		err := group.Wait()
		if err != nil {
			log.WithError(err).Errorf("distribute-read has failed due to an error")
		}
		close(resultCh)
	}()

	// collect all the different distributed parts
	var (
		resultCount int

		parts = make([][]byte, requiredObjectCount)
	)

	for result := range resultCh {
		// put the part in the correct slot
		parts[result.Index] = result.Data
		resultCount++

		if resultCount == minimumShardCount {
			break
		}
	}

	// ensure that we have received all the different parts
	if resultCount < minimumShardCount {
		return nil, ErrShardsUnavailable
	}

	// decode the distributed data
	data, err := ds.dec.Decode(parts, cfg.Size)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) != cfg.Size {
		return nil, ErrInvalidDataSize
	}

	// return decoded object
	return data, nil
}

// CheckChunk implements storage.ChunkStorage.CheckChunk
func (ds *DistributedChunkStorage) CheckChunk(cfg ChunkConfig, fast bool) (CheckStatus, error) {
	// validate the input shard count
	objectCount := len(cfg.Objects)

	// validate that we have enough objects specified
	requiredObjectCount := ds.dec.RequiredShardCount()
	if requiredObjectCount != objectCount {
		return CheckStatusInvalid, ErrUnexpectedObjectCount
	}
	minimumValidObjectCount := ds.dec.MinimumValidShardCount()

	// define the target amount of valid objects to be searched for
	searchObjectCount := requiredObjectCount
	if fast {
		searchObjectCount = minimumValidObjectCount
	}

	// define the jobCount
	jobCount := ds.jobCount
	if jobCount > searchObjectCount {
		jobCount = searchObjectCount
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	group, ctx := errgroup.WithContext(ctx)

	// create a channel-based iterator, to get the objects,
	// in sequence as given, and thread-save.
	objectIndexCh := make(chan int, jobCount)
	go func() {
		defer close(objectIndexCh)
		for index := range cfg.Objects {
			select {
			case objectIndexCh <- index:
			case <-ctx.Done():
				return
			}
		}
	}()

	// request the worker goroutines,
	// to get exactly searchShardCount amount of valid shards to be found,
	// or less if that couldn't be achieved, but not more.
	requestCh := make(chan struct{}, jobCount)
	go func() {
		defer close(requestCh) // closes itself
		for i := searchObjectCount; i > 0; i-- {
			select {
			case requestCh <- struct{}{}:
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
	// create all the actual workers
	for i := 0; i < jobCount; i++ {
		group.Go(func() error {
			var (
				open   bool
				err    error
				status datastor.ObjectStatus
				shard  datastor.Shard
				index  int
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
				// or until we have confirmed a valid object
			validateLoop:
				for {
					// fetch a next available object
					select {
					case index, open = <-objectIndexCh:
						if !open {
							return nil
						}
					case <-ctx.Done():
						return nil
					}
					object = cfg.Objects[index]

					// first get the shard for that object, if possible
					shard, err = ds.cluster.GetShard(object.ShardID)
					if err != nil {
						log.WithFields(log.Fields{
							"shard":  object.ShardID,
							"object": object.Key,
						}).WithError(err).Errorf("error while fetching object")
						continue validateLoop
					}

					// validate if the object's status for this shard is OK
					status, err = shard.GetObjectStatus(object.Key)
					if err != nil {
						log.WithFields(log.Fields{
							"shard":  object.ShardID,
							"object": object.Key,
						}).WithError(err).Errorf("error while validating object")
						continue validateLoop
					}
					if status != datastor.ObjectStatusOK {
						log.Debugf("object %q stored on shard %q is not valid: %s",
							object.Key, object.Key, status)
						continue validateLoop
					}

					// shard is reachable and contains a valid object,
					// notify the result collector about it
					select {
					case resultCh <- struct{}{}:
						break validateLoop
					case <-ctx.Done():
						return nil
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
			log.WithError(err).Errorf("distribute-check has failed due to an error")
		}
		close(resultCh)
	}()

	// count how many shards are valid
	var validObjectCount int
	// fetch all results
	for range resultCh {
		validObjectCount++
	}

	// return the result
	if validObjectCount == requiredObjectCount {
		return CheckStatusOptimal, nil
	}
	if validObjectCount >= minimumValidObjectCount {
		return CheckStatusValid, nil
	}
	return CheckStatusInvalid, nil
}

// RepairChunk implements storage.ChunkStorage.RepairChunk
func (ds *DistributedChunkStorage) RepairChunk(cfg ChunkConfig) (*ChunkConfig, error) {
	obj, err := ds.readChunk(cfg, true)
	if err != nil {
		return nil, err
	}
	return ds.WriteChunk(obj)
}

// DeleteChunk implements storage.ChunkStorage.DeleteChunk
func (ds *DistributedChunkStorage) DeleteChunk(cfg ChunkConfig) error {
	objectLength := len(cfg.Objects)
	if objectLength == 0 {
		// if no objects are given, something is wrong
		return ErrUnexpectedObjectCount
	}

	if objectLength == 1 {
		// it will be weird if only 1 object is given,
		// but if so, we don't really want to spin any goroutines
		obj := &cfg.Objects[0]
		shard, err := ds.cluster.GetShard(obj.ShardID)
		if err != nil {
			return err
		}
		return shard.DeleteObject(obj.Key)
	}

	// limit our job count,
	// in case we don't have that many objects to delete
	jobCount := ds.jobCount
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
				shard, err = ds.cluster.GetShard(obj.ShardID)
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
func (ds *DistributedChunkStorage) Close() error {
	return ds.cluster.Close()
}

// DistributedEncoderDecoder is the type used internally to
// read and write the data of objects, read and written using the DistributedChunkStorage.
type DistributedEncoderDecoder interface {
	// Encode object data into multiple (distributed) parts,
	// such that those parts can be reconstructed when the data has to be read again.
	Encode(data []byte) (parts [][]byte, err error)
	// Decode the different parts back into the original data slice,
	// as it was given in the original Encode call.
	Decode(parts [][]byte, dataSize int64) (data []byte, err error)

	// MinimumValidShardCount returns the minimum valid shard count required,
	// in order to decode a distributed object.
	MinimumValidShardCount() int

	// RequiredShardCount returns the shard count which is expected.
	// Meaning that the parts given to the Decode method will have to be exactly the number
	// returned by ths method, or else that method will fail.
	RequiredShardCount() int
}

// NewReedSolomonEncoderDecoder creates a new ReedSolomonEncoderDecoder.
// See `ReedSolomonEncoderDecoder` for more information.
func NewReedSolomonEncoderDecoder(dataShardCount, parityShardCount int) (*ReedSolomonEncoderDecoder, error) {
	if dataShardCount < 1 {
		return nil, errors.New("dataShardCount has to be at least 1")
	}
	if parityShardCount < 1 {
		return nil, errors.New("parityShardCount has to be at least 1")
	}

	er, err := reedsolomon.New(dataShardCount, parityShardCount)
	if err != nil {
		return nil, err
	}
	return &ReedSolomonEncoderDecoder{
		dataShardCount:   dataShardCount,
		parityShardCount: parityShardCount,
		shardCount:       dataShardCount + parityShardCount,
		er:               er,
	}, nil
}

// ReedSolomonEncoderDecoder implements the DistributedEncoderDecoder,
// using the erasure encoding library github.com/templexxx/reedsolomon.
//
// This implementation is also used as the default DistributedEncoderDecoder
// for the DistributedChunkStorage storage type.
type ReedSolomonEncoderDecoder struct {
	dataShardCount, parityShardCount int                         // data and parity count
	shardCount                       int                         // dataShardCount + parityShardCount
	er                               reedsolomon.EncodeReconster // encoder + decoder
}

// Encode implements DistributedEncoderDecoder.Encode
func (rs *ReedSolomonEncoderDecoder) Encode(data []byte) ([][]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("no data given to encode")
	}

	parts := rs.splitData(data)
	parities := reedsolomon.NewMatrix(rs.parityShardCount, len(parts[0]))
	parts = append(parts, parities...)
	err := rs.er.Encode(parts)
	return parts, err
}

// Decode implements DistributedEncoderDecoder.Decode
func (rs *ReedSolomonEncoderDecoder) Decode(parts [][]byte, dataSize int64) ([]byte, error) {
	if len(parts) != rs.shardCount {
		return nil, errors.New("unexpected amount of parts given to decode")
	}

	if err := rs.er.ReconstructData(parts); err != nil {
		return nil, err
	}

	var (
		data   = make([]byte, dataSize)
		offset int64
	)
	for i := 0; i < rs.dataShardCount; i++ {
		copy(data[offset:], parts[i])
		offset += int64(len(parts[i]))
		if offset >= dataSize {
			break
		}
	}
	return data, nil
}

// MinimumValidShardCount implements DistributedEncoderDecoder.MinimumValidShardCount
func (rs *ReedSolomonEncoderDecoder) MinimumValidShardCount() int {
	return rs.dataShardCount
}

// RequiredShardCount implements DistributedEncoderDecoder.RequiredShardCount
func (rs *ReedSolomonEncoderDecoder) RequiredShardCount() int {
	return rs.shardCount
}

func (rs *ReedSolomonEncoderDecoder) splitData(data []byte) [][]byte {
	data = rs.padIfNeeded(data)
	chunkSize := len(data) / rs.dataShardCount
	chunks := make([][]byte, rs.dataShardCount)

	for i := 0; i < rs.dataShardCount; i++ {
		chunks[i] = data[i*chunkSize : (i+1)*chunkSize]
	}
	return chunks
}

func (rs *ReedSolomonEncoderDecoder) padIfNeeded(data []byte) []byte {
	padLen := rs.getPadLen(len(data))
	if padLen == 0 {
		return data
	}

	pad := make([]byte, padLen)
	return append(data, pad...)
}

func (rs *ReedSolomonEncoderDecoder) getPadLen(dataLen int) int {
	const padFactor = 256
	maxPadLen := rs.dataShardCount * padFactor
	mod := dataLen % maxPadLen
	if mod == 0 {
		return 0
	}
	return maxPadLen - mod
}

var (
	_ ChunkStorage = (*DistributedChunkStorage)(nil)

	_ DistributedEncoderDecoder = (*ReedSolomonEncoderDecoder)(nil)
)
