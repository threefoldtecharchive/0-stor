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
	"crypto/rand"
	"fmt"
	"io"
	mrand "math/rand"
	"testing"

	"github.com/threefoldtech/0-stor/client/datastor/pipeline/crypto"
	"github.com/threefoldtech/0-stor/client/datastor/pipeline/storage"
	"github.com/threefoldtech/0-stor/client/processing"

	"github.com/stretchr/testify/assert"
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

	cluster, cleanup, err := newZdbServerCluster(requiredShardCount(cfg))
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

	cluster, cleanup, err := newZdbServerCluster(requiredShardCount(cfg))
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

	cluster, cleanup, err := newZdbServerCluster(1)
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

	cluster, cleanup, err := newZdbServerCluster(requiredShardCount(cfg))
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

// TestAsyncDataSplitterEOF ensures that known errors get ignored,
// and that read content gets returned prior to error checking.
// Added as part of the fix for https://github.com/threefoldtech/0-stor/pull/513
func TestAsyncDataSplitterEOF(t *testing.T) {
	// test that we do not return io.EOF
	r := &eofReader{bytes.NewReader(nil)}
	inputCh, splitter := newAsyncDataSplitter(
		context.Background(), r, 1, 1)
	go func() {
		err := splitter()
		if err != nil {
			panic(err)
		}
	}()
	input, ok := <-inputCh
	if !assert.False(t, ok) {
		t.Fatalf("unexpected input: %v", input)
	}

	// test that we return also partial content,
	// even when receiving EOF at the end
	r = &eofReader{bytes.NewReader([]byte{1, 2, 3})}
	inputCh, splitter = newAsyncDataSplitter(
		context.Background(), r, 2, 1)
	go func() {
		err := splitter()
		if err != nil {
			panic(err)
		}
	}()
	input, ok = <-inputCh
	require.True(t, ok)
	require.EqualValues(t, 0, input.Index)
	require.Equal(t, []byte{1, 2}, input.Data)
	input, ok = <-inputCh
	require.True(t, ok)
	require.EqualValues(t, 1, input.Index)
	require.Equal(t, []byte{3}, input.Data)
	input, ok = <-inputCh
	if !assert.False(t, ok) {
		t.Fatalf("unexpected input: %v", input)
	}
}

type eofReader struct {
	io.Reader
}

func (r *eofReader) Read(p []byte) (n int, err error) {
	n, err = r.Reader.Read(p)
	if err != nil {
		return
	}
	if n != len(p) {
		err = io.EOF
	}
	return
}

// Test that the async splitter could read
// the chunks properly with correct chunk size.
// even though the underlying reader doesn't
// always read the full chunck.
// Added as a fix reported at https://github.com/threefoldtech/0-stor/issues/499#issuecomment-374141004
func TestAsyncDataSplitterReadFullChunk(t *testing.T) {
	const (
		chunkSize    = 100
		numChunks    = 100
		lastChunkLen = 10
	)
	var (
		data = make([]byte, chunkSize*(numChunks-1)+lastChunkLen)
	)
	_, err := rand.Read(data)
	require.NoError(t, err)

	// test that we return also partial content,
	// even when receiving EOF at the end
	r := &notFullReader{
		Reader: bytes.NewReader(data),
	}
	inputCh, splitter := newAsyncDataSplitter(
		context.Background(), r, chunkSize, 1)
	go func() {
		err := splitter()
		if err != nil {
			panic(err)
		}
	}()

	var n int
	for input := range inputCh {
		n++
		if n < numChunks {
			require.Len(t, input.Data, chunkSize)
		} else {
			require.Len(t, input.Data, lastChunkLen)
		}
	}
}

// a reader that sometimes doesn't doesn't
// fill all the given buffer when doing
// read operation
type notFullReader struct {
	io.Reader
	count int
}

func (r *notFullReader) Read(p []byte) (int, error) {
	r.count++
	toRead := len(p)
	if r.count%2 == 0 {
		half := toRead / 2
		toRead = mrand.Intn(half) + half - 1
	}
	buf := make([]byte, toRead)
	n, err := r.Reader.Read(buf)
	if n != 0 {
		copy(p, buf)
	}
	return n, err
}
