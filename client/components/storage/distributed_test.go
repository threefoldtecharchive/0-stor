package storage

import (
	"crypto/rand"
	"fmt"
	"sync"
	"testing"

	"github.com/zero-os/0-stor/client/datastor"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDistributedStoragePanics(t *testing.T) {
	require.Panics(t, func() {
		NewDistributedObjectStorage(nil, 1, 1, -1)
	}, "no cluster given given")
}

func TestNewDistributedStorageErrors(t *testing.T) {
	_, err := NewDistributedObjectStorage(dummyCluster{}, 0, 1, -1)
	require.Error(t, err, "no valid k (data shard count) given")
	_, err = NewDistributedObjectStorage(dummyCluster{}, 1, 0, -1)
	require.Error(t, err, "no valid m (parity shard count) given")
}

func TestDistributedStorageReadCheckWrite(t *testing.T) {
	t.Run("k=1,m=1,jobCount=D", func(t *testing.T) {
		cluster, cleanup, err := newGRPCServerCluster(4)
		require.NoError(t, err)
		defer cleanup()

		storage, err := NewDistributedObjectStorage(cluster, 1, 1, 0)
		require.NoError(t, err)

		testStorageReadCheckWrite(t, storage)
	})

	t.Run("k=2,m=1,jobCount=D", func(t *testing.T) {
		cluster, cleanup, err := newGRPCServerCluster(6)
		require.NoError(t, err)
		defer cleanup()

		storage, err := NewDistributedObjectStorage(cluster, 2, 1, 0)
		require.NoError(t, err)

		testStorageReadCheckWrite(t, storage)
	})

	t.Run("k=1,m=2,jobCount=D", func(t *testing.T) {
		cluster, cleanup, err := newGRPCServerCluster(6)
		require.NoError(t, err)
		defer cleanup()

		storage, err := NewDistributedObjectStorage(cluster, 1, 2, 0)
		require.NoError(t, err)

		testStorageReadCheckWrite(t, storage)
	})

	t.Run("k=2,m=2,jobCount=1", func(t *testing.T) {
		cluster, cleanup, err := newGRPCServerCluster(8)
		require.NoError(t, err)
		defer cleanup()

		storage, err := NewDistributedObjectStorage(cluster, 2, 2, 1)
		require.NoError(t, err)

		testStorageReadCheckWrite(t, storage)
	})

	t.Run("k=8,m=8,jobCount=D", func(t *testing.T) {
		cluster, cleanup, err := newGRPCServerCluster(16)
		require.NoError(t, err)
		defer cleanup()

		storage, err := NewDistributedObjectStorage(cluster, 8, 8, 0)
		require.NoError(t, err)

		testStorageReadCheckWrite(t, storage)
	})

	t.Run("k=4,m=8,jobCount=D", func(t *testing.T) {
		cluster, cleanup, err := newGRPCServerCluster(16)
		require.NoError(t, err)
		defer cleanup()

		storage, err := NewDistributedObjectStorage(cluster, 4, 8, 0)
		require.NoError(t, err)

		testStorageReadCheckWrite(t, storage)
	})

	t.Run("k=8,m=4,jobCount=D", func(t *testing.T) {
		cluster, cleanup, err := newGRPCServerCluster(16)
		require.NoError(t, err)
		defer cleanup()

		storage, err := NewDistributedObjectStorage(cluster, 8, 4, 0)
		require.NoError(t, err)

		testStorageReadCheckWrite(t, storage)
	})

	t.Run("k=8,m=8,jobCount=1", func(t *testing.T) {
		cluster, cleanup, err := newGRPCServerCluster(16)
		require.NoError(t, err)
		defer cleanup()

		storage, err := NewDistributedObjectStorage(cluster, 8, 8, 1)
		require.NoError(t, err)

		testStorageReadCheckWrite(t, storage)
	})
}

func TestDistributedStorageCheckRepair(t *testing.T) {
	t.Run("k=1,m=1,jobCount=1", func(t *testing.T) {
		testDistributedStorageCheckRepair(t, 1, 1, 1)
	})
	t.Run("k=1,m=1,jobCount=D", func(t *testing.T) {
		testDistributedStorageCheckRepair(t, 1, 1, 0)
	})
	t.Run("k=4,m=2,jobCount=1", func(t *testing.T) {
		testDistributedStorageCheckRepair(t, 4, 2, 1)
	})
	t.Run("k=4,m=2,jobCount=D", func(t *testing.T) {
		testDistributedStorageCheckRepair(t, 4, 2, 0)
	})
	t.Run("k=10,m=3,jobCount=D", func(t *testing.T) {
		testDistributedStorageCheckRepair(t, 10, 3, 0)
	})
	t.Run("k=10,m=3,jobCount=1", func(t *testing.T) {
		testDistributedStorageCheckRepair(t, 10, 3, 1)
	})
}

func testDistributedStorageCheckRepair(t *testing.T, k, m, jobCount int) {
	require := require.New(t)

	cluster, cleanup, err := newGRPCServerCluster((k + m) * 2)
	require.NoError(err)
	defer cleanup()

	storage, err := NewDistributedObjectStorage(cluster, k, m, jobCount)
	require.NoError(err)
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
	require.Equal(ObjectCheckStatusValid, status)

	outputObject, err := storage.Read(cfg)
	require.NoError(err)
	require.Equal(inputObject, outputObject)

	// now let's drop shards, as long as this still results in a valid, but not optimal result

	for n := 1; n <= m; n++ {
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
		require.Len(cfg.Shards, k+m)
		require.Equal(dataSize, cfg.DataSize)

		// now we should get an optimal check result again

		status, err = storage.Check(cfg, false)
		require.NoError(err)
		require.Equal(ObjectCheckStatusOptimal, status)

		outputObject, err = storage.Read(cfg)
		require.NoError(err)
		require.Equal(inputObject, outputObject)
	}

	// now let's drop more than the allowed shard count,
	// this should always make our check fail, and repairing/reading should never be possible
	for n := m + 1; n <= k+m; n++ {
		invalidateShards(t, cfg.Shards, n, key, cluster)

		status, err := storage.Check(cfg, false)
		require.NoError(err)
		require.Equal(ObjectCheckStatusInvalid, status)

		status, err = storage.Check(cfg, true)
		require.NoError(err)
		require.Equal(ObjectCheckStatusInvalid, status)

		_, err = storage.Read(cfg)
		require.Error(err)

		_, err = storage.Repair(cfg)
		require.Error(err)

		_, err = storage.Read(cfg)
		require.Error(err)

		// restore by writing, so our next iteration works again

		cfg, err = storage.Write(inputObject)
		require.NoError(err)
		require.Equal(inputObject.Key, cfg.Key)
		require.Equal(dataSize, cfg.DataSize)
	}
}

func invalidateShards(t *testing.T, shards []string, n int, key []byte, cluster datastor.Cluster) {
	// compute invalid indices
	var (
		validIndices []int
		length       = len(shards)
	)
	if n != len(shards) {
		for i := 0; i < length; i++ {
			validIndices = append(validIndices, i)
		}
		realLength := int64(length)
		for i := 0; i < n; i++ {
			index := datastor.RandShardIndex(realLength)
			validIndices = append(validIndices[:index], validIndices[index+1:]...)
			realLength--
		}
	}

	fmt.Println("len(validIndices) = ", len(validIndices))
	// invalidate the shards, which have non-valid indices
	for i, shardID := range shards {
		if len(validIndices) > 0 && validIndices[0] == i {
			validIndices = validIndices[1:]
			continue
		}

		shard, err := cluster.GetShard(shardID)
		require.NoError(t, err)
		require.NotNil(t, shard)

		err = shard.DeleteObject(key)
		require.NoError(t, err)
	}
}

func TestReedSolomonEncoderDecoderErrors(t *testing.T) {
	require := require.New(t)

	_, err := NewReedSolomonEncoderDecoder(0, 1)
	require.Error(err, "k is too small")
	_, err = NewReedSolomonEncoderDecoder(1, 0)
	require.Error(err, "m is too small")

	_, err = NewReedSolomonEncoderDecoder(0, 1)
	require.Error(err, "k is too small")
	_, err = NewReedSolomonEncoderDecoder(1, 0)
	require.Error(err, "m is too small")

	require.Error(func() error {
		ed, err := NewReedSolomonEncoderDecoder(1, 1)
		require.NoError(err)
		_, err = ed.Encode(nil)
		return err
	}(), "cannot encode nil-data")
	require.Error(func() error {
		ed, err := NewReedSolomonEncoderDecoder(1, 1)
		require.NoError(err)
		_, err = ed.Decode(nil, 1)
		return err
	}(), "cannot decode 0 parts")
}

func TestReedSolomonEncoderDecoder(t *testing.T) {
	t.Run("k=1, m=1", func(t *testing.T) {
		testReedSolomonEncoderDecoder(t, 1, 1)
	})
	t.Run("k=1, m=4", func(t *testing.T) {
		testReedSolomonEncoderDecoder(t, 1, 4)
	})
	t.Run("k=4, m=1", func(t *testing.T) {
		testReedSolomonEncoderDecoder(t, 4, 1)
	})
	t.Run("k=4, m=4", func(t *testing.T) {
		testReedSolomonEncoderDecoder(t, 4, 4)
	})
	t.Run("k=16, m=1", func(t *testing.T) {
		testReedSolomonEncoderDecoder(t, 16, 1)
	})
	t.Run("k=1, m=16", func(t *testing.T) {
		testReedSolomonEncoderDecoder(t, 1, 16)
	})
	t.Run("k=16, m=16", func(t *testing.T) {
		testReedSolomonEncoderDecoder(t, 16, 16)
	})
}

func TestReedSolomonEncoderDecoderAsyncUsage(t *testing.T) {
	t.Run("k=1, m=1, jc=2", func(t *testing.T) {
		testReedSolomonEncoderDecoderAsyncUsage(t, 1, 1, 2)
	})
	t.Run("k=1, m=1, jc=16", func(t *testing.T) {
		testReedSolomonEncoderDecoderAsyncUsage(t, 1, 1, 16)
	})
	t.Run("k=4, m=4, jc=2", func(t *testing.T) {
		testReedSolomonEncoderDecoderAsyncUsage(t, 4, 4, 2)
	})
	t.Run("k=4, m=4, jc=16", func(t *testing.T) {
		testReedSolomonEncoderDecoderAsyncUsage(t, 4, 4, 16)
	})
	t.Run("k=16, m=16, jc=2", func(t *testing.T) {
		testReedSolomonEncoderDecoderAsyncUsage(t, 16, 16, 2)
	})
	t.Run("k=16, m=16, jc=16", func(t *testing.T) {
		testReedSolomonEncoderDecoderAsyncUsage(t, 16, 16, 16)
	})
	t.Run("k=16, m=16, jc=16", func(t *testing.T) {
		testReedSolomonEncoderDecoderAsyncUsage(t, 16, 16, 128)
	})
}

func TestReedSolomonEncoderDecoderResilience(t *testing.T) {
	t.Run("k=1, m=1", func(t *testing.T) {
		testReedSolomonEncoderDecoderResilience(t, 1, 1)
	})
	t.Run("k=2, m=1", func(t *testing.T) {
		testReedSolomonEncoderDecoderResilience(t, 2, 1)
	})
	t.Run("k=1, m=2", func(t *testing.T) {
		testReedSolomonEncoderDecoderResilience(t, 1, 2)
	})
	t.Run("k=5, m=2", func(t *testing.T) {
		testReedSolomonEncoderDecoderResilience(t, 5, 2)
	})
	t.Run("k=10, m=3", func(t *testing.T) {
		testReedSolomonEncoderDecoderResilience(t, 10, 3)
	})
	t.Run("k=16, m=16", func(t *testing.T) {
		testReedSolomonEncoderDecoderResilience(t, 16, 16)
	})
}

func testReedSolomonEncoderDecoderAsyncUsage(t *testing.T, k, m, jobCount int) {
	assert := assert.New(t)

	ed, err := NewReedSolomonEncoderDecoder(k, m)
	require.NoError(t, err)
	require.NotNil(t, ed)

	require.Equal(t, k, ed.MinimumValidShardCount())
	require.Equal(t, k+m, ed.RequiredShardCount())

	var wg sync.WaitGroup
	wg.Add(jobCount)

	for i := 0; i < jobCount; i++ {
		go func() {
			defer wg.Done()

			input := make([]byte, 4096)
			_, err := rand.Read(input)
			assert.NoError(err)

			parts, err := ed.Encode(input)
			assert.NoError(err)
			assert.NotEmpty(parts)

			output, err := ed.Decode(parts, len(input))
			assert.NoError(err)
			assert.Equal(input, output)
		}()
	}

	wg.Wait()
}

func testReedSolomonEncoderDecoder(t *testing.T, k, m int) {
	require := require.New(t)

	ed, err := NewReedSolomonEncoderDecoder(k, m)
	require.NoError(err)
	require.NotNil(ed)

	require.Equal(k, ed.MinimumValidShardCount())
	require.Equal(k+m, ed.RequiredShardCount())

	testCases := []string{
		"a",
		"Hello, World!",
		func() string {
			b := make([]byte, 4096)
			_, err := rand.Read(b)
			require.NoError(err)
			return string(b)
		}(),
		"大家好",
	}

	for _, testCase := range testCases {
		parts, err := ed.Encode([]byte(testCase))
		require.NoError(err)
		require.NotEmpty(parts)

		data, err := ed.Decode(parts, len(testCase))
		require.NoError(err)
		require.Equal(testCase, string(data))
	}
}

func testReedSolomonEncoderDecoderResilience(t *testing.T, k, m int) {
	require := assert.New(t)

	ed, err := NewReedSolomonEncoderDecoder(k, m)
	require.NoError(err)
	require.NotNil(ed)

	const (
		repeatCount = 8
		dataSize    = 512
	)

	require.Equal(k, ed.MinimumValidShardCount())
	require.Equal(k+m, ed.RequiredShardCount())

	for tryCount := 0; tryCount < repeatCount; tryCount++ {
		input := make([]byte, dataSize)
		_, err = rand.Read(input)
		require.NoError(err)

		parts, err := ed.Encode(input)
		require.NoError(err)
		require.NotEmpty(parts)

		// test recovery, which should be possible, as long as `len(parts) >= k`
		for n := 0; n <= m; n++ {
			parts := getAllPartsMinusNParts(parts, n)
			require.True(len(parts) >= k)

			output, err := ed.Decode(parts, dataSize)
			require.NoErrorf(err, "tryCount=%d; n=%d", tryCount, n)
			require.Equalf(input, output, "tryCount=%d; n=%d", tryCount, n)
		}
	}
}

func getAllPartsMinusNParts(parts [][]byte, n int) [][]byte {
	// compute invalid indices
	var (
		validIndices []int
		length       = len(parts)
	)
	for i := 0; i < length; i++ {
		validIndices = append(validIndices, i)
	}
	realLength := int64(length)
	for i := 0; i < n; i++ {
		index := datastor.RandShardIndex(realLength)
		validIndices = append(validIndices[:index], validIndices[index+1:]...)
		realLength--
	}

	// create output parts
	outputParts := make([][]byte, length)
	for i, part := range parts {
		if len(validIndices) == 0 || validIndices[0] != i {
			continue
		}
		validIndices = validIndices[1:]
		outputPart := make([]byte, len(part))
		copy(outputPart, part)
		outputParts[i] = outputPart
	}
	return outputParts
}
