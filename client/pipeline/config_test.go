package pipeline

import (
	"bytes"
	"crypto/rand"
	mathRand "math/rand"
	"strings"
	"testing"

	"gopkg.in/yaml.v2"

	"github.com/zero-os/0-stor/client/pipeline/crypto"
	"github.com/zero-os/0-stor/client/pipeline/processing"

	"github.com/stretchr/testify/require"
)

var configTestCases = []struct {
	Description string
	Config      Config
}{
	// a nil config is valid,
	// will produce a single_object pipeline, with no processor
	// and the default hasher
	{"nil-config", Config{}},

	// some single processor-only configs
	{"compression(default)", Config{
		Compression: CompressionConfig{Mode: processing.CompressionModeDefault},
	}},
	{"compression(gzip_speed)", Config{
		Compression: CompressionConfig{
			Type: processing.CompressionTypeGZip,
			Mode: processing.CompressionModeBestSpeed,
		},
	}},
	{"encryption(default_128_bit)", Config{
		Encryption: EncryptionConfig{PrivateKey: randomString(16)},
	}},
	{"encryption(default_196_bit)", Config{
		Encryption: EncryptionConfig{PrivateKey: randomString(24)},
	}},
	{"encryption(default_256_bit)", Config{
		Encryption: EncryptionConfig{PrivateKey: randomString(32)},
	}},

	// some chained processor-only configs
	{"compression(default)+encryption(default_256-bit)", Config{
		Compression: CompressionConfig{Mode: processing.CompressionModeDefault},
		Encryption:  EncryptionConfig{PrivateKey: randomString(32)},
	}},
	{"compression(lz4)+encryption(default_256-bit)", Config{
		Compression: CompressionConfig{
			Type: processing.CompressionTypeLZ4,
			Mode: processing.CompressionModeDefault,
		},
		Encryption: EncryptionConfig{PrivateKey: randomString(32)},
	}},

	// configs which define both a processor and customized hashing
	{"hashing(blake2b_256)+encryption(default_256)", Config{
		Hashing:    HashingConfig{Type: crypto.HashTypeBlake2b256},
		Encryption: EncryptionConfig{PrivateKey: randomString(32)},
	}},
	{"hashing(blake2b_256+separate_key_256)+encryption(default_256)", Config{
		Hashing: HashingConfig{
			Type:       crypto.HashTypeBlake2b256,
			PrivateKey: randomString(32),
		},
		Encryption: EncryptionConfig{PrivateKey: randomString(32)},
	}},
	{"hashing(blake2b_256+separate_key_128)+encryption(default_256)", Config{
		Hashing: HashingConfig{
			Type:       crypto.HashTypeBlake2b256,
			PrivateKey: randomString(16),
		},
		Encryption: EncryptionConfig{PrivateKey: randomString(32)},
	}},
	{"hashing(blake2b_512)+encryption(default_256)", Config{
		Hashing:    HashingConfig{Type: crypto.HashTypeBlake2b512},
		Encryption: EncryptionConfig{PrivateKey: randomString(32)},
	}},
	{"hashing(blake2b_512+separate_key_256)+encryption(default_256)", Config{
		Hashing: HashingConfig{
			Type:       crypto.HashTypeBlake2b512,
			PrivateKey: randomString(32),
		},
		Encryption: EncryptionConfig{PrivateKey: randomString(32)},
	}},
	{"hashing(blake2b_512+separate_key_512)+encryption(default_256)", Config{
		Hashing: HashingConfig{
			Type:       crypto.HashTypeBlake2b512,
			PrivateKey: randomString(64),
		},
		Encryption: EncryptionConfig{PrivateKey: randomString(32)},
	}},
	{"hashing(default_512)+encryption(default_256)", Config{
		Hashing:    HashingConfig{Type: crypto.DefaultHash512Type},
		Encryption: EncryptionConfig{PrivateKey: randomString(32)},
	}},
	{"hashing(default_512+separate_key_512)+encryption(default_256)", Config{
		Hashing: HashingConfig{
			Type:       crypto.DefaultHash512Type,
			PrivateKey: randomString(64),
		},
		Encryption: EncryptionConfig{PrivateKey: randomString(32)},
	}},

	// chunk-only configs
	{"chunk_1", Config{ChunkSize: 1}},
	{"chunk_42", Config{ChunkSize: 42}},
	{"chunk_128", Config{ChunkSize: 128}},
	{"chunk_4096", Config{ChunkSize: 4096}},

	// distribution-only configs
	{"replication_2", Config{
		Distribution: ObjectDistributionConfig{DataShardCount: 2},
	}},
	{"replication_8", Config{
		Distribution: ObjectDistributionConfig{DataShardCount: 8},
	}},
	{"erasure_code_k=1_m=1", Config{
		Distribution: ObjectDistributionConfig{
			DataShardCount:   1,
			ParityShardCount: 1,
		},
	}},
	{"erasure_code_k=10_m=3", Config{
		Distribution: ObjectDistributionConfig{
			DataShardCount:   10,
			ParityShardCount: 3,
		},
	}},

	// no-processor configs
	{"chunk(1)+replication(2)", Config{
		ChunkSize:    1,
		Distribution: ObjectDistributionConfig{DataShardCount: 2},
	}},
	{"chunk(42)+erasure_code(k=4_m=2)", Config{
		ChunkSize: 42,
		Distribution: ObjectDistributionConfig{
			DataShardCount:   4,
			ParityShardCount: 2,
		},
	}},
	{"chunk(42)+replication(2)", Config{
		ChunkSize:    42,
		Distribution: ObjectDistributionConfig{DataShardCount: 2},
	}},
	{"chunk(127)+erasure_code(k=1_m=1)", Config{
		ChunkSize: 127,
		Distribution: ObjectDistributionConfig{
			DataShardCount:   1,
			ParityShardCount: 1,
		},
	}},
	{"hashing(blake2b_512)+chunk(129)+erasure_code(k=1_m=1)", Config{
		ChunkSize: 129,
		Hashing:   HashingConfig{Type: crypto.HashTypeBlake2b512},
		Distribution: ObjectDistributionConfig{
			DataShardCount:   1,
			ParityShardCount: 1,
		},
	}},

	{"hashing(default)+chunk(128)+encryption(default_256)+replication(1)", Config{
		ChunkSize:  128,
		Hashing:    HashingConfig{Type: crypto.DefaultHash256Type},
		Encryption: EncryptionConfig{PrivateKey: randomString(32)},
		Distribution: ObjectDistributionConfig{
			DataShardCount: 1,
		},
	}},
	{"hashing(default)+chunk(64)+compression(default)+replication(1)", Config{
		ChunkSize:   64,
		Hashing:     HashingConfig{Type: crypto.DefaultHash256Type},
		Compression: CompressionConfig{Mode: processing.CompressionModeDefault},
		Distribution: ObjectDistributionConfig{
			DataShardCount: 1,
		},
	}},
	{"hashing(blake2b_256)+chunk(32)+encryption(default_256)+replication(1)", Config{
		ChunkSize:  32,
		Hashing:    HashingConfig{Type: crypto.HashTypeBlake2b256},
		Encryption: EncryptionConfig{PrivateKey: randomString(32)},
		Distribution: ObjectDistributionConfig{
			DataShardCount: 1,
		},
	}},
	{"hashing(blake2b_256)+chunk(128)+compression(default)+replication(1)", Config{
		ChunkSize:   128,
		Hashing:     HashingConfig{Type: crypto.HashTypeBlake2b256},
		Compression: CompressionConfig{Mode: processing.CompressionModeDefault},
		Distribution: ObjectDistributionConfig{
			DataShardCount: 1,
		},
	}},
	{"hashing(blake2b_256)+chunk(128)+compression(default)+encryption(default_256)+erasure_code(k=1_m=1)", Config{
		ChunkSize:   128,
		Hashing:     HashingConfig{Type: crypto.HashTypeBlake2b256},
		Compression: CompressionConfig{Mode: processing.CompressionModeDefault},
		Encryption:  EncryptionConfig{PrivateKey: randomString(32)},
		Distribution: ObjectDistributionConfig{
			DataShardCount:   1,
			ParityShardCount: 1,
		},
	}},
	{"hashing(blake2b_512)+chunk(8)+compression(gzip_speed)+encryption(aes_256)+erasure_code(k=10_m=3)", Config{
		ChunkSize: 8,
		Hashing:   HashingConfig{Type: crypto.HashTypeBlake2b512},
		Compression: CompressionConfig{
			Type: processing.CompressionTypeGZip,
			Mode: processing.CompressionModeBestSpeed,
		},
		Encryption: EncryptionConfig{PrivateKey: randomString(32)},
		Distribution: ObjectDistributionConfig{
			DataShardCount:   10,
			ParityShardCount: 3,
		},
	}},
}

var yamlConfigTestCases = []struct {
	Description string
	YAMLConfig  string
}{
	{`nil-config`, `---
`}, {`hashing(blake2b_512)+chunk(64)+erasure_code(k=1_m=1)`, `---
chunk_size: 64
hashing:
    type: blake2b_512
distribution:
    data_shards: 1
    parity_shards: 1
`}, {`chunk(32)+erasure_code(k=1_m=1)`, `---
chunk_size: 32
distribution:
    data_shards: 1
    parity_shards: 1
`}, {`chunk(16)+compression(default)+erasure_code(k=1_m=1)`, `---
chunk_size: 16
compression:
    mode: default
distribution:
    data_shards: 1
    parity_shards: 1
`}, {`hashing(blake2b_256)+chunk(19)+compression(default)+encryption(default_256)+erasure_code(k=1_m=1)`, `---
chunk_size: 19
hashing:
    type: blake2b_256
encryption:
    private_key: 01234567890123456789012345678901
distribution:
   data_shards: 1
   parity_shards: 1
`}, {`hashing(blake2b_256+key)+chunk(32)+compression(gzip_speed)+encryption(aes_256)+erasure_code(k=10_m=3)`, `---
chunk_size: 32
hashing:
    type: blake2b_256
    private_key: 12345678901234567890123456789012
encryption:
    type: aes
    private_key: 01234567890123456789012345678901
compression:
    type: gzip
    mode: best_speed
distribution:
    data_shards: 10
    parity_shards: 3
`},
}

func randomString(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return string(b)
}

func TestConfigBasedPipelines_WriteRead(t *testing.T) {
	for _, testCase := range configTestCases {
		t.Run(testCase.Description, func(t *testing.T) {
			shardCount := requiredShardCount(testCase.Config.Distribution)
			cluster, cleanup, err := newGRPCServerCluster(shardCount)
			require.NoError(t, err)
			defer cleanup()

			pipeline, err := NewPipeline(testCase.Config, cluster, 0)
			require.NoError(t, err)

			testPipelineWriteRead(t, pipeline)
		})
	}
}

func TestYAMLConfigBasedPipelines_WriteRead(t *testing.T) {
	for _, testCase := range yamlConfigTestCases {
		t.Run(testCase.Description, func(t *testing.T) {
			var cfg Config
			err := yaml.Unmarshal([]byte(testCase.YAMLConfig), &cfg)
			require.NoErrorf(t, err, "invalid yaml: %v", testCase.YAMLConfig)

			shardCount := requiredShardCount(cfg.Distribution)
			cluster, cleanup, err := newGRPCServerCluster(shardCount)
			require.NoError(t, err)
			defer cleanup()

			pipeline, err := NewPipeline(cfg, cluster, 0)
			require.NoError(t, err)

			testPipelineWriteRead(t, pipeline)
		})
	}
}

func requiredShardCount(cfg ObjectDistributionConfig) int {
	if cfg.DataShardCount <= 0 {
		return 1
	}
	if cfg.ParityShardCount <= 0 {
		return cfg.DataShardCount
	}
	return cfg.DataShardCount + cfg.ParityShardCount
}

func testPipelineWriteRead(t *testing.T, pipeline Pipeline) {
	t.Run("fixed-data", func(t *testing.T) {
		testCases := []struct {
			Data    string
			RefList []string
		}{
			{"a", nil},
			{"Hello, World!", nil},
			{"大家好", nil},
			{"This... is my finger :)", nil},
		}
		for _, testCase := range testCases {
			testPipelineWriteReadCycle(t, pipeline, testCase.Data, testCase.RefList)
		}
	})

	t.Run("random-data", func(t *testing.T) {
		for i := 0; i < 4; i++ {
			inputData := make([]byte, mathRand.Int31n(256)+1)
			rand.Read(inputData)

			testPipelineWriteReadCycle(t, pipeline, string(inputData), nil)
		}
	})
}

func testPipelineWriteReadCycle(t *testing.T, pipeline Pipeline, inputData string, inputRefList []string) {
	r := strings.NewReader(inputData)

	chunks, err := pipeline.Write(r, inputRefList)
	require.NoError(t, err)
	require.NotEmpty(t, chunks)

	buf := bytes.NewBuffer(nil)
	outputRefList, err := pipeline.Read(chunks, buf)
	require.NoError(t, err)
	outputData := string(buf.Bytes())

	require.Equal(t, inputData, outputData)
	require.Equal(t, inputRefList, outputRefList)
}
