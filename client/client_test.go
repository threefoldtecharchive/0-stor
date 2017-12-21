package client

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"sync"
	"testing"
	"time"

	"github.com/zero-os/0-stor/client/pipeline/processing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zero-os/0-stor/client/datastor"
	storgrpc "github.com/zero-os/0-stor/client/datastor/grpc"
	"github.com/zero-os/0-stor/client/itsyouonline"
	"github.com/zero-os/0-stor/client/metastor/etcd"
	"github.com/zero-os/0-stor/client/metastor/memory"
	"github.com/zero-os/0-stor/client/pipeline"
	"github.com/zero-os/0-stor/server/api"
	"github.com/zero-os/0-stor/server/api/grpc"
	"github.com/zero-os/0-stor/server/db/badger"
)

const (
	// path to testing public key
	testPubKeyPath = "./../devcert/jwt_pub.pem"
)

func testGRPCServer(t testing.TB, n int) ([]api.Server, func()) {
	require := require.New(t)

	servers := make([]api.Server, n)
	dirs := make([]string, n)

	for i := 0; i < n; i++ {

		tmpDir, err := ioutil.TempDir("", "0stortest")
		require.NoError(err)
		dirs[i] = tmpDir

		db, err := badger.New(path.Join(tmpDir, "data"), path.Join(tmpDir, "meta"))
		require.NoError(err)

		server, err := grpc.New(db, nil, 4, 0)
		require.NoError(err)

		go func() {
			err := server.Listen("localhost:0")
			require.NoError(err, "server failed to start listening")
		}()

		servers[i] = server
	}

	clean := func() {
		for _, server := range servers {
			server.Close()
		}
		for _, dir := range dirs {
			os.RemoveAll(dir)
		}
	}

	return servers, clean
}

func getTestClient(cfg Config) (*Client, error) {
	var (
		err             error
		datastorCluster datastor.Cluster
	)
	// create datastor cluster
	if cfg.IYO != (itsyouonline.Config{}) {
		client, err := itsyouonline.NewClient(cfg.IYO)
		if err == nil {
			tokenGetter := jwtTokenGetterFromIYOClient(
				cfg.IYO.Organization, client)
			datastorCluster, err = storgrpc.NewCluster(cfg.DataStor.Shards, cfg.Namespace, tokenGetter)
		}
	} else {
		datastorCluster, err = storgrpc.NewCluster(cfg.DataStor.Shards, cfg.Namespace, nil)
	}
	if err != nil {
		return nil, err
	}

	// create data pipeline, using our datastor cluster
	dataPipeline, err := pipeline.NewPipeline(cfg.Pipeline, datastorCluster, -1)
	if err != nil {
		return nil, err
	}

	// if no metadata shards are given, we'll use a memory client
	if len(cfg.MetaStor.Shards) == 0 {
		return NewClient(datastorCluster, memory.NewClient(), dataPipeline)
	}

	// create metastor client first,
	// and than create our master 0-stor client with all features.
	metastorClient, err := etcd.NewClient(cfg.MetaStor.Shards)
	if err != nil {
		return nil, err
	}
	return NewClient(datastorCluster, metastorClient, dataPipeline)
}

func TestRoundTripGRPC(t *testing.T) {
	servers, serverClean := testGRPCServer(t, 4)
	defer serverClean()

	shards := make([]string, len(servers))
	for i, server := range servers {
		shards[i] = server.Address()
	}

	config := Config{
		Namespace: "namespace1",
		DataStor:  DataStorConfig{Shards: shards},
	}

	const blockSize = 64

	tt := []struct {
		name string

		BlockSize int

		DataShards   int
		ParityShards int

		Compress   bool
		EncryptKey string
	}{
		{
			name: "simple-write",
		},
		{
			name:      "chunks",
			BlockSize: blockSize,
		},
		{
			name:     "compress",
			Compress: true,
		},
		{
			name:       "encrypt",
			EncryptKey: "cF0BFpIsljOS8UmaP8YRHRX0nBPVRVPw",
		},
		{
			name:       "replication",
			DataShards: len(shards),
		},
		{
			name:         "distribution",
			DataShards:   2,
			ParityShards: 1,
		},
		{
			name:         "chunks-distribution",
			BlockSize:    blockSize,
			DataShards:   2,
			ParityShards: 1,
		},
		{
			name:       "compress-encrypt",
			Compress:   true,
			EncryptKey: "cF0BFpIsljOS8UmaP8YRHRX0nBPVRVPw",
		},
		{
			name:       "chunk-compress-encrypt",
			BlockSize:  blockSize,
			Compress:   true,
			EncryptKey: "cF0BFpIsljOS8UmaP8YRHRX0nBPVRVPw",
		},
		{
			name:       "compress-encrypt-replication",
			Compress:   true,
			EncryptKey: "cF0BFpIsljOS8UmaP8YRHRX0nBPVRVPw",
			DataShards: len(shards),
		},
		{
			name:         "compress-encrypt-distribution",
			Compress:     true,
			EncryptKey:   "cF0BFpIsljOS8UmaP8YRHRX0nBPVRVPw",
			DataShards:   2,
			ParityShards: 1,
		},
		{
			name:       "chunks-compress-encrypt-replication",
			BlockSize:  blockSize,
			Compress:   true,
			EncryptKey: "cF0BFpIsljOS8UmaP8YRHRX0nBPVRVPw",
			DataShards: len(shards),
		},
		{
			name:         "chunks-compress-encrypt-distribution",
			BlockSize:    blockSize,
			Compress:     true,
			EncryptKey:   "cF0BFpIsljOS8UmaP8YRHRX0nBPVRVPw",
			DataShards:   2,
			ParityShards: 1,
		},
	}

	for _, tc := range tt {

		t.Run(tc.name, func(t *testing.T) {
			config.Pipeline.BlockSize = tc.BlockSize
			if tc.Compress {
				config.Pipeline.Compression.Mode = processing.CompressionModeDefault
			} else {
				config.Pipeline.Compression.Mode = processing.CompressionModeDisabled
			}
			config.Pipeline.Encryption.PrivateKey = tc.EncryptKey
			config.Pipeline.Distribution.DataShardCount = tc.DataShards
			config.Pipeline.Distribution.ParityShardCount = tc.ParityShards

			c, err := getTestClient(config)
			require.NoError(t, err, "fail to create client")

			data := make([]byte, blockSize*4)
			_, err = rand.Read(data)
			require.NoError(t, err, "fail to read random data")

			// write data to the store
			key := []byte("testkey")
			meta, err := c.Write(key, data)
			require.NoError(t, err, "fail to write data to the store")

			// validate metadata
			assert.Equal(t, key, meta.Key, "Key in metadata is not the same")
			// assert.EqualValues(t, len(data), meta.Size(), "size in the metadat doesn't correspond with the size of the data")
			for _, chunk := range meta.Chunks {
				for _, shard := range chunk.Shards {
					assert.Contains(t, shards, shard, "shards in metadata is not one of the shards configured in the client")
				}
			}

			// b, err := json.Marshal(meta)
			// require.NoError(t, err)
			// fmt.Println(string(b))

			// read data back
			dataRead, err := c.Read(key)
			require.NoError(t, err, "fail to read data from the store")
			if bytes.Compare(data, dataRead) != 0 {
				t.Errorf("data read from store is not the same as original data")
				t.Errorf("len original: %d len actual %d", len(data), len(dataRead))
			}

			//delete data
			err = c.DeleteWithMeta(meta)
			require.NoError(t, err, "failed to delete from the store")

			// makes sure metadata does not exist anymore
			_, err = c.metastorClient.GetMetadata(key)
			require.Error(t, err)
		})
	}
}

func TestBlocksizes(t *testing.T) {
	servers, serverClean := testGRPCServer(t, 4)
	defer serverClean()

	shards := make([]string, len(servers))
	for i, server := range servers {
		shards[i] = server.Address()
	}

	const baseBlockSize = 8

	config := newDefaultConfig(shards, 0)

	for i := 0; i <= 5; i++ {
		var blockSize int
		if i == 0 {
			blockSize = baseBlockSize * 10
		} else {
			blockSize = baseBlockSize * 10 * (i * 10)
		}

		t.Run(fmt.Sprint(blockSize), func(t *testing.T) {
			config.Pipeline.BlockSize = blockSize
			c, err := getTestClient(config)
			require.NoError(t, err, "fail to create client")

			data := make([]byte, blockSize)
			_, err = rand.Read(data)
			require.NoError(t, err, "fail to read random data")

			// write data to the store
			key := []byte(fmt.Sprintf("testkey-%d", i))
			meta, err := c.Write(key, data)
			require.NoError(t, err, "fail to write data to the store")

			// validate metadata
			assert.Equal(t, key, meta.Key, "Key in metadata is not the same")
			for _, chunk := range meta.Chunks {
				for _, shard := range chunk.Shards {
					assert.Contains(t, shards, shard, "shards in metadata is not one of the shards configured in the client")
				}
			}

			// read data back
			dataRead, err := c.Read(key)
			require.NoError(t, err, "fail to read data from the store")
			if bytes.Compare(data, dataRead) != 0 {
				t.Errorf("data read from store is not the same as original data")
				t.Errorf("len original: %d len actual %d", len(data), len(dataRead))
			}
		})
	}
}

func TestMultipleDownload(t *testing.T) {
	// #test for https://github.com/zero-os/0-stor/issues/208

	servers, serverClean := testGRPCServer(t, 4)
	defer serverClean()

	shards := make([]string, len(servers))
	for i, server := range servers {
		shards[i] = server.Address()
	}

	const blockSize = 256

	config := newDefaultConfig(shards, blockSize)

	c, err := getTestClient(config)
	require.NoError(t, err, "fail to create client")
	defer c.Close()

	data := make([]byte, blockSize/16)

	_, err = rand.Read(data)
	require.NoError(t, err, "fail to read random data")
	key := []byte("testkey")

	_, err = c.Write(key, data)
	require.NoError(t, err, "fail write data")

	for i := 0; i < 100; i++ {
		result, err := c.Read(key)
		require.NoError(t, err, "fail read data")
		assert.Equal(t, data, result)
	}
}

func TestConcurrentWriteRead(t *testing.T) {
	t.SkipNow()

	servers, serverClean := testGRPCServer(t, 4)
	defer serverClean()

	shards := make([]string, len(servers))
	for i, server := range servers {
		shards[i] = server.Address()
	}

	const blockSize = 128

	config := newDefaultConfig(shards, blockSize)

	doReadWrite := func(i, size int) {
		c, err := getTestClient(config)
		require.NoError(t, err, "fail to create client")
		defer c.Close()

		data := make([]byte, size)
		_, err = rand.Read(data)
		require.NoError(t, err, "fail to read random data")
		key := []byte(fmt.Sprintf("testkey-%d", i))

		_, err = c.Write(key, data)
		require.NoError(t, err, "fail write data")

		result, err := c.Read(key)
		require.NoError(t, err, "fail read data")
		require.Equal(t, data, result, "data read is not same as data written")
	}

	// Seems we can't increased the number of concurrent write more then around 32
	for concurrent := 1; concurrent <= 64; concurrent *= 2 {
		for size := blockSize; size < blockSize*10; size *= 4 {
			name := fmt.Sprintf("Concurrent client: %d - Size of the data: %d", concurrent, size)
			t.Log(name)

			wg := &sync.WaitGroup{}
			wg.Add(10)
			start := time.Now()
			for i := 0; i < 10; i++ {
				go func(i int) {
					defer wg.Done()
					doReadWrite(i, size)
				}(i)
			}
			wg.Wait()
			end := time.Now()
			t.Logf("duration %d ms\n\n", (end.Sub(start).Nanoseconds() / 1000000))
		}
	}
}

func BenchmarkWriteFilesSizes(b *testing.B) {
	servers, serverClean := testGRPCServer(b, 4)
	defer serverClean()

	shards := make([]string, len(servers))
	for i, server := range servers {
		shards[i] = server.Address()
	}

	config := newDefaultConfig(shards, 1024*1024)

	c, err := getTestClient(config)
	require.NoError(b, err, "fail to create client")
	defer c.Close()

	tt := []struct {
		Size int
	}{
		{1024},              // 1k
		{1024 * 4},          // 4k
		{1024 * 10},         // 10k
		{1024 * 1024},       // 1M
		{1024 * 1024 * 10},  // 10M
		{1024 * 1024 * 100}, // 100M
		{1024 * 1024 * 500}, // 500M
	}

	for _, tc := range tt {

		b.Run(fmt.Sprintf("%d", tc.Size), func(b *testing.B) {

			data := make([]byte, tc.Size)
			_, err = rand.Read(data)
			require.NoError(b, err, "fail to read random data")

			key := []byte("testkey")

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				// write data to the store
				_, err := c.Write(key, data)
				require.NoError(b, err, "fail to write data to the store")
			}
		})
	}
}

func TestIssue225(t *testing.T) {
	servers, serverClean := testGRPCServer(t, 4)
	defer serverClean()

	shards := make([]string, len(servers))
	for i, server := range servers {
		shards[i] = server.Address()
	}

	const blockSize = 256

	config := newDefaultConfig(shards, blockSize)

	c, err := getTestClient(config)
	require.NoError(t, err, "fail to create client")
	defer c.Close()

	data := make([]byte, blockSize*11)

	_, err = rand.Read(data)
	require.NoError(t, err, "fail to read random data")
	key := []byte("testkey")

	_, err = c.Write(key, data)
	require.NoError(t, err, "fail write data")

	result, err := c.Read(key)
	require.NoError(t, err, "fail read data")
	assert.Equal(t, data, result)
}

func newDefaultConfig(dataShards []string, blockSize int) Config {
	return Config{
		Namespace: "namespace1",
		DataStor: DataStorConfig{
			Shards: dataShards,
		},
		Pipeline: pipeline.Config{
			BlockSize: blockSize,
			Compression: pipeline.CompressionConfig{
				Mode: processing.CompressionModeDefault,
			},
			Encryption: pipeline.EncryptionConfig{
				PrivateKey: "cF0BFpIsljOS8UmaP8YRHRX0nBPVRVPw",
			},
			Distribution: pipeline.ObjectDistributionConfig{
				DataShardCount:   2,
				ParityShardCount: 1,
			},
		},
	}
}

// func BenchmarkDirectWriteGRPC(b *testing.B) {
// 	servers, serverClean := testGRPCServer(b, 1)
// 	defer serverClean()

// 	shards := make([]string, len(servers))
// 	for i, server := range servers {
// 		shards[i] = server.ListenAddress()
// 	}

// 	conf := config.Config{
// 		Organization: "testorg",
// 		Namespace:    "testnamespace",
//
// 		Shards:       shards,
// 		MetaShards:   []string{"test"},
// 		IYOAppID:     "id",
// 		IYOSecret:    "secret",
// 	}

// 	for _, proto := range []string{"rest", "grpc"} {
// 		b.Run(proto, func(b *testing.B) {
// 			c, err := getTestClient(&conf)
// 			require.NoError(b, err, "fail to create client")

// 			data := make([]byte, 1024*1024)
// 			_, err = rand.Read(data)
// 			require.NoError(b, err, "fail to read random data")

// 			// write data to the store

// 			b.ResetTimer()
// 			for i := 0; i < b.N; i++ {
// 				key := []byte("testkey")
// 				_, err := c.Write(key, data, nil, nil, nil)
// 				require.NoError(b, err, "fail to write data to the store")
// 			}
// 		})
// 	}
// }
