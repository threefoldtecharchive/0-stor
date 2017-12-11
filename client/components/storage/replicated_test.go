package storage

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zero-os/0-stor/client/datastor"
)

func TestNewReplicatedStoragePanics(t *testing.T) {
	require.Panics(t, func() {
		NewReplicatedObjectStorage(nil, 1, -1)
	}, "no cluster given given")
	require.Panics(t, func() {
		NewReplicatedObjectStorage(dummyCluster{}, 0, -1)
	}, "no valid replicationNr given")
}

func TestReplicationStorageReadCheckWrite(t *testing.T) {
	t.Run("replicationNr=1,jobCount=D", func(t *testing.T) {
		cluster, cleanup, err := newGRPCServerCluster(2)
		require.NoError(t, err)
		defer cleanup()

		storage := NewReplicatedObjectStorage(cluster, 1, 0)
		require.NotNil(t, storage)

		testStorageReadCheckWrite(t, storage)
	})

	t.Run("replicationNr=2,jobCount=D", func(t *testing.T) {
		cluster, cleanup, err := newGRPCServerCluster(4)
		require.NoError(t, err)
		defer cleanup()

		storage := NewReplicatedObjectStorage(cluster, 2, 0)
		require.NotNil(t, storage)

		testStorageReadCheckWrite(t, storage)
	})

	t.Run("replicationNr=2,jobCount=1", func(t *testing.T) {
		cluster, cleanup, err := newGRPCServerCluster(4)
		require.NoError(t, err)
		defer cleanup()

		storage := NewReplicatedObjectStorage(cluster, 2, 1)
		require.NotNil(t, storage)

		testStorageReadCheckWrite(t, storage)
	})

	t.Run("replicationNr=16,jobCount=D", func(t *testing.T) {
		cluster, cleanup, err := newGRPCServerCluster(32)
		require.NoError(t, err)
		defer cleanup()

		storage := NewReplicatedObjectStorage(cluster, 16, 0)
		require.NotNil(t, storage)

		testStorageReadCheckWrite(t, storage)
	})

	t.Run("replicationNr=16,jobCount=1", func(t *testing.T) {
		cluster, cleanup, err := newGRPCServerCluster(32)
		require.NoError(t, err)
		defer cleanup()

		storage := NewReplicatedObjectStorage(cluster, 16, 1)
		require.NotNil(t, storage)

		testStorageReadCheckWrite(t, storage)
	})
}

func TestReplicatedStorageCheckRepair(t *testing.T) {
	t.Run("replicationNr=1,jobCount=1", func(t *testing.T) {
		testReplicatedStorageCheckRepair(t, 1, 1)
	})
	t.Run("replicationNr=1,jobCount=D", func(t *testing.T) {
		testReplicatedStorageCheckRepair(t, 1, 0)
	})
	t.Run("replicationNr=2,jobCount=1", func(t *testing.T) {
		testReplicatedStorageCheckRepair(t, 2, 1)
	})
	t.Run("replicationNr=2,jobCount=D", func(t *testing.T) {
		testReplicatedStorageCheckRepair(t, 2, 0)
	})
	t.Run("replicationNr=4,jobCount=D", func(t *testing.T) {
		testReplicatedStorageCheckRepair(t, 4, 0)
	})
	t.Run("replicationNr=4,jobCount=1", func(t *testing.T) {
		testReplicatedStorageCheckRepair(t, 4, 1)
	})
	t.Run("replicationNr=16,jobCount=D", func(t *testing.T) {
		testReplicatedStorageCheckRepair(t, 16, 0)
	})
}

func testReplicatedStorageCheckRepair(t *testing.T, replicationNr, jobCount int) {
	require := require.New(t)

	cluster, cleanup, err := newGRPCServerCluster(replicationNr * 2)
	require.NoError(err)
	defer cleanup()

	storage := NewReplicatedObjectStorage(cluster, replicationNr, jobCount)
	require.NotNil(storage)

	const (
		dataSize = 512
	)

	key := []byte("myKey")
	inputObject := datastor.Object{
		Key:           key,
		Data:          make([]byte, dataSize),
		ReferenceList: []string{"uer1", "user2"},
	}
	_, err = rand.Read(inputObject.Data)
	require.NoError(err)

	cfg, err := storage.Write(inputObject)
	require.NoError(err)
	require.Equal(inputObject.Key, cfg.Key)
	require.Equal(dataSize, cfg.DataSize)

	// with all shards intact, we should have an optional result, and reading should be possible

	status, err := storage.Check(cfg, false)
	require.NoError(err)
	require.Equal(ObjectCheckStatusOptimal, status)

	status, err = storage.Check(cfg, true)
	require.NoError(err)
	if replicationNr == 1 {
		require.Equal(ObjectCheckStatusOptimal, status)
	} else {
		require.Equal(ObjectCheckStatusValid, status)
	}

	outputObject, err := storage.Read(cfg)
	require.NoError(err)
	require.Equal(inputObject, outputObject)

	// now let's drop shards, as long as there is still 2 replications it should be fine

	for n := 1; n < replicationNr-1; n++ {
		invalidateShards(t, cfg.Shards, n, key, cluster)

		// now that our shards have been messed with,
		// we have a valid, but not-optimal result (still usable/readable though)

		status, err := storage.Check(cfg, false)
		require.NoError(err)
		require.Equal(ObjectCheckStatusValid, status)

		status, err = storage.Check(cfg, true)
		require.NoError(err)
		require.Equal(ObjectCheckStatusValid, status)

		outputObject, err := storage.Read(cfg)
		require.NoError(err)
		require.Equal(inputObject, outputObject)

		// let's repair it to make it optimal once again,
		// this will change our config though

		cfg, err = storage.Repair(cfg)
		require.NoError(err)
		require.Equal(inputObject.Key, cfg.Key)
		require.Len(cfg.Shards, replicationNr)
		require.Equal(dataSize, cfg.DataSize)

		// now we should get an optimal check result again

		status, err = storage.Check(cfg, false)
		require.NoError(err)
		require.Equal(ObjectCheckStatusOptimal, status)

		outputObject, err = storage.Read(cfg)
		require.NoError(err)
		require.Equal(inputObject, outputObject)
	}

	// if we have only 1 shard, we should be able to repair

	invalidateShards(t, cfg.Shards, replicationNr-1, key, cluster)

	status, err = storage.Check(cfg, false)
	require.NoError(err)

	if replicationNr == 1 {
		require.Equal(ObjectCheckStatusOptimal, status)
	} else {
		require.Equal(ObjectCheckStatusValid, status)
	}

	status, err = storage.Check(cfg, true)
	require.NoError(err)
	if replicationNr == 1 {
		require.Equal(ObjectCheckStatusOptimal, status)
	} else {
		require.Equal(ObjectCheckStatusValid, status)
	}

	outputObject, err = storage.Read(cfg)
	require.NoError(err)
	require.Equal(inputObject, outputObject)

	cfg, err = storage.Repair(cfg)
	require.NoError(err)
	require.Equal(inputObject.Key, cfg.Key)
	require.Len(cfg.Shards, replicationNr)
	require.Equal(dataSize, cfg.DataSize)

	outputObject, err = storage.Read(cfg)
	require.NoError(err)
	require.Equal(inputObject, outputObject)

	// restore by writing, so our last group of tests can be done as well

	cfg, err = storage.Write(inputObject)
	require.NoError(err)
	require.Equal(inputObject.Key, cfg.Key)
	require.Equal(dataSize, cfg.DataSize)

	// now let's invalidate it all, this should make our check fail,
	// and it should make repairing impossible

	invalidateShards(t, cfg.Shards, replicationNr, key, cluster)

	status, err = storage.Check(cfg, false)
	require.NoError(err)
	require.Equal(ObjectCheckStatusInvalid, status)

	status, err = storage.Check(cfg, true)
	require.NoError(err)
	require.Equal(ObjectCheckStatusInvalid, status)

	_, err = storage.Read(cfg)
	require.Error(err)

	_, err = storage.Repair(cfg)
	require.Error(err)
}
