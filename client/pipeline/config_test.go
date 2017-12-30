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
	"crypto/rand"
	"errors"
	"io"
	mathRand "math/rand"
	"os"
	"strings"
	"testing"

	"github.com/zero-os/0-stor/client/pipeline/crypto"
	"github.com/zero-os/0-stor/client/pipeline/processing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
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
	{"chunk_1", Config{BlockSize: 1}},
	{"chunk_42", Config{BlockSize: 42}},
	{"chunk_128", Config{BlockSize: 128}},
	{"chunk_4096", Config{BlockSize: 4096}},

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
		BlockSize:    1,
		Distribution: ObjectDistributionConfig{DataShardCount: 2},
	}},
	{"chunk(42)+erasure_code(k=4_m=2)", Config{
		BlockSize: 42,
		Distribution: ObjectDistributionConfig{
			DataShardCount:   4,
			ParityShardCount: 2,
		},
	}},
	{"chunk(42)+replication(2)", Config{
		BlockSize:    42,
		Distribution: ObjectDistributionConfig{DataShardCount: 2},
	}},
	{"chunk(127)+erasure_code(k=1_m=1)", Config{
		BlockSize: 127,
		Distribution: ObjectDistributionConfig{
			DataShardCount:   1,
			ParityShardCount: 1,
		},
	}},
	{"hashing(blake2b_512)+chunk(129)+erasure_code(k=1_m=1)", Config{
		BlockSize: 129,
		Hashing:   HashingConfig{Type: crypto.HashTypeBlake2b512},
		Distribution: ObjectDistributionConfig{
			DataShardCount:   1,
			ParityShardCount: 1,
		},
	}},

	{"hashing(default)+chunk(128)+encryption(default_256)+replication(1)", Config{
		BlockSize:  128,
		Hashing:    HashingConfig{Type: crypto.DefaultHash256Type},
		Encryption: EncryptionConfig{PrivateKey: randomString(32)},
		Distribution: ObjectDistributionConfig{
			DataShardCount: 1,
		},
	}},
	{"hashing(default)+chunk(64)+compression(default)+replication(1)", Config{
		BlockSize:   64,
		Hashing:     HashingConfig{Type: crypto.DefaultHash256Type},
		Compression: CompressionConfig{Mode: processing.CompressionModeDefault},
		Distribution: ObjectDistributionConfig{
			DataShardCount: 1,
		},
	}},
	{"hashing(blake2b_256)+chunk(32)+encryption(default_256)+replication(1)", Config{
		BlockSize:  32,
		Hashing:    HashingConfig{Type: crypto.HashTypeBlake2b256},
		Encryption: EncryptionConfig{PrivateKey: randomString(32)},
		Distribution: ObjectDistributionConfig{
			DataShardCount: 1,
		},
	}},
	{"hashing(blake2b_256)+chunk(128)+compression(default)+replication(1)", Config{
		BlockSize:   128,
		Hashing:     HashingConfig{Type: crypto.HashTypeBlake2b256},
		Compression: CompressionConfig{Mode: processing.CompressionModeDefault},
		Distribution: ObjectDistributionConfig{
			DataShardCount: 1,
		},
	}},
	{"hashing(blake2b_256)+chunk(128)+compression(default)+encryption(default_256)+erasure_code(k=1_m=1)", Config{
		BlockSize:   128,
		Hashing:     HashingConfig{Type: crypto.HashTypeBlake2b256},
		Compression: CompressionConfig{Mode: processing.CompressionModeDefault},
		Encryption:  EncryptionConfig{PrivateKey: randomString(32)},
		Distribution: ObjectDistributionConfig{
			DataShardCount:   1,
			ParityShardCount: 1,
		},
	}},
	{"hashing(blake2b_512)+chunk(8)+compression(gzip_speed)+encryption(aes_256)+erasure_code(k=10_m=3)", Config{
		BlockSize: 8,
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
block_size: 64
hashing:
    type: blake2b_512
distribution:
    data_shards: 1
    parity_shards: 1
`}, {`chunk(32)+erasure_code(k=1_m=1)`, `---
block_size: 32
distribution:
    data_shards: 1
    parity_shards: 1
`}, {`chunk(16)+compression(default)+erasure_code(k=1_m=1)`, `---
block_size: 16
compression:
    mode: default
distribution:
    data_shards: 1
    parity_shards: 1
`}, {`hashing(blake2b_256)+chunk(19)+compression(default)+encryption(default_256)+erasure_code(k=1_m=1)`, `---
block_size: 19
hashing:
    type: blake2b_256
encryption:
    private_key: 01234567890123456789012345678901
distribution:
   data_shards: 1
   parity_shards: 1
`}, {`hashing(blake2b_256+key)+chunk(32)+compression(gzip_speed)+encryption(aes_256)+erasure_code(k=10_m=3)`, `---
block_size: 32
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

			testPipelineWriteReadDelete(t, pipeline)
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

			testPipelineWriteReadDelete(t, pipeline)
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

func testPipelineWriteReadDelete(t *testing.T, pipeline Pipeline) {
	t.Run("fixed-data", func(t *testing.T) {
		testCases := []string{
			"a",
			"Hello, World!",
			"大家好",
			"This... is my finger :)",
		}
		for _, testCase := range testCases {
			testPipelineWriteReadDeleteCycle(t, pipeline, testCase)
		}
	})

	t.Run("random-data", func(t *testing.T) {
		for i := 0; i < 4; i++ {
			inputData := make([]byte, mathRand.Int31n(256)+1)
			rand.Read(inputData)

			testPipelineWriteReadDeleteCycle(t, pipeline, string(inputData))
		}
	})
}

func testPipelineWriteReadDeleteCycle(t *testing.T, pipeline Pipeline, inputData string) {
	r := strings.NewReader(inputData)

	chunks, err := pipeline.Write(r)
	require.NoError(t, err)
	require.NotEmpty(t, chunks)

	buf := bytes.NewBuffer(nil)
	err = pipeline.Read(chunks, buf)
	require.NoError(t, err)
	outputData := string(buf.Bytes())

	require.Equal(t, inputData, outputData)

	// cleanup
	err = pipeline.Delete(chunks)
	require.NoError(t, err)

	// reading should no longer be possible
	buf.Reset()
	err = pipeline.Read(chunks, buf)
	require.Error(t, err)
}

func TestPipelineReadWrite_Readme(t *testing.T) {
	testPipelineReadWriteFile(t, "../../README.md")
}

func TestPipelineReadWrite_Changelog(t *testing.T) {
	testPipelineReadWriteFile(t, "../../CHANGELOG.md")
}

func TestPipelineReadWrite_Issue225(t *testing.T) {
	testPipelineReadWriteFile(t, "../../fixtures/client/issue_225.txt")
}

func testPipelineReadWriteFile(t *testing.T, path string) {
	testCases := []struct {
		Name   string
		Config Config
	}{
		{"default", Config{}},
		{"compression(snappy)", Config{
			Compression: CompressionConfig{
				Mode: processing.CompressionModeDefault,
				Type: processing.CompressionTypeSnappy,
			},
		}},
		{"chunk(4096)", Config{
			BlockSize: 4096,
		}},
		{"chunk(4096)+compression(snappy)", Config{
			BlockSize: 4096,
			Compression: CompressionConfig{
				Mode: processing.CompressionModeDefault,
				Type: processing.CompressionTypeSnappy,
			},
		}},
		{"encryption(aes_256)", Config{
			Encryption: EncryptionConfig{
				PrivateKey: randomString(32),
				Type:       processing.EncryptionTypeAES,
			},
		}},
		{"chunk(4096)+encryption(aes_256)", Config{
			BlockSize: 4096,
			Encryption: EncryptionConfig{
				PrivateKey: randomString(32),
				Type:       processing.EncryptionTypeAES,
			},
		}},
		{"chunk(4096)+compression(snappy)+encryption(aes_256)", Config{
			BlockSize: 4096,
			Compression: CompressionConfig{
				Mode: processing.CompressionModeDefault,
				Type: processing.CompressionTypeSnappy,
			},
			Encryption: EncryptionConfig{
				PrivateKey: randomString(32),
				Type:       processing.EncryptionTypeAES,
			},
		}},
		{"distribution(k=4+m=2)", Config{
			Distribution: ObjectDistributionConfig{
				DataShardCount:   4,
				ParityShardCount: 2,
			},
		}},
		{"compression(snappy)+encryption(aes_256)+distribution(k=4+m=2)", Config{
			Compression: CompressionConfig{
				Mode: processing.CompressionModeDefault,
				Type: processing.CompressionTypeSnappy,
			},
			Encryption: EncryptionConfig{
				PrivateKey: randomString(32),
				Type:       processing.EncryptionTypeAES,
			},
			Distribution: ObjectDistributionConfig{
				DataShardCount:   4,
				ParityShardCount: 2,
			},
		}},
		{"hash(blake2b_256)+compression(snappy)+encryption(aes_256)+distribution(k=4+m=2)", Config{
			Hashing: HashingConfig{
				Type: crypto.HashTypeBlake2b256,
			},
			Compression: CompressionConfig{
				Mode: processing.CompressionModeDefault,
				Type: processing.CompressionTypeSnappy,
			},
			Encryption: EncryptionConfig{
				PrivateKey: randomString(32),
				Type:       processing.EncryptionTypeAES,
			},
			Distribution: ObjectDistributionConfig{
				DataShardCount:   4,
				ParityShardCount: 2,
			},
		}},
		{"chunk(4096)+replication(k=4)", Config{
			BlockSize: 4096,
			Distribution: ObjectDistributionConfig{
				DataShardCount: 4,
			},
		}},
		{"chunk(512)+replication(k=2+m=1)", Config{
			BlockSize: 512,
			Distribution: ObjectDistributionConfig{
				DataShardCount:   2,
				ParityShardCount: 1,
			},
		}},
		{"hashing(blake2b_256)+chunk(4096)+compression(snappy)+encryption(aes_256)+distribution(k=4+m=2", Config{
			BlockSize: 4096,
			Hashing:   HashingConfig{Type: crypto.HashTypeBlake2b256},
			Compression: CompressionConfig{
				Type: processing.CompressionTypeSnappy,
				Mode: processing.CompressionModeDefault,
			},
			Encryption: EncryptionConfig{PrivateKey: randomString(32)},
			Distribution: ObjectDistributionConfig{
				DataShardCount:   4,
				ParityShardCount: 2,
			},
		}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			shardCount := requiredShardCount(testCase.Config.Distribution)
			cluster, cleanup, err := newGRPCServerCluster(shardCount)
			require.NoError(t, err)
			defer cleanup()

			pipeline, err := NewPipeline(testCase.Config, cluster, -1)
			require.NoError(t, err)

			testPipelineFileCycle(t, pipeline, path)
		})
	}
}

func testPipelineFileCycle(t *testing.T, pipeline Pipeline, path string) {
	file, err := os.Open(path)
	require.NoError(t, err)
	defer file.Close()
	r := &readerWithBlockCollector{Reader: file}

	chunks, err := pipeline.Write(r)
	require.NoError(t, err)

	w := &blockValidator{ExpectedContent: r.Buffer.Bytes()}
	err = pipeline.Read(chunks, w)
	require.NoError(t, err)
}

type readerWithBlockCollector struct {
	io.Reader
	Buffer bytes.Buffer
}

func (r *readerWithBlockCollector) Read(p []byte) (n int, err error) {
	n, err = r.Reader.Read(p)
	if err != nil {
		return n, err
	}

	return r.Buffer.Write(p)
}

type blockValidator struct {
	ExpectedContent []byte
	Offset          int
}

func (w *blockValidator) Write(p []byte) (n int, err error) {
	if w.Offset >= len(w.ExpectedContent) {
		return 0, errors.New("received block, while expecting EOF")
	}
	max := w.Offset + len(p)
	if max > len(w.ExpectedContent) {
		return 0, errors.New("received unexpected block (OOB)")
	}
	if bytes.Compare(w.ExpectedContent[w.Offset:max], p) != 0 {
		return 0, errors.New("received unexpected block (not equal)")
	}
	w.Offset = max
	return len(p), nil
}
