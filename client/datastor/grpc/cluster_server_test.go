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

package grpc

import (
	"context"
	"crypto/rand"
	"fmt"
	mathRand "math/rand"
	"testing"

	"github.com/zero-os/0-stor/client/datastor"

	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

func TestNewClusterImplicitErrors(t *testing.T) {
	require := require.New(t)

	cluster, err := NewCluster([]string{""}, "foo", nil)
	require.Error(err, "can't connect to first given address")
	require.Nil(cluster)

	addr, cleanup, err := newServer()
	require.NoError(err)
	defer cleanup()

	cluster, err = NewCluster([]string{addr, ""}, "foo", nil)
	require.Error(err, "can't connect to second given address")
	require.Nil(cluster)
}

func TestGetShard(t *testing.T) {
	require := require.New(t)

	cluster, clusterCleanup, err := newServerCluster(1)
	require.NoError(err)
	defer clusterCleanup()
	require.Equal(1, cluster.ListedShardCount())

	_, addr, clientCleanup, err := newServerClient()
	require.NoError(err)
	defer clientCleanup()

	shard, err := cluster.GetShard(addr)
	require.NoError(err)
	require.NotNil(shard)
	require.Equal(addr, shard.Identifier())

	shard, err = cluster.GetShard(addr)
	require.NoError(err)
	require.NotNil(shard)
	require.Equal(addr, shard.Identifier())

	shard, err = cluster.GetShard(cluster.listedSlice[0].address)
	require.NoError(err)
	require.NotNil(shard)
	require.Equal(cluster.listedSlice[0].address, shard.Identifier())
}

func TestGetRandomShards(t *testing.T) {
	require := require.New(t)

	cluster, clusterCleanup, err := newServerCluster(3)
	require.NoError(err)
	defer clusterCleanup()
	require.Equal(3, cluster.ListedShardCount())

	var ids []string
	for _, shard := range cluster.listedSlice {
		ids = append(ids, shard.Identifier())
	}

	for i := 0; i < 32; i++ {
		shard, err := cluster.GetRandomShard()
		require.NoError(err)
		require.NotNil(shard)
		id := shard.Identifier()
		require.NotEmpty(id)
		require.True(id == ids[0] || id == ids[1] || id == ids[2])
	}

	it := cluster.GetRandomShardIterator(nil)
	require.NotNil(it)

	require.Panics(func() {
		it.Shard()
	}, "invalid iterator, need to call Next First")

	keys := map[string]struct{}{
		ids[0]: {},
		ids[1]: {},
		ids[2]: {},
	}
	for it.Next() {
		shard := it.Shard()
		require.NotNil(shard)

		id := shard.Identifier()
		require.NotEmpty(id)
		_, ok := keys[id]
		require.True(ok)
		delete(keys, id)
	}
	require.Empty(keys)

	it = cluster.GetRandomShardIterator([]string{ids[1]})
	require.NotNil(it)

	keys = map[string]struct{}{
		ids[0]: {},
		ids[2]: {},
	}
	for it.Next() {
		shard := it.Shard()
		require.NotNil(shard)

		id := shard.Identifier()
		require.NotEmpty(id)
		_, ok := keys[id]
		require.True(ok)
		delete(keys, id)
	}
	require.Empty(keys)
}

func TestGetRandomShardAsync(t *testing.T) {
	require := require.New(t)

	const jobs = 128

	cluster, clusterCleanup, err := newServerCluster(jobs)
	require.NoError(err)
	defer clusterCleanup()
	require.Equal(jobs, cluster.ListedShardCount())

	group, ctx := errgroup.WithContext(context.Background())

	type writeResult struct {
		shardID string
		object  datastor.Object
	}
	ch := make(chan writeResult, jobs)

	for i := 0; i < jobs; i++ {
		group.Go(func() error {
			data := make([]byte, mathRand.Int31n(4096)+1)
			rand.Read(data)

			shard, err := cluster.GetRandomShard()
			if err != nil {
				return fmt.Errorf("get rand shard failed: %v", err)
			}

			key, err := shard.CreateObject(data)
			if err != nil {
				return fmt.Errorf("set error for data in shard %q: %v",
					shard.Identifier(), err)
			}

			result := writeResult{
				shardID: shard.Identifier(),
				object: datastor.Object{
					Key:  key,
					Data: data,
				},
			}

			select {
			case ch <- result:
			case <-ctx.Done():
			}

			return nil
		})

		group.Go(func() error {
			var result writeResult
			select {
			case result = <-ch:
			case <-ctx.Done():
				return nil
			}

			shard, err := cluster.GetShard(result.shardID)
			if err != nil {
				return fmt.Errorf("get shard %q for key %q: %v",
					result.shardID, result.object.Key, err)
			}
			require.Equal(result.shardID, shard.Identifier())

			outputObject, err := shard.GetObject(result.object.Key)
			if err != nil {
				return fmt.Errorf("get error for key %q in shard %q: %v",
					result.object.Key, result.shardID, err)
			}

			require.NotNil(outputObject)
			object := result.object

			require.Equal(object.Key, outputObject.Key)
			require.Len(outputObject.Data, len(object.Data))
			require.Equal(outputObject.Data, object.Data)

			return nil
		})
	}

	err = group.Wait()
	require.NoError(err)
}

func TestGetRandomShardIteratorAsync(t *testing.T) {
	require := require.New(t)

	const jobs = 32

	cluster, clusterCleanup, err := newServerCluster(jobs)
	require.NoError(err)
	defer clusterCleanup()
	require.Equal(jobs, cluster.ListedShardCount())

	group, ctx := errgroup.WithContext(context.Background())

	type writeResult struct {
		shardID string
		object  datastor.Object
	}
	ch := make(chan writeResult, jobs)

	shardCh := datastor.ShardIteratorChannel(ctx, cluster.GetRandomShardIterator(nil), jobs)
	require.NotNil(shardCh)

	for i := 0; i < jobs; i++ {
		group.Go(func() error {
			data := make([]byte, mathRand.Int31n(4096)+1)
			rand.Read(data)

			var shard datastor.Shard
			select {
			case shard = <-shardCh:
			case <-ctx.Done():
				return nil
			}

			key, err := shard.CreateObject(data)
			if err != nil {
				return fmt.Errorf("set error for data in shard %q: %v",
					shard.Identifier(), err)
			}

			result := writeResult{
				shardID: shard.Identifier(),
				object: datastor.Object{
					Key:  key,
					Data: data,
				},
			}

			select {
			case ch <- result:
			case <-ctx.Done():
			}

			return nil
		})

		group.Go(func() error {
			var result writeResult
			select {
			case result = <-ch:
			case <-ctx.Done():
				return nil
			}

			shard, err := cluster.GetShard(result.shardID)
			if err != nil {
				return fmt.Errorf("get shard %q for key %q: %v",
					result.shardID, result.object.Key, err)
			}
			require.Equal(result.shardID, shard.Identifier())

			outputObject, err := shard.GetObject(result.object.Key)
			if err != nil {
				return fmt.Errorf("get error for key %q in shard %q: %v",
					result.object.Key, result.shardID, err)
			}

			require.NotNil(outputObject)
			object := result.object

			require.Equal(object.Key, outputObject.Key)
			require.Len(outputObject.Data, len(object.Data))
			require.Equal(outputObject.Data, object.Data)

			return nil
		})
	}

	err = group.Wait()
	require.NoError(err)
}
