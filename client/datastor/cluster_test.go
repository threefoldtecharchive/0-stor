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
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestShardIteratorChannelPanics(t *testing.T) {
	require := require.New(t)

	require.Panics(func() {
		ShardIteratorChannel(nil, NewRandomShardIterator(nil), 1)
	}, "no context given")
	require.Panics(func() {
		ShardIteratorChannel(context.Background(), nil, 1)
	}, "no iterator given")
}

func TestShardIteratorChannel(t *testing.T) {
	require := require.New(t)

	ch := ShardIteratorChannel(context.Background(),
		NewRandomShardIterator([]Shard{&stubShard{id: "a"}}), -1)
	require.NotNil(ch)
	select {
	case shard, open := <-ch:
		require.True(open)
		require.NotNil(shard)
		require.Equal("a", shard.Identifier())
	case <-time.After(time.Millisecond * 500):
		t.Fatal("timed out while waiting for iterator ch")
	}
	select {
	case _, open := <-ch:
		require.False(open)
	case <-time.After(time.Millisecond * 500):
		t.Fatal("timed out while waiting for iterator ch")
	}

	ch = ShardIteratorChannel(context.Background(),
		NewRandomShardIterator([]Shard{&stubShard{id: "a"}, &stubShard{id: "b"}}), -1)
	require.NotNil(ch)

	ids := map[string]struct{}{
		"a": {},
		"b": {},
	}
	for shard := range ch {
		require.NotNil(shard)
		id := shard.Identifier()
		require.NotEmpty(id)
		_, ok := ids[id]
		require.True(ok)
		delete(ids, id)
	}
	require.Empty(ids)
}

func TestRandomShardIterator(t *testing.T) {
	require := require.New(t)

	it := NewRandomShardIterator(nil)
	require.NotNil(it, "can create a nop-RandomShardIterator")
	require.Panics(func() {
		it.Shard()
	}, "need to call next first")
	require.False(it.Next())
	require.Panics(func() {
		it.Shard()
	}, "need to call next first, but this is obviously not possible, hence an invalid iterator")

	it = NewRandomShardIterator([]Shard{&stubShard{id: "a"}})
	require.NotNil(it)
	require.Panics(func() {
		it.Shard()
	}, "need to call next first")
	require.True(it.Next())
	require.Equal("a", it.Shard().Identifier())
	require.Equal("a", it.Shard().Identifier())
	require.False(it.Next())
	require.Equal("a", it.Shard().Identifier())

	it = NewRandomShardIterator([]Shard{&stubShard{id: "a"}, &stubShard{id: "b"}, &stubShard{id: "c"}})
	require.NotNil(it)
	require.Panics(func() {
		it.Shard()
	}, "need to call next first")

	var (
		ids = map[string]struct{}{
			"a": {},
			"b": {},
			"c": {},
		}
		lastID string
	)

	for it.Next() {
		shard := it.Shard()
		require.NotNil(shard)

		lastID = shard.Identifier()
		_, ok := ids[lastID]
		require.True(ok)
		delete(ids, lastID)

		require.Equal(lastID, it.Shard().Identifier())
		require.Equal(lastID, it.Shard().Identifier())
	}

	require.Empty(ids)
	require.Equal(lastID, it.Shard().Identifier())
	require.Equal(lastID, it.Shard().Identifier())
}

func TestLazyShardIteratorPanics(t *testing.T) {
	require := require.New(t)

	require.Panics(func() {
		NewLazyShardIterator(nil, []string{"a"})
	}, "no cluster given")

	require.Panics(func() {
		NewLazyShardIterator(new(stubCluster), nil)
	}, "no shards given")
}

func TestLazyShardIterator(t *testing.T) {
	require := require.New(t)

	it := NewLazyShardIterator(new(stubCluster), []string{"a"})
	require.NotNil(it)

	require.Panics(func() {
		it.Shard()
	}, "it will panic before the first Next call")

	require.True(it.Next())
	require.Equal("a", it.Shard().Identifier())
	require.Equal("a", it.Shard().Identifier())
	require.False(it.Next())
	require.Equal("a", it.Shard().Identifier())

	it = NewLazyShardIterator(new(stubCluster), []string{"b", "a", "c"})
	require.NotNil(it)

	require.Panics(func() {
		it.Shard()
	}, "it will panic before the first Next call")

	require.True(it.Next())
	require.Equal("b", it.Shard().Identifier())
	require.Equal("b", it.Shard().Identifier())
	require.True(it.Next())
	require.Equal("a", it.Shard().Identifier())
	require.True(it.Next())
	require.Equal("c", it.Shard().Identifier())
	require.Equal("c", it.Shard().Identifier())
	require.Equal("c", it.Shard().Identifier())
	require.False(it.Next())
	require.Equal("c", it.Shard().Identifier())
}

type (
	stubCluster struct {
		shards []string
	}

	stubShard struct {
		Client
		id string
	}
)

func (sc *stubCluster) GetShard(id string) (Shard, error) {
	return &stubShard{id: id}, nil
}

func (sc *stubCluster) GetRandomShard() (Shard, error) {
	n := int64(len(sc.shards))
	index := RandShardIndex(n)
	return &stubShard{id: sc.shards[index]}, nil
}

func (sc *stubCluster) GetRandomShardIterator(exceptShards []string) ShardIterator {
	slice := sc.filteredSlice(exceptShards)
	return NewRandomShardIterator(slice)
}

func (sc *stubCluster) ListedShardCount() int {
	return len(sc.shards)
}

func (sc *stubCluster) Close() error { return nil }

func (sc *stubCluster) filteredSlice(exceptShards []string) []Shard {
	if len(exceptShards) == 0 {
		slice := make([]Shard, len(sc.shards))
		for i := range slice {
			slice[i] = &stubShard{id: sc.shards[i]}
		}
		return slice
	}

	fm := make(map[string]struct{}, len(exceptShards))
	for _, id := range exceptShards {
		fm[id] = struct{}{}
	}

	var (
		ok       bool
		filtered = make([]Shard, 0, len(sc.shards))
	)
	for _, shard := range sc.shards {
		if _, ok = fm[shard]; !ok {
			filtered = append(filtered, &stubShard{id: shard})
		}
	}
	return filtered
}

func (ss *stubShard) Identifier() string {
	return ss.id
}

var (
	_ Cluster = (*stubCluster)(nil)

	_ Shard = (*stubShard)(nil)
)

func TestRandShardIndex(t *testing.T) {
	require := require.New(t)

	const max = 1024 * 64

	for u := 0; u < 16; u++ {
		seen := make(map[int64]int, max)

		for i := 0; i < max; i++ {
			n := RandShardIndex(max)
			require.True(n >= 0 && n < max)

			x := seen[n]
			x++
			require.True(x < 32)
			seen[n] = x
		}
	}
}
