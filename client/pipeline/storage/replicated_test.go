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
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewReplicatedStoragePanics(t *testing.T) {
	require.Panics(t, func() {
		NewReplicatedChunkStorage(nil, 1, -1)
	}, "no cluster given given")
	require.Panics(t, func() {
		NewReplicatedChunkStorage(dummyCluster{}, 0, -1)
	}, "no valid dataShardCount given")
}

func TestReplicationStorageReadCheckWriteDelete(t *testing.T) {
	t.Run("dataShardCount=1,jobCount=D", func(t *testing.T) {
		cluster, cleanup, err := newGRPCServerCluster(2)
		require.NoError(t, err)
		defer cleanup()
		defer cluster.Close()

		storage, err := NewReplicatedChunkStorage(cluster, 1, 0)
		require.NoError(t, err)

		testStorageReadCheckWriteDelete(t, storage)
	})

	t.Run("dataShardCount=2,jobCount=D", func(t *testing.T) {
		cluster, cleanup, err := newGRPCServerCluster(4)
		require.NoError(t, err)
		defer cleanup()
		defer cluster.Close()

		storage, err := NewReplicatedChunkStorage(cluster, 2, 0)
		require.NoError(t, err)

		testStorageReadCheckWriteDelete(t, storage)
	})

	t.Run("dataShardCount=2,jobCount=1", func(t *testing.T) {
		cluster, cleanup, err := newGRPCServerCluster(4)
		require.NoError(t, err)
		defer cleanup()
		defer cluster.Close()

		storage, err := NewReplicatedChunkStorage(cluster, 2, 1)
		require.NoError(t, err)

		testStorageReadCheckWriteDelete(t, storage)
	})

	t.Run("dataShardCount=16,jobCount=D", func(t *testing.T) {
		cluster, cleanup, err := newGRPCServerCluster(32)
		require.NoError(t, err)
		defer cleanup()
		defer cluster.Close()

		storage, err := NewReplicatedChunkStorage(cluster, 16, 0)
		require.NoError(t, err)

		testStorageReadCheckWriteDelete(t, storage)
	})

	t.Run("dataShardCount=16,jobCount=1", func(t *testing.T) {
		cluster, cleanup, err := newGRPCServerCluster(32)
		require.NoError(t, err)
		defer cleanup()
		defer cluster.Close()

		storage, err := NewReplicatedChunkStorage(cluster, 16, 1)
		require.NoError(t, err)

		testStorageReadCheckWriteDelete(t, storage)
	})
}

func TestReplicatedStorageCheckRepair(t *testing.T) {
	t.Run("dataShardCount=1,jobCount=1", func(t *testing.T) {
		testReplicatedStorageCheckRepair(t, 1, 1)
	})
	t.Run("dataShardCount=1,jobCount=D", func(t *testing.T) {
		testReplicatedStorageCheckRepair(t, 1, 0)
	})
	t.Run("dataShardCount=2,jobCount=1", func(t *testing.T) {
		testReplicatedStorageCheckRepair(t, 2, 1)
	})
	t.Run("dataShardCount=2,jobCount=D", func(t *testing.T) {
		testReplicatedStorageCheckRepair(t, 2, 0)
	})
	t.Run("dataShardCount=4,jobCount=D", func(t *testing.T) {
		testReplicatedStorageCheckRepair(t, 4, 0)
	})
	t.Run("dataShardCount=4,jobCount=1", func(t *testing.T) {
		testReplicatedStorageCheckRepair(t, 4, 1)
	})
	t.Run("dataShardCount=16,jobCount=D", func(t *testing.T) {
		testReplicatedStorageCheckRepair(t, 16, 0)
	})
}

func testReplicatedStorageCheckRepair(t *testing.T, dataShardCount, jobCount int) {
	require := require.New(t)

	cluster, cleanup, err := newGRPCServerCluster(dataShardCount * 2)
	require.NoError(err)
	defer cleanup()
	defer cluster.Close()

	storage, err := NewReplicatedChunkStorage(cluster, dataShardCount, jobCount)
	require.NoError(err)

	const (
		dataSize = 512
	)

	input := make([]byte, dataSize)
	_, err = rand.Read(input)
	require.NoError(err)

	cfg, err := storage.WriteChunk(input)
	require.NoError(err)
	require.NotNil(cfg)
	require.Equal(int64(dataSize), cfg.Size)

	// with all shards intact, we should have an optional result, and reading should be possible

	status, err := storage.CheckChunk(*cfg, false)
	require.NoError(err)
	require.Equal(CheckStatusOptimal, status)

	status, err = storage.CheckChunk(*cfg, true)
	require.NoError(err)
	if dataShardCount == 1 {
		require.Equal(CheckStatusOptimal, status)
	} else {
		require.Equal(CheckStatusValid, status)
	}

	output, err := storage.ReadChunk(*cfg)
	require.NoError(err)
	require.Equal(input, output)

	// now let's drop shards, as long as there is still 2 replications it should be fine

	for n := 1; n < dataShardCount-1; n++ {
		invalidateObjects(t, cfg.Objects, n, cluster)

		// now that our shards have been messed with,
		// we have a valid, but not-optimal result (still usable/readable though)

		status, err := storage.CheckChunk(*cfg, false)
		require.NoError(err)
		require.Equal(CheckStatusValid, status)

		status, err = storage.CheckChunk(*cfg, true)
		require.NoError(err)
		require.Equal(CheckStatusValid, status)

		output, err := storage.ReadChunk(*cfg)
		require.NoError(err)
		require.Equal(input, output)

		// let's repair it to make it optimal once again,
		// this will change our config though

		cfg, err = storage.RepairChunk(*cfg)
		require.NoError(err)
		require.Len(cfg.Objects, dataShardCount)
		require.Equal(int64(dataSize), cfg.Size)

		// now we should get an optimal check result again

		status, err = storage.CheckChunk(*cfg, false)
		require.NoError(err)
		require.Equal(CheckStatusOptimal, status)

		output, err = storage.ReadChunk(*cfg)
		require.NoError(err)
		require.Equal(input, output)
	}

	// if we have only 1 shard, we should be able to repair

	invalidateObjects(t, cfg.Objects, dataShardCount-1, cluster)

	status, err = storage.CheckChunk(*cfg, false)
	require.NoError(err)

	if dataShardCount == 1 {
		require.Equal(CheckStatusOptimal, status)
	} else {
		require.Equal(CheckStatusValid, status)
	}

	status, err = storage.CheckChunk(*cfg, true)
	require.NoError(err)
	if dataShardCount == 1 {
		require.Equal(CheckStatusOptimal, status)
	} else {
		require.Equal(CheckStatusValid, status)
	}

	output, err = storage.ReadChunk(*cfg)
	require.NoError(err)
	require.Equal(input, output)

	cfg, err = storage.RepairChunk(*cfg)
	require.NoError(err)
	require.Len(cfg.Objects, dataShardCount)
	require.Equal(int64(dataSize), cfg.Size)

	output, err = storage.ReadChunk(*cfg)
	require.NoError(err)
	require.Equal(input, output)

	// restore by writing, so our last group of tests can be done as well

	cfg, err = storage.WriteChunk(input)
	require.NoError(err)
	require.Equal(int64(dataSize), cfg.Size)

	// now let's invalidate it all, this should make our check fail,
	// and it should make repairing impossible

	invalidateObjects(t, cfg.Objects, dataShardCount, cluster)

	status, err = storage.CheckChunk(*cfg, false)
	require.NoError(err)
	require.Equal(CheckStatusInvalid, status)

	status, err = storage.CheckChunk(*cfg, true)
	require.NoError(err)
	require.Equal(CheckStatusInvalid, status)

	_, err = storage.ReadChunk(*cfg)
	require.Error(err)

	_, err = storage.RepairChunk(*cfg)
	require.Error(err)
}
