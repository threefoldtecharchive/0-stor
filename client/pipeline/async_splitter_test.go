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

package pipeline

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/zero-os/0-stor/client/pipeline/crypto"
	"github.com/zero-os/0-stor/client/pipeline/processing"
	"github.com/zero-os/0-stor/client/pipeline/storage"

	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

func TestAsyncSplitterPipeline_WriteReadDeleteCheck(t *testing.T) {
	t.Run("block_size=1+pure-default", func(t *testing.T) {
		testAsyncSplitterPipelineWriteReadDeleteCheckCycle(t, ObjectDistributionConfig{}, 1, nil, nil)
	})
	t.Run("block_size=256+secure_hasher", func(t *testing.T) {
		pk := []byte(randomString(32))
		hc := func() (crypto.Hasher, error) {
			return crypto.NewBlake2b256Hasher(pk)
		}
		testAsyncSplitterPipelineWriteReadDeleteCheckCycle(t, ObjectDistributionConfig{}, 256, nil, hc)
	})
	t.Run("block_size=128+distribution(k=10+m=3)+secure_hasher+lz4_compressor", func(t *testing.T) {
		pc := func() (processing.Processor, error) {
			return processing.NewLZ4CompressorDecompressor(processing.CompressionModeDefault)
		}
		pk := []byte(randomString(32))
		hc := func() (crypto.Hasher, error) {
			return crypto.NewBlake2b256Hasher(pk)
		}
		testAsyncSplitterPipelineWriteReadDeleteCheckCycle(t, ObjectDistributionConfig{
			DataShardCount:   10,
			ParityShardCount: 3,
		}, 128, pc, hc)
	})
}

func testAsyncSplitterPipelineWriteReadDeleteCheckCycle(t *testing.T, cfg ObjectDistributionConfig, blockSize int, pc ProcessorConstructor, hc HasherConstructor) {
	require := require.New(t)

	cluster, cleanup, err := newGRPCServerCluster(requiredShardCount(cfg))
	require.NoError(err)
	defer cleanup()

	os, err := NewChunkStorage(cfg, cluster, -1)
	require.NoError(err)

	pipeline := NewAsyncSplitterPipeline(os, blockSize, pc, hc, -1)

	testPipelineWriteReadDeleteCheck(t, pipeline)
}

func TestAsyncSplitterPipeline_CheckRepair(t *testing.T) {
	t.Run("block_size=1+pure-default", func(t *testing.T) {
		testAsyncSplitterPipelineCheckRepairCycle(t, ObjectDistributionConfig{}, 1, nil, nil)
	})
	t.Run("block_size=256+secure_hasher", func(t *testing.T) {
		pk := []byte(randomString(32))
		hc := func() (crypto.Hasher, error) {
			return crypto.NewBlake2b256Hasher(pk)
		}
		testAsyncSplitterPipelineCheckRepairCycle(t, ObjectDistributionConfig{}, 256, nil, hc)
	})
	t.Run("block_size=128+distribution(k=10+m=3)+secure_hasher+lz4_compressor", func(t *testing.T) {
		pc := func() (processing.Processor, error) {
			return processing.NewLZ4CompressorDecompressor(processing.CompressionModeDefault)
		}
		pk := []byte(randomString(32))
		hc := func() (crypto.Hasher, error) {
			return crypto.NewBlake2b256Hasher(pk)
		}
		testAsyncSplitterPipelineCheckRepairCycle(t, ObjectDistributionConfig{
			DataShardCount:   10,
			ParityShardCount: 3,
		}, 128, pc, hc)
	})
}

func testAsyncSplitterPipelineCheckRepairCycle(t *testing.T, cfg ObjectDistributionConfig, blockSize int, pc ProcessorConstructor, hc HasherConstructor) {
	require := require.New(t)

	cluster, cleanup, err := newGRPCServerCluster(requiredShardCount(cfg))
	require.NoError(err)
	defer cleanup()

	os, err := NewChunkStorage(cfg, cluster, -1)
	require.NoError(err)

	pipeline := NewAsyncSplitterPipeline(os, blockSize, pc, hc, -1)

	testPipelineCheckRepair(t, pipeline)
}

func TestNewAsyncSplitterPipeline(t *testing.T) {
	require.Panics(t, func() {
		NewAsyncSplitterPipeline(nil, 42, nil, nil, -1)
	}, "no object storage given")

	cluster, cleanup, err := newGRPCServerCluster(1)
	require.NoError(t, err)
	defer cleanup()
	storage, err := storage.NewRandomChunkStorage(cluster)
	require.NoError(t, err)

	require.Panics(t, func() {
		NewAsyncSplitterPipeline(storage, 0, nil, nil, -1)
	}, "no block size given")
}

func TestDefaultAsyncSplitterPipelines(t *testing.T) {
	t.Run("block_size=1+pure-default", func(t *testing.T) {
		testDefaultAsyncSplitterPipeline(t, ObjectDistributionConfig{}, 1, nil, nil)
	})
	t.Run("block_size=1+distribution(k=1)", func(t *testing.T) {
		testDefaultAsyncSplitterPipeline(t, ObjectDistributionConfig{
			DataShardCount: 1,
		}, 1, nil, nil)
	})
	t.Run("block_size=8+distribution(k=1+m=1)", func(t *testing.T) {
		testDefaultAsyncSplitterPipeline(t, ObjectDistributionConfig{
			DataShardCount:   1,
			ParityShardCount: 1,
		}, 8, nil, nil)
	})
	t.Run("block_size=256+secure_hasher", func(t *testing.T) {
		pk := []byte(randomString(32))
		hc := func() (crypto.Hasher, error) {
			return crypto.NewBlake2b256Hasher(pk)
		}
		testDefaultAsyncSplitterPipeline(t, ObjectDistributionConfig{}, 256, nil, hc)
	})
	t.Run("block_size=42+gzip_speed_compressor", func(t *testing.T) {
		pc := func() (processing.Processor, error) {
			return processing.NewGZipCompressorDecompressor(processing.CompressionModeBestSpeed)
		}
		testDefaultAsyncSplitterPipeline(t, ObjectDistributionConfig{}, 42, pc, nil)
	})
	t.Run("block_size=64+secure_hasher+lz4_compressor", func(t *testing.T) {
		pc := func() (processing.Processor, error) {
			return processing.NewLZ4CompressorDecompressor(processing.CompressionModeDefault)
		}
		pk := []byte(randomString(32))
		hc := func() (crypto.Hasher, error) {
			return crypto.NewBlake2b256Hasher(pk)
		}
		testDefaultAsyncSplitterPipeline(t, ObjectDistributionConfig{}, 64, pc, hc)
	})
	t.Run("block_size=128+distribution(k=10+m=3)+secure_hasher+lz4_compressor", func(t *testing.T) {
		pc := func() (processing.Processor, error) {
			return processing.NewLZ4CompressorDecompressor(processing.CompressionModeDefault)
		}
		pk := []byte(randomString(32))
		hc := func() (crypto.Hasher, error) {
			return crypto.NewBlake2b256Hasher(pk)
		}
		testDefaultAsyncSplitterPipeline(t, ObjectDistributionConfig{
			DataShardCount:   10,
			ParityShardCount: 3,
		}, 128, pc, hc)
	})
}

func testDefaultAsyncSplitterPipeline(t *testing.T, cfg ObjectDistributionConfig, blockSize int, pc ProcessorConstructor, hc HasherConstructor) {
	require := require.New(t)

	cluster, cleanup, err := newGRPCServerCluster(requiredShardCount(cfg))
	require.NoError(err)
	defer cleanup()

	os, err := NewChunkStorage(cfg, cluster, -1)
	require.NoError(err)

	pipeline := NewAsyncSplitterPipeline(os, blockSize, pc, hc, -1)
	testPipelineWriteReadDelete(t, pipeline)
}

func TestAsyncDataSplitter(t *testing.T) {
	testCases := []struct {
		Input          string
		BlockSize      int
		ExpectedOutput []string
	}{
		{"", 1, nil},
		{"", 42, nil},
		{"a", 1, []string{"a"}},
		{"a", 256, []string{"a"}},
		{"ab", 1, []string{"a", "b"}},
		{"ab", 256, []string{"ab"}},
		{"abcd", 2, []string{"ab", "cd"}},
		{"abcde", 2, []string{"ab", "cd", "e"}},
	}
	for _, testCase := range testCases {
		input := []byte(testCase.Input)
		output := make([][]byte, len(testCase.ExpectedOutput))
		for i, str := range testCase.ExpectedOutput {
			output[i] = []byte(str)
		}
		testAsyncDataSplitterCycle(t, input, output, testCase.BlockSize)
	}
}

func testAsyncDataSplitterCycle(t *testing.T, input []byte, output [][]byte, blockSize int) {
	r := bytes.NewReader(input)
	group, ctx := errgroup.WithContext(context.Background())

	inputCh, splitter := newAsyncDataSplitter(ctx, r, blockSize, len(output))
	group.Go(splitter)

	outputLength := len(output)
	out := make([][]byte, outputLength)
	group.Go(func() error {
		for input := range inputCh {
			if input.Index < 0 || input.Index >= outputLength {
				return fmt.Errorf("received invalid input index '%d'", input.Index)
			}
			if len(out[input.Index]) != 0 {
				return fmt.Errorf("received double input index '%d'", input.Index)
			}
			out[input.Index] = input.Data
		}
		return nil
	})
	err := group.Wait()
	require.NoError(t, err)
	require.Equal(t, output, out)
}
