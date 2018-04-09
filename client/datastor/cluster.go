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

package datastor

import (
	"context"
	"crypto/rand"
	"errors"
	"math/big"
	mathRand "math/rand"

	log "github.com/sirupsen/logrus"
)

var (
	// ErrNoShardsAvailable is returned in case no shards are available in a cluster,
	// e.g. after filtering them with an exception list of non-desired shards.
	ErrNoShardsAvailable = errors.New("no shards available")
)

// Cluster can be used to group a bunch of zstordb servers
// together, so to speak, and be able to get a client
// for each of them. What interface is used for this,
// is up to the specific Cluster implementation.
type Cluster interface {
	// GetClient returns a client for the zstordb server
	// at the given (shard) address.
	// The given shard does not have to be part of
	// the predefined shard list, which makes up this cluster,
	// it can be any address, as long as it points to
	// a valid and supported zstordb server.
	GetShard(id string) (Shard, error)

	// GetRandomClient gets any cluster available in this cluster:
	// it only ever returns a client created from a shard
	// which comes from the pre-defined shard-list
	// (given at creation time of this cluster):
	GetRandomShard() (Shard, error)

	// GetRandomShardIterator can be used to get an iterator (channel),
	// which will give you a random shard, until either ll shards have been exhausted,
	// or the given context has been cancelled. It is guaranteed by the implementation,
	// that each returned shard, from the same random shard iterator,
	// hasn't been returned before by that iterator (channel).
	//
	// exceptShards is an optional input parameter,
	// when given, the iterator won't return the listed shards
	// which are part of that exceptShards slice.
	GetRandomShardIterator(exceptShards []string) ShardIterator

	// ListedShardCount returns the amount of listed shards available in this cluster.
	ListedShardCount() int

	// Close any open resources.
	Close() error
}

// ShardIterator defines the interface of an iterator which can be used
// to get different (random) shards, without ever getting the same shard back.
//
// ShardIterator is /NOT/ thread-safe,
// should you want to use it on multiple goroutines,
// wrap the iterator using the function `ShardIteratorChannel`,
// and use that function instead of this iterator directly.
type ShardIterator interface {
	// Next moves the iterator to the next available shard
	// if this is not possible it will return false
	// This function has to be called prior to the first Shard call.
	Next() bool

	// Shard returns the current shard, you can call this function as much as you want.
	// The function will return another Shard or become invalid,
	// after you call Next again.
	Shard() Shard
}

// Shard defines the interface of a cluster shard.
// It adds some functionality on top of a normal client,
// to make it work within a cluster.
type Shard interface {
	Client

	// Identifier returns the (unique) identifier of this shard.
	Identifier() string
}

// ShardIteratorChannel takes a context and an iterator,
// and allows you to turn the given iterator into a thread-safe iterator using channels.
// Due to the impact on the performance the channel-based approach brings with it,
// it isn't recommended unless you really need it.
func ShardIteratorChannel(ctx context.Context, iterator ShardIterator, bufferSize int) <-chan Shard {
	if ctx == nil {
		panic("no context given")
	}
	if iterator == nil {
		panic("no shard iterator given")
	}
	if bufferSize < 1 {
		log.Debug("ShardIteratorChannel wasn't passed a valid bufferSize, defaulting to 1")
		bufferSize = 1
	}

	ch := make(chan Shard, bufferSize)
	go func() {
		defer close(ch)
		for iterator.Next() {
			select {
			case ch <- iterator.Shard():
			case <-ctx.Done():
				return
			}
		}
	}()
	return ch
}

// NewLazyShardIterator creates a new LazyShardIterator.
// All input parameters are required, if either of them is empty/nil the function will panic.
// See `LazyShardIterator` for more information about this iterator type.
func NewLazyShardIterator(cluster Cluster, shards []string) *LazyShardIterator {
	if cluster == nil {
		panic("no cluster given to get shards from")
	}
	length := len(shards)
	if length == 0 {
		panic("no shards given to iterate on")
	}
	return &LazyShardIterator{
		cluster:  cluster,
		shards:   shards,
		curShard: nil,
		curIndex: -1,
		maxIndex: length - 1,
	}
}

// LazyShardIterator can be used to create an iterator,
// using a cluster and some amount of shards.
// It's lazy in that it only gets the next available shard from cluster,
// when the Next function is called, rather than pre-loading all of the potential shards at once.
type LazyShardIterator struct {
	cluster            Cluster
	shards             []string
	curShard           Shard
	curIndex, maxIndex int
}

// Next implements ShardIterator.Next
func (it *LazyShardIterator) Next() bool {
	var (
		err   error
		shard string
	)
	for it.curIndex < it.maxIndex {
		it.curIndex++
		shard = it.shards[it.curIndex]
		it.curShard, err = it.cluster.GetShard(shard)
		if err == nil {
			return true
		}
		log.Errorf("LazyShardIterator: error while getting shard %q: %v", shard, err)
	}
	return false
}

// Shard implements ShardIterator.Shard
func (it *LazyShardIterator) Shard() Shard {
	if it.curShard == nil {
		panic("invalid LazyShardIterator, ensure to make a successful Next() call first")
	}
	return it.curShard
}

// NewRandomShardIterator creates a new random shard Iterator.
// See `RandomShardIterator` for more information.
func NewRandomShardIterator(slice []Shard) *RandomShardIterator {
	return &RandomShardIterator{
		slice:   slice,
		length:  int64(len(slice)),
		current: nil,
	}
}

// RandomShardIterator implements the ShardIterator interface,
// in order to get a unique pseudo-random GRPC-interfaced datastor client for each iteration.
// The iterator is finished when all clients of the cluster have been exhausted.
type RandomShardIterator struct {
	slice   []Shard
	length  int64
	current Shard
}

// Next implements ShardIterator.Next
func (it *RandomShardIterator) Next() bool {
	if it.length < 1 {
		return false
	}

	index := RandShardIndex(it.length)
	it.length--
	it.slice[index], it.slice[it.length] = it.slice[it.length], it.slice[index]
	it.current = it.slice[it.length]
	return true
}

// Shard implements ShardIterator.Shard
func (it *RandomShardIterator) Shard() Shard {
	if it.current == nil {
		panic("invalid shard iterator, ensure to make a successful Next() call first")
	}
	return it.current
}

// RandShardIndex generates a (pseudo) random shard index in the range of [0, n)
func RandShardIndex(n int64) int64 {
	big, err := rand.Int(rand.Reader, big.NewInt(n))
	if err != nil {
		log.Errorf(
			"generating a random number in range of [0, %d) failed (%v),"+
				"falling back to math.Rand", n, err)
		return mathRand.Int63n(n)
	}
	return big.Int64()
}

var (
	_ ShardIterator = (*LazyShardIterator)(nil)
	_ ShardIterator = (*RandomShardIterator)(nil)
)
