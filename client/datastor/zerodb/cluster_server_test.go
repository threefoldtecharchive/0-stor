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

package zerodb

import (
	"context"
	"crypto/rand"
	"fmt"
	mathRand "math/rand"
	"sort"
	"testing"

	"github.com/threefoldtech/0-stor/client/datastor"
	zdbtest "github.com/threefoldtech/0-stor/client/datastor/zerodb/test"

	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

func TestNewClusterImplicitErrors(t *testing.T) {
	require := require.New(t)

	const (
		passwd    = "passwd"
		namespace = "ns"
	)

	cluster, err := NewCluster([]string{""}, passwd, namespace, nil, datastor.SpreadingTypeRandom)
	require.Error(err, "can't connect to first given address")
	require.Nil(cluster)

	_, addr, cleanup, err := zdbtest.NewInMem0DBServer(namespace)
	require.NoError(err)
	defer cleanup()

	cluster, err = NewCluster([]string{addr, ""}, passwd, namespace, nil, datastor.SpreadingTypeRandom)
	require.Error(err, "can't connect to second given address")
	require.Nil(cluster)
}

func TestGetShard(t *testing.T) {
	require := require.New(t)
	const (
		passwd    = "passwd"
		namespace = "ns"
	)

	cluster, clusterCleanup, err := newServerCluster(1, datastor.SpreadingTypeRandom)
	require.NoError(err)
	defer clusterCleanup()
	require.Equal(1, cluster.ListedShardCount())

	_, addr, clientCleanup, err := newServerClient(passwd, namespace)
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

	test := func(t *testing.T, cluster *Cluster) {

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

		it := cluster.GetShardIterator(nil)
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

		it = cluster.GetShardIterator([]string{ids[1]})
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

	t.Run("random", func(t *testing.T) {
		cluster, clusterCleanup, err := newServerCluster(3, datastor.SpreadingTypeRandom)
		require.NoError(err)
		defer clusterCleanup()
		require.Equal(3, cluster.ListedShardCount())

		test(t, cluster)
	})

	t.Run("least used", func(t *testing.T) {
		cluster, clusterCleanup, err := newServerCluster(3, datastor.SpreadingTypeLeastUsed)
		require.NoError(err)
		defer clusterCleanup()
		require.Equal(3, cluster.ListedShardCount())

		test(t, cluster)
	})
}

func TestGetRandomShardAsync(t *testing.T) {
	require := require.New(t)

	const jobs = 128

	// creates cluster
	cluster, clusterCleanup, err := newServerCluster(jobs, datastor.SpreadingTypeRandom)
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

		// fire the write worker
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

		// fire the read worker
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

			//require.Equal(object.Key, outputObject.Key)
			require.Len(outputObject.Data, len(object.Data))
			//require.Equal(outputObject.Data, object.Data)

			return nil
		})
	}

	err = group.Wait()
	require.NoError(err)
}

func TestGetShardIteratorAsync(t *testing.T) {
	require := require.New(t)

	const jobs = 32

	cluster, clusterCleanup, err := newServerCluster(jobs, datastor.SpreadingTypeRandom)
	require.NoError(err)
	defer clusterCleanup()
	require.Equal(jobs, cluster.ListedShardCount())

	group, ctx := errgroup.WithContext(context.Background())

	type writeResult struct {
		shardID string
		object  datastor.Object
	}
	ch := make(chan writeResult, jobs)

	shardCh := datastor.ShardIteratorChannel(ctx, cluster.GetShardIterator(nil), jobs)
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

func spreadTestSetup(t *testing.T) ([]*zdbtest.InMem0DBServer, *Cluster, func()) {
	require := require.New(t)
	var (
		addresses []string
		servers   []*zdbtest.InMem0DBServer
		cleanups  []func()
	)

	const (
		namespace = "ns"
		passwd    = "passwd"
		count     = 100
	)

	for i := 0; i < count; i++ {
		server, addr, cleanup, err := zdbtest.NewInMem0DBServer(namespace)
		require.NoError(err)
		cleanups = append(cleanups, cleanup)
		addresses = append(addresses, addr)
		servers = append(servers, server)
	}

	cluster, err := NewCluster(addresses, passwd, namespace, nil, datastor.SpreadingTypeRandom)
	require.NoError(err)

	cleanup := func() {
		cluster.Close()
		for _, cleanup := range cleanups {
			cleanup()
		}
	}
	return servers, cluster, cleanup
}

func spreadTest(t *testing.T, blockSize int, iterFactory func([]string) datastor.ShardIterator) {
	require := require.New(t)
	for i := 0; i < 5000; i++ {
		it := iterFactory(nil)
		require.NotNil(it)

		b := make([]byte, mathRand.Intn(1024)*10)
		i, err := rand.Read(b)
		require.NoError(err)
		require.Equal(i, len(b))

		for i := 0; i+blockSize < len(b); i = i + blockSize {
			it.Next()
			shard := it.Shard()
			require.NotNil(shard)

			_, err = shard.CreateObject(b[i : i+blockSize])
			require.NoError(err)
		}
		if it.Next() {
			shard := it.Shard()
			require.NotNil(shard)

			_, err = shard.CreateObject(b[i:])
			require.NoError(err)
		}

	}
}

func TestClusterSpread(t *testing.T) {
	t.Run("random", func(t *testing.T) {
		servers, cluster, cleanup := spreadTestSetup(t)
		defer cleanup()
		cluster.spreadingType = datastor.SpreadingTypeRandom
		spreadTest(t, 4096, cluster.GetShardIterator)

		used := make([]int, len(servers))
		for i, server := range servers {
			used[i] = server.ItemsSize()
		}
		sort.Ints(used)
		fmt.Printf("biggest utilization difference: %d\n", used[len(used)-1]-used[0])
	})

	t.Run("least used", func(t *testing.T) {
		servers, cluster, cleanup := spreadTestSetup(t)
		nrShards := len(cluster.listedSlice)
		defer cleanup()
		cluster.spreadingType = datastor.SpreadingTypeLeastUsed
		spreadTest(t, 4096, cluster.GetShardIterator)

		used := make([]int, len(servers))
		for i, server := range servers {
			used[i] = server.ItemsSize()
		}
		sort.Ints(used)
		difference := used[len(used)-1] - used[0]
		fmt.Printf("biggest utilization difference: %d\n", difference)
		require.True(t, difference <= 4096)
		require.Equal(t, len(cluster.listedSlice), nrShards)
	})

}

// create cluster with `count` number of server
func newServerCluster(count int, spreadingType datastor.SpreadingType) (clu *Cluster, cleanup func(), err error) {
	var (
		addresses []string
		cleanups  []func()
		addr      string
	)

	const (
		namespace = "ns"
		passwd    = "passwd"
	)

	for i := 0; i < count; i++ {
		_, addr, cleanup, err = zdbtest.NewInMem0DBServer(namespace)
		if err != nil {
			return
		}
		cleanups = append(cleanups, cleanup)
		addresses = append(addresses, addr)
	}

	clu, err = NewCluster(addresses, passwd, namespace, nil, spreadingType)
	if err != nil {
		return
	}

	cleanup = func() {
		clu.Close()
		for _, cleanup := range cleanups {
			cleanup()
		}
	}
	return
}

// creates server and client
func newServerClient(passwd, namespace string) (cli *Client, addr string, cleanup func(), err error) {
	var serverCleanup func()

	_, addr, serverCleanup, err = zdbtest.NewInMem0DBServer(namespace)
	if err != nil {
		return
	}
	cli, err = NewClient(addr, passwd, namespace)
	if err != nil {
		return
	}

	cleanup = func() {
		serverCleanup()
		cli.Close()
	}
	return
}
