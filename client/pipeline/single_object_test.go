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
	"testing"

	"github.com/zero-os/0-stor/client/pipeline/crypto"
	"github.com/zero-os/0-stor/client/pipeline/processing"

	"github.com/stretchr/testify/require"
)

func TestNewSingleObjectPipelinePanics(t *testing.T) {
	require.Panics(t, func() {
		NewSingleObjectPipeline(nil, nil, nil)
	}, "no object storage given")
}

func TestSingleObjectPipeline_WriteReadDeleteCheck(t *testing.T) {
	t.Run("pure-default", func(t *testing.T) {
		testSingleObjectPipelineWriteReadDeleteCheckCycle(t, ObjectDistributionConfig{}, nil, nil)
	})
	t.Run("secure_hasher", func(t *testing.T) {
		pk := []byte(randomString(32))
		hc := func() (crypto.Hasher, error) {
			return crypto.NewBlake2b256Hasher(pk)
		}
		testSingleObjectPipelineWriteReadDeleteCheckCycle(t, ObjectDistributionConfig{}, nil, hc)
	})
	t.Run("distribution(k=10+m=3)+secure_hasher+lz4_compressor", func(t *testing.T) {
		pc := func() (processing.Processor, error) {
			return processing.NewLZ4CompressorDecompressor(processing.CompressionModeDefault)
		}
		pk := []byte(randomString(32))
		hc := func() (crypto.Hasher, error) {
			return crypto.NewBlake2b256Hasher(pk)
		}
		testSingleObjectPipelineWriteReadDeleteCheckCycle(t, ObjectDistributionConfig{
			DataShardCount:   10,
			ParityShardCount: 3,
		}, pc, hc)
	})
}

func testSingleObjectPipelineWriteReadDeleteCheckCycle(t *testing.T, cfg ObjectDistributionConfig, pc ProcessorConstructor, hc HasherConstructor) {
	require := require.New(t)

	cluster, cleanup, err := newGRPCServerCluster(requiredShardCount(cfg))
	require.NoError(err)
	defer cleanup()

	os, err := NewChunkStorage(cfg, cluster, -1)
	require.NoError(err)

	pipeline := NewSingleObjectPipeline(os, pc, hc)

	testPipelineWriteReadDeleteCheck(t, pipeline)
}

func TestSingleObjectPipeline_CheckRepair(t *testing.T) {
	t.Run("pure-default", func(t *testing.T) {
		testSingleObjectPipelineCheckRepairCycle(t, ObjectDistributionConfig{}, nil, nil)
	})
	t.Run("secure_hasher", func(t *testing.T) {
		pk := []byte(randomString(32))
		hc := func() (crypto.Hasher, error) {
			return crypto.NewBlake2b256Hasher(pk)
		}
		testSingleObjectPipelineCheckRepairCycle(t, ObjectDistributionConfig{}, nil, hc)
	})
	t.Run("distribution(k=10+m=3)+secure_hasher+lz4_compressor", func(t *testing.T) {
		pc := func() (processing.Processor, error) {
			return processing.NewLZ4CompressorDecompressor(processing.CompressionModeDefault)
		}
		pk := []byte(randomString(32))
		hc := func() (crypto.Hasher, error) {
			return crypto.NewBlake2b256Hasher(pk)
		}
		testSingleObjectPipelineCheckRepairCycle(t, ObjectDistributionConfig{
			DataShardCount:   10,
			ParityShardCount: 3,
		}, pc, hc)
	})
}

func testSingleObjectPipelineCheckRepairCycle(t *testing.T, cfg ObjectDistributionConfig, pc ProcessorConstructor, hc HasherConstructor) {
	require := require.New(t)

	cluster, cleanup, err := newGRPCServerCluster(requiredShardCount(cfg))
	require.NoError(err)
	defer cleanup()

	os, err := NewChunkStorage(cfg, cluster, -1)
	require.NoError(err)

	pipeline := NewSingleObjectPipeline(os, pc, hc)

	testPipelineCheckRepair(t, pipeline)
}

func TestDefaultSingleObjectPipelines(t *testing.T) {
	t.Run("pure-default", func(t *testing.T) {
		testDefaultSingleObjectPipeline(t, ObjectDistributionConfig{}, nil, nil)
	})
	t.Run("distribution(k=1)", func(t *testing.T) {
		testDefaultSingleObjectPipeline(t, ObjectDistributionConfig{
			DataShardCount: 1,
		}, nil, nil)
	})
	t.Run("distribution(k=1+m=1)", func(t *testing.T) {
		testDefaultSingleObjectPipeline(t, ObjectDistributionConfig{
			DataShardCount:   1,
			ParityShardCount: 1,
		}, nil, nil)
	})
	t.Run("secure_hasher", func(t *testing.T) {
		pk := []byte(randomString(32))
		hc := func() (crypto.Hasher, error) {
			return crypto.NewBlake2b256Hasher(pk)
		}
		testDefaultSingleObjectPipeline(t, ObjectDistributionConfig{}, nil, hc)
	})
	t.Run("gzip_speed_compressor", func(t *testing.T) {
		pc := func() (processing.Processor, error) {
			return processing.NewGZipCompressorDecompressor(processing.CompressionModeBestSpeed)
		}
		testDefaultSingleObjectPipeline(t, ObjectDistributionConfig{}, pc, nil)
	})
	t.Run("secure_hasher+lz4_compressor", func(t *testing.T) {
		pc := func() (processing.Processor, error) {
			return processing.NewLZ4CompressorDecompressor(processing.CompressionModeDefault)
		}
		pk := []byte(randomString(32))
		hc := func() (crypto.Hasher, error) {
			return crypto.NewBlake2b256Hasher(pk)
		}
		testDefaultSingleObjectPipeline(t, ObjectDistributionConfig{}, pc, hc)
	})
	t.Run("distribution(k=10+m=3)+secure_hasher+lz4_compressor", func(t *testing.T) {
		pc := func() (processing.Processor, error) {
			return processing.NewLZ4CompressorDecompressor(processing.CompressionModeDefault)
		}
		pk := []byte(randomString(32))
		hc := func() (crypto.Hasher, error) {
			return crypto.NewBlake2b256Hasher(pk)
		}
		testDefaultSingleObjectPipeline(t, ObjectDistributionConfig{
			DataShardCount:   10,
			ParityShardCount: 3,
		}, pc, hc)
	})
}

func testDefaultSingleObjectPipeline(t *testing.T, cfg ObjectDistributionConfig, pc ProcessorConstructor, hc HasherConstructor) {
	require := require.New(t)

	cluster, cleanup, err := newGRPCServerCluster(requiredShardCount(cfg))
	require.NoError(err)
	defer cleanup()

	os, err := NewChunkStorage(cfg, cluster, -1)
	require.NoError(err)

	pipeline := NewSingleObjectPipeline(os, pc, hc)
	testPipelineWriteReadDelete(t, pipeline)
}
