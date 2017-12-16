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

	os, err := NewObjectStorage(cfg, cluster, -1)
	require.NoError(err)

	pipeline := NewSingleObjectPipeline(os, pc, hc)
	testPipelineWriteRead(t, pipeline)
}
