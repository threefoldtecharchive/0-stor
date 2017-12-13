package grpc

import (
	"context"
	"crypto/rand"
	"fmt"
	mathRand "math/rand"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zero-os/0-stor/client/datastor"
	"golang.org/x/sync/errgroup"
)

func TestGetShard(t *testing.T) {
	require := require.New(t)

	cluster, clusterCleanup, err := newServerCluster(1)
	require.NoError(err)
	defer clusterCleanup()

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
		ids[0]: struct{}{},
		ids[1]: struct{}{},
		ids[2]: struct{}{},
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
		ids[0]: struct{}{},
		ids[2]: struct{}{},
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

	group, ctx := errgroup.WithContext(context.Background())

	type writeResult struct {
		shardID string
		object  datastor.Object
	}
	ch := make(chan writeResult, jobs)

	for i := 0; i < jobs; i++ {
		i := i
		group.Go(func() error {
			key := []byte(fmt.Sprintf("key#%d", i+1))
			data := make([]byte, mathRand.Int31n(4096)+1)
			rand.Read(data)

			refList := make([]string, mathRand.Int31n(16)+1)
			for i := range refList {
				id := make([]byte, mathRand.Int31n(128)+1)
				rand.Read(id)
				refList[i] = string(id)
			}

			object := datastor.Object{
				Key:           key,
				Data:          data,
				ReferenceList: refList,
			}

			shard, err := cluster.GetRandomShard()
			if err != nil {
				return fmt.Errorf("get rand shard for key %q: %v", object.Key, err)
			}

			err = shard.SetObject(object)
			if err != nil {
				return fmt.Errorf("set error for key %q in shard %q: %v",
					object.Key, shard.Identifier(), err)
			}

			result := writeResult{
				shardID: shard.Identifier(),
				object:  object,
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
			require.Len(outputObject.ReferenceList, len(object.ReferenceList))
			require.Equal(outputObject.ReferenceList, object.ReferenceList)

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

	group, ctx := errgroup.WithContext(context.Background())

	type writeResult struct {
		shardID string
		object  datastor.Object
	}
	ch := make(chan writeResult, jobs)

	shardCh := datastor.ShardIteratorChannel(ctx, cluster.GetRandomShardIterator(nil), jobs)
	require.NotNil(shardCh)

	for i := 0; i < jobs; i++ {
		i := i
		group.Go(func() error {
			key := []byte(fmt.Sprintf("key#%d", i+1))
			data := make([]byte, mathRand.Int31n(4096)+1)
			rand.Read(data)

			refList := make([]string, mathRand.Int31n(16)+1)
			for i := range refList {
				id := make([]byte, mathRand.Int31n(128)+1)
				rand.Read(id)
				refList[i] = string(id)
			}

			object := datastor.Object{
				Key:           key,
				Data:          data,
				ReferenceList: refList,
			}

			var shard datastor.Shard
			select {
			case shard = <-shardCh:
			case <-ctx.Done():
				return nil
			}

			err := shard.SetObject(object)
			if err != nil {
				return fmt.Errorf("set error for key %q in shard %q: %v",
					object.Key, shard.Identifier(), err)
			}

			result := writeResult{
				shardID: shard.Identifier(),
				object:  object,
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
			require.Len(outputObject.ReferenceList, len(object.ReferenceList))
			require.Equal(outputObject.ReferenceList, object.ReferenceList)

			return nil
		})
	}

	err = group.Wait()
	require.NoError(err)
}
