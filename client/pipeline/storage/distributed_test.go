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
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"io/ioutil"
	mathRand "math/rand"
	"sync"
	"testing"

	"github.com/zero-os/0-stor/client/datastor"
	"github.com/zero-os/0-stor/client/metastor"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

func TestNewDistributedStoragePanics(t *testing.T) {
	require.Panics(t, func() {
		NewDistributedChunkStorage(nil, 1, 1, -1)
	}, "no cluster given given")
}

func TestNewDistributedStorageErrors(t *testing.T) {
	_, err := NewDistributedChunkStorage(dummyCluster{}, 0, 1, -1)
	require.Error(t, err, "no valid dataShardCount (data shard count) given")
	_, err = NewDistributedChunkStorage(dummyCluster{}, 1, 0, -1)
	require.Error(t, err, "no valid parityShardCount (parity shard count) given")
}

func TestDistributedStorageReadCheckWriteDelete(t *testing.T) {
	t.Run("dataShardCount=1,parityShardCount=1,jobCount=D", func(t *testing.T) {
		cluster, cleanup, err := newGRPCServerCluster(4)
		require.NoError(t, err)
		defer cleanup()
		defer cluster.Close()

		storage, err := NewDistributedChunkStorage(cluster, 1, 1, 0)
		require.NoError(t, err)

		testStorageReadCheckWriteDelete(t, storage)
	})

	t.Run("dataShardCount=2,parityShardCount=1,jobCount=D", func(t *testing.T) {
		cluster, cleanup, err := newGRPCServerCluster(6)
		require.NoError(t, err)
		defer cleanup()
		defer cluster.Close()

		storage, err := NewDistributedChunkStorage(cluster, 2, 1, 0)
		require.NoError(t, err)

		testStorageReadCheckWriteDelete(t, storage)
	})

	t.Run("dataShardCount=1,parityShardCount=2,jobCount=D", func(t *testing.T) {
		cluster, cleanup, err := newGRPCServerCluster(6)
		require.NoError(t, err)
		defer cleanup()
		defer cluster.Close()

		storage, err := NewDistributedChunkStorage(cluster, 1, 2, 0)
		require.NoError(t, err)

		testStorageReadCheckWriteDelete(t, storage)
	})

	t.Run("dataShardCount=2,parityShardCount=2,jobCount=1", func(t *testing.T) {
		cluster, cleanup, err := newGRPCServerCluster(8)
		require.NoError(t, err)
		defer cleanup()
		defer cluster.Close()

		storage, err := NewDistributedChunkStorage(cluster, 2, 2, 1)
		require.NoError(t, err)

		testStorageReadCheckWriteDelete(t, storage)
	})

	t.Run("dataShardCount=8,parityShardCount=8,jobCount=D", func(t *testing.T) {
		cluster, cleanup, err := newGRPCServerCluster(16)
		require.NoError(t, err)
		defer cleanup()
		defer cluster.Close()

		storage, err := NewDistributedChunkStorage(cluster, 8, 8, 0)
		require.NoError(t, err)

		testStorageReadCheckWriteDelete(t, storage)
	})

	t.Run("dataShardCount=4,parityShardCount=8,jobCount=D", func(t *testing.T) {
		cluster, cleanup, err := newGRPCServerCluster(16)
		require.NoError(t, err)
		defer cleanup()
		defer cluster.Close()

		storage, err := NewDistributedChunkStorage(cluster, 4, 8, 0)
		require.NoError(t, err)

		testStorageReadCheckWriteDelete(t, storage)
	})

	t.Run("dataShardCount=8,parityShardCount=4,jobCount=D", func(t *testing.T) {
		cluster, cleanup, err := newGRPCServerCluster(16)
		require.NoError(t, err)
		defer cleanup()
		defer cluster.Close()

		storage, err := NewDistributedChunkStorage(cluster, 8, 4, 0)
		require.NoError(t, err)

		testStorageReadCheckWriteDelete(t, storage)
	})

	t.Run("dataShardCount=8,parityShardCount=8,jobCount=1", func(t *testing.T) {
		cluster, cleanup, err := newGRPCServerCluster(16)
		require.NoError(t, err)
		defer cleanup()
		defer cluster.Close()

		storage, err := NewDistributedChunkStorage(cluster, 8, 8, 1)
		require.NoError(t, err)

		testStorageReadCheckWriteDelete(t, storage)
	})
}

func TestDistributedStorageCheckRepair(t *testing.T) {
	t.Run("dataShardCount=1,parityShardCount=1,jobCount=1", func(t *testing.T) {
		testDistributedStorageCheckRepair(t, 1, 1, 1)
	})
	t.Run("dataShardCount=1,parityShardCount=1,jobCount=D", func(t *testing.T) {
		testDistributedStorageCheckRepair(t, 1, 1, 0)
	})
	t.Run("dataShardCount=4,parityShardCount=2,jobCount=1", func(t *testing.T) {
		testDistributedStorageCheckRepair(t, 4, 2, 1)
	})
	t.Run("dataShardCount=4,parityShardCount=2,jobCount=D", func(t *testing.T) {
		testDistributedStorageCheckRepair(t, 4, 2, 0)
	})
	t.Run("dataShardCount=10,parityShardCount=3,jobCount=D", func(t *testing.T) {
		testDistributedStorageCheckRepair(t, 10, 3, 0)
	})
	t.Run("dataShardCount=10,parityShardCount=3,jobCount=1", func(t *testing.T) {
		testDistributedStorageCheckRepair(t, 10, 3, 1)
	})
}

func testDistributedStorageCheckRepair(t *testing.T, dataShardCount, parityShardCount, jobCount int) {
	require := require.New(t)

	cluster, cleanup, err := newGRPCServerCluster((dataShardCount + parityShardCount) * 2)
	require.NoError(err)
	defer cleanup()
	defer cluster.Close()

	storage, err := NewDistributedChunkStorage(cluster, dataShardCount, parityShardCount, jobCount)
	require.NoError(err)
	require.NotNil(storage)

	const (
		dataSize = 512
	)

	input := make([]byte, dataSize)
	_, err = rand.Read(input)
	require.NoError(err)

	cfg, err := storage.WriteChunk(input)
	require.NoError(err)
	require.Equal(int64(dataSize), cfg.Size)

	// with all shards intact, we should have an optional result, and reading should be possible

	status, err := storage.CheckChunk(*cfg, false)
	require.NoError(err)
	require.Equal(CheckStatusOptimal, status)

	status, err = storage.CheckChunk(*cfg, true)
	require.NoError(err)
	require.Equal(CheckStatusValid, status)

	output, err := storage.ReadChunk(*cfg)
	require.NoError(err)
	require.Equal(input, output)

	// now let's drop shards, as long as this still results in a valid, but not optimal result

	for n := 1; n <= parityShardCount; n++ {
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
		require.Len(cfg.Objects, dataShardCount+parityShardCount)
		require.Equal(int64(dataSize), cfg.Size)

		// now we should get an optimal check result again

		status, err = storage.CheckChunk(*cfg, false)
		require.NoError(err)
		require.Equal(CheckStatusOptimal, status)

		output, err = storage.ReadChunk(*cfg)
		require.NoError(err)
		require.Equal(input, output)
	}

	// now let's drop more than the allowed shard count,
	// this should always make our check fail, and repairing/reading should never be possible
	for n := parityShardCount + 1; n <= dataShardCount+parityShardCount; n++ {
		invalidateObjects(t, cfg.Objects, n, cluster)

		status, err := storage.CheckChunk(*cfg, false)
		require.NoError(err)
		require.Equal(CheckStatusInvalid, status)

		status, err = storage.CheckChunk(*cfg, true)
		require.NoError(err)
		require.Equal(CheckStatusInvalid, status)

		_, err = storage.ReadChunk(*cfg)
		require.Error(err)

		_, err = storage.RepairChunk(*cfg)
		require.Error(err)

		_, err = storage.ReadChunk(*cfg)
		require.Error(err)

		// restore by writing, so our next iteration works again

		cfg, err = storage.WriteChunk(input)
		require.NoError(err)
		require.Equal(int64(dataSize), cfg.Size)
		require.Len(cfg.Objects, dataShardCount+parityShardCount)
	}
}

func invalidateObjects(t *testing.T, objects []metastor.Object, n int, cluster datastor.Cluster) {
	// compute invalid indices
	var (
		validIndices []int
		length       = len(objects)
	)
	if n != length {
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

	// invalidate the objects, which have non-valid indices
	for i, object := range objects {
		if len(validIndices) > 0 && validIndices[0] == i {
			validIndices = validIndices[1:]
			continue
		}

		shard, err := cluster.GetShard(object.ShardID)
		require.NoError(t, err)
		require.NotNil(t, shard)

		err = shard.DeleteObject(object.Key)
		require.NoError(t, err)
	}
}

func TestReedSolomonEncoderDecoderErrors(t *testing.T) {
	require := require.New(t)

	_, err := NewReedSolomonEncoderDecoder(0, 1)
	require.Error(err, "dataShardCount is too small")
	_, err = NewReedSolomonEncoderDecoder(1, 0)
	require.Error(err, "parityShardCount is too small")

	_, err = NewReedSolomonEncoderDecoder(0, 1)
	require.Error(err, "dataShardCount is too small")
	_, err = NewReedSolomonEncoderDecoder(1, 0)
	require.Error(err, "parityShardCount is too small")

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
	t.Run("dataShardCount=1, parityShardCount=1", func(t *testing.T) {
		testReedSolomonEncoderDecoder(t, 1, 1)
	})
	t.Run("dataShardCount=1, parityShardCount=4", func(t *testing.T) {
		testReedSolomonEncoderDecoder(t, 1, 4)
	})
	t.Run("dataShardCount=4, parityShardCount=1", func(t *testing.T) {
		testReedSolomonEncoderDecoder(t, 4, 1)
	})
	t.Run("dataShardCount=4, parityShardCount=4", func(t *testing.T) {
		testReedSolomonEncoderDecoder(t, 4, 4)
	})
	t.Run("dataShardCount=16, parityShardCount=1", func(t *testing.T) {
		testReedSolomonEncoderDecoder(t, 16, 1)
	})
	t.Run("dataShardCount=1, parityShardCount=16", func(t *testing.T) {
		testReedSolomonEncoderDecoder(t, 1, 16)
	})
	t.Run("dataShardCount=16, parityShardCount=16", func(t *testing.T) {
		testReedSolomonEncoderDecoder(t, 16, 16)
	})
}

func TestReedSolomonEncoderDecoder_Issue225(t *testing.T) {
	t.Run("dataShardCount=1, parityShardCount=1", func(t *testing.T) {
		testReedSolomonEncoderDecoderIssue225Cycle(t, 1, 1)
	})
	t.Run("dataShardCount=4, parityShardCount=2", func(t *testing.T) {
		testReedSolomonEncoderDecoderIssue225Cycle(t, 4, 2)
	})
	t.Run("dataShardCount=10, parityShardCount=3", func(t *testing.T) {
		testReedSolomonEncoderDecoderIssue225Cycle(t, 10, 3)
	})
}

func testReedSolomonEncoderDecoderIssue225Cycle(t *testing.T, k, m int) {
	ed, err := NewReedSolomonEncoderDecoder(k, m)
	require.NoError(t, err)
	require.NotNil(t, ed)

	input, err := ioutil.ReadFile("../../../fixtures/client/issue_225.txt")
	require.NoError(t, err)

	parts, err := ed.Encode(input)
	require.NoError(t, err)

	output, err := ed.Decode(parts, int64(len(input)))
	require.NoError(t, err)
	require.Equal(t, input, output)
}

func TestReedSolomonEncoderDecoder_Issue225_Async(t *testing.T) {
	t.Run("dataShardCount=1, parityShardCount=1", func(t *testing.T) {
		testReedSolomonEncoderDecoderIssue225AsyncCycle(t, 1, 1)
	})
	t.Run("dataShardCount=4, parityShardCount=2", func(t *testing.T) {
		testReedSolomonEncoderDecoderIssue225AsyncCycle(t, 4, 2)
	})
	t.Run("dataShardCount=10, parityShardCount=3", func(t *testing.T) {
		testReedSolomonEncoderDecoderIssue225AsyncCycle(t, 10, 3)
	})
}

func testReedSolomonEncoderDecoderIssue225AsyncCycle(t *testing.T, k, m int) {
	ed, err := NewReedSolomonEncoderDecoder(k, m)
	require.NoError(t, err)
	require.NotNil(t, ed)

	const (
		blockCount   = 32
		maxBlockSize = 4096
	)

	// Write Direction: Generate and encode all parts

	group, ctx := errgroup.WithContext(context.Background())
	subGroup, ctx := errgroup.WithContext(ctx)

	inputCh := make(chan []byte, blockCount)
	subGroup.Go(func() error {
		defer close(inputCh)
		for i := 0; i < blockCount; i++ {
			data := make([]byte, mathRand.Int31n(maxBlockSize-32)+32)
			for i := range data {
				data[i] = '7'
			}
			select {
			case inputCh <- data:
			case <-ctx.Done():
				return nil
			}
		}
		return nil
	})

	type partsPackage struct {
		Parts    [][]byte
		DataSize int64
	}

	partsCh := make(chan partsPackage, blockCount)
	for i := 0; i < DefaultJobCount; i++ {
		subGroup.Go(func() error {
			for input := range inputCh {
				parts, err := ed.Encode(input)
				if err != nil {
					return err
				}
				select {
				case partsCh <- partsPackage{parts, int64(len(input))}:
				case <-ctx.Done():
					return nil
				}
			}
			return nil
		})
	}

	group.Go(func() error {
		err := subGroup.Wait()
		close(partsCh)
		return err
	})

	var allParts []partsPackage
	group.Go(func() error {
		for parts := range partsCh {
			allParts = append(allParts, parts)
		}
		return nil
	})

	err = group.Wait()
	require.NoError(t, err)
	require.Len(t, allParts, blockCount)

	// Read Direction: Decode all parts and validate they are correct

	group, ctx = errgroup.WithContext(context.Background())
	subGroup, ctx = errgroup.WithContext(ctx)

	expectedBlock := make([]byte, maxBlockSize)
	for i := range expectedBlock {
		expectedBlock[i] = '7'
	}

	partsCh = make(chan partsPackage, blockCount)
	subGroup.Go(func() error {
		defer close(partsCh)
		for _, parts := range allParts {
			select {
			case partsCh <- parts:
			case <-ctx.Done():
				return nil
			}
		}
		return nil
	})

	inputCh = make(chan []byte, blockCount)
	for i := 0; i < DefaultJobCount; i++ {
		subGroup.Go(func() error {
			for parts := range partsCh {
				input, err := ed.Decode(parts.Parts, parts.DataSize)
				if err != nil {
					return err
				}
				select {
				case inputCh <- input:
				case <-ctx.Done():
					return nil
				}
			}
			return nil
		})
	}

	group.Go(func() error {
		err := subGroup.Wait()
		close(inputCh)
		return err
	})

	var allBlocks [][]byte
	group.Go(func() error {
		for input := range inputCh {
			if bytes.Compare(expectedBlock[:len(input)], input) != 0 {
				return errors.New("input doesn't equal the expected content")
			}
			allBlocks = append(allBlocks, input)
		}
		return nil
	})

	err = group.Wait()
	require.NoError(t, err)
	require.Len(t, allBlocks, blockCount)
}

func TestReedSolomonEncoderDecoderAsyncUsage(t *testing.T) {
	t.Run("dataShardCount=1, parityShardCount=1, jc=2", func(t *testing.T) {
		testReedSolomonEncoderDecoderAsyncUsage(t, 1, 1, 2)
	})
	t.Run("dataShardCount=1, parityShardCount=1, jc=16", func(t *testing.T) {
		testReedSolomonEncoderDecoderAsyncUsage(t, 1, 1, 16)
	})
	t.Run("dataShardCount=4, parityShardCount=4, jc=2", func(t *testing.T) {
		testReedSolomonEncoderDecoderAsyncUsage(t, 4, 4, 2)
	})
	t.Run("dataShardCount=4, parityShardCount=4, jc=16", func(t *testing.T) {
		testReedSolomonEncoderDecoderAsyncUsage(t, 4, 4, 16)
	})
	t.Run("dataShardCount=16, parityShardCount=16, jc=2", func(t *testing.T) {
		testReedSolomonEncoderDecoderAsyncUsage(t, 16, 16, 2)
	})
	t.Run("dataShardCount=16, parityShardCount=16, jc=16", func(t *testing.T) {
		testReedSolomonEncoderDecoderAsyncUsage(t, 16, 16, 16)
	})
	t.Run("dataShardCount=16, parityShardCount=16, jc=16", func(t *testing.T) {
		testReedSolomonEncoderDecoderAsyncUsage(t, 16, 16, 128)
	})
}

func TestReedSolomonEncoderDecoderResilience(t *testing.T) {
	t.Run("dataShardCount=1, parityShardCount=1", func(t *testing.T) {
		testReedSolomonEncoderDecoderResilience(t, 1, 1)
	})
	t.Run("dataShardCount=2, parityShardCount=1", func(t *testing.T) {
		testReedSolomonEncoderDecoderResilience(t, 2, 1)
	})
	t.Run("dataShardCount=1, parityShardCount=2", func(t *testing.T) {
		testReedSolomonEncoderDecoderResilience(t, 1, 2)
	})
	t.Run("dataShardCount=5, parityShardCount=2", func(t *testing.T) {
		testReedSolomonEncoderDecoderResilience(t, 5, 2)
	})
	t.Run("dataShardCount=10, parityShardCount=3", func(t *testing.T) {
		testReedSolomonEncoderDecoderResilience(t, 10, 3)
	})
	t.Run("dataShardCount=16, parityShardCount=16", func(t *testing.T) {
		testReedSolomonEncoderDecoderResilience(t, 16, 16)
	})
}

func testReedSolomonEncoderDecoderAsyncUsage(t *testing.T, dataShardCount, parityShardCount, jobCount int) {
	assert := assert.New(t)

	ed, err := NewReedSolomonEncoderDecoder(dataShardCount, parityShardCount)
	require.NoError(t, err)
	require.NotNil(t, ed)

	require.Equal(t, dataShardCount, ed.MinimumValidShardCount())
	require.Equal(t, dataShardCount+parityShardCount, ed.RequiredShardCount())

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

			output, err := ed.Decode(parts, int64(len(input)))
			assert.NoError(err)
			assert.Equal(input, output)
		}()
	}

	wg.Wait()
}

func testReedSolomonEncoderDecoder(t *testing.T, dataShardCount, parityShardCount int) {
	require := require.New(t)

	ed, err := NewReedSolomonEncoderDecoder(dataShardCount, parityShardCount)
	require.NoError(err)
	require.NotNil(ed)

	require.Equal(dataShardCount, ed.MinimumValidShardCount())
	require.Equal(dataShardCount+parityShardCount, ed.RequiredShardCount())

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

		data, err := ed.Decode(parts, int64(len(testCase)))
		require.NoError(err)
		require.Equal(testCase, string(data))
	}
}

func testReedSolomonEncoderDecoderResilience(t *testing.T, dataShardCount, parityShardCount int) {
	require := assert.New(t)

	ed, err := NewReedSolomonEncoderDecoder(dataShardCount, parityShardCount)
	require.NoError(err)
	require.NotNil(ed)

	const (
		repeatCount = 8
		dataSize    = 512
	)

	require.Equal(dataShardCount, ed.MinimumValidShardCount())
	require.Equal(dataShardCount+parityShardCount, ed.RequiredShardCount())

	for tryCount := 0; tryCount < repeatCount; tryCount++ {
		input := make([]byte, dataSize)
		_, err = rand.Read(input)
		require.NoError(err)

		parts, err := ed.Encode(input)
		require.NoError(err)
		require.NotEmpty(parts)

		// test recovery, which should be possible, as long as `len(parts) >= dataShardCount`
		for n := 0; n <= parityShardCount; n++ {
			parts := getAllPartsMinusNParts(parts, n)
			require.True(len(parts) >= dataShardCount)

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
