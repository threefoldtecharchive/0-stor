package client

import (
	"crypto/rand"
	"io/ioutil"
	"os"
	"path"
	"testing"

	jwtgo "github.com/dgrijalva/jwt-go"
	"github.com/zero-os/0-stor/client/lib/compress"
	"github.com/zero-os/0-stor/client/lib/distribution"
	"github.com/zero-os/0-stor/client/lib/encrypt"
	"github.com/zero-os/0-stor/client/lib/replication"
	"github.com/zero-os/0-stor/client/meta/embedserver"
	"github.com/zero-os/0-stor/stubs"

	log "github.com/Sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zero-os/0-stor/client/config"
	"github.com/zero-os/0-stor/server/jwt"
	"github.com/zero-os/0-stor/server/storserver"
)

func TestMain(m *testing.M) {
	log.SetLevel(log.DebugLevel)
	os.Exit(m.Run())
}

// func testGRPCServer(t *testing.T, n int) (storserver.StoreServer, *itsyouonline.Client, string, func()) {
func testGRPCServer(t testing.TB, n int) ([]storserver.StoreServer, func()) {
	servers := make([]storserver.StoreServer, n)
	dirs := make([]string, n)

	for i := 0; i < n; i++ {

		tmpDir, err := ioutil.TempDir("", "0stortest")
		require.NoError(t, err)
		dirs[i] = tmpDir

		server, err := storserver.NewGRPC(path.Join(tmpDir, "data"), path.Join(tmpDir, "meta"))
		require.NoError(t, err)

		_, err = server.Listen("localhost:0")
		require.NoError(t, err, "server failed to start listening")

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

func getTestClient(conf *config.Config) (*Client, error) {

	pubKey, err := ioutil.ReadFile("../devcert/jwt_pub.pem")
	if err != nil {
		return nil, err
	}
	jwt.SetJWTPublicKey(string(pubKey))

	b, err := ioutil.ReadFile("../devcert/jwt_key.pem")
	if err != nil {
		return nil, err
	}

	key, err := jwtgo.ParseECPrivateKeyFromPEM(b)
	if err != nil {
		return nil, err
	}

	iyoCl, err := stubs.NewStubIYOClient("testorg", key)
	if err != nil {
		return nil, err
	}

	return newClient(conf, iyoCl)
}

func TestRoundTripGRPC(t *testing.T) {
	etcd, err := embedserver.New()
	require.NoError(t, err, "fail to start embebed etcd server")
	defer etcd.Stop()

	servers, serverClean := testGRPCServer(t, 4)
	defer serverClean()

	shards := make([]string, len(servers))
	for i, server := range servers {
		shards[i] = server.Addr()
	}

	conf := config.Config{
		Organization: "testorg",
		Namespace:    "testnamespace",
		Protocol:     "grpc",
		Shards:       shards,
		MetaShards:   []string{etcd.ListenAddr()},
		IYOAppID:     "id",
		IYOSecret:    "secret",
	}

	tt := []struct {
		name  string
		pipes []config.Pipe
	}{
		{
			"no-pipe",
			[]config.Pipe{},
		},
		{
			"pipe-compress",
			[]config.Pipe{
				{
					Name: "compress",
					Type: "compress",
					Config: compress.Config{
						Type: compress.TypeSnappy,
					},
				},
			},
		},
		{
			"pipe-encrypt",
			[]config.Pipe{
				{
					Name: "encrypt",
					Type: "encrypt",
					Config: encrypt.Config{
						Type:    encrypt.TypeAESGCM,
						PrivKey: "cF0BFpIsljOS8UmaP8YRHRX0nBPVRVPw",
					},
				},
			},
		},
		{
			"pipe-replicattion-async",
			[]config.Pipe{
				{
					Name: "replication",
					Type: "replication",
					Config: replication.Config{
						Async:  true,
						Number: len(servers),
					},
				},
			},
		},
		{
			"pipe-replicattion-sync",
			[]config.Pipe{
				{
					Name: "replication",
					Type: "replication",
					Config: replication.Config{
						Async:  false,
						Number: len(servers),
					},
				},
			},
		},
		{
			"pip-distribution",
			[]config.Pipe{
				{
					Name: "distribution",
					Type: "distribution",
					Config: distribution.Config{
						Data:   3,
						Parity: 1,
					},
				},
			},
		},
		{
			"pip-compress-encrypt",
			[]config.Pipe{
				{
					Name: "compress",
					Type: "compress",
					Config: compress.Config{
						Type: compress.TypeSnappy,
					},
				},
				{
					Name: "encrypt",
					Type: "encrypt",
					Config: encrypt.Config{
						Type:    encrypt.TypeAESGCM,
						PrivKey: "cF0BFpIsljOS8UmaP8YRHRX0nBPVRVPw",
					},
				},
			},
		},
		{
			"pip-compress-encrypt-replication",
			[]config.Pipe{
				{
					Name: "compress",
					Type: "compress",
					Config: compress.Config{
						Type: compress.TypeSnappy,
					},
				},
				{
					Name: "encrypt",
					Type: "encrypt",
					Config: encrypt.Config{
						Type:    encrypt.TypeAESGCM,
						PrivKey: "cF0BFpIsljOS8UmaP8YRHRX0nBPVRVPw",
					},
				},
				{
					Name: "replication",
					Type: "replication",
					Config: replication.Config{
						Async:  true,
						Number: len(servers),
					},
				},
			},
		},
		{
			"pip-compress-distribution",
			[]config.Pipe{
				{
					Name: "compress",
					Type: "compress",
					Config: compress.Config{
						Type: compress.TypeSnappy,
					},
				},
				{
					Name: "distribution",
					Type: "distribution",
					Config: distribution.Config{
						Data:   3,
						Parity: 1,
					},
				},
			},
		},
		{
			"pip-compress-encrypt-distribution",
			[]config.Pipe{
				{
					Name: "compress",
					Type: "compress",
					Config: compress.Config{
						Type: compress.TypeSnappy,
					},
				},
				{
					Name: "encrypt",
					Type: "encrypt",
					Config: encrypt.Config{
						Type:    encrypt.TypeAESGCM,
						PrivKey: "cF0BFpIsljOS8UmaP8YRHRX0nBPVRVPw",
					},
				},
				{
					Name: "distribution",
					Type: "distribution",
					Config: distribution.Config{
						Data:   3,
						Parity: 1,
					},
				},
			},
		},
	}

	for _, tc := range tt {

		t.Run(tc.name, func(t *testing.T) {

			conf.Pipes = tc.pipes
			c, err := getTestClient(&conf)
			require.NoError(t, err, "fail to create client")

			data := make([]byte, 1024*1024)
			_, err = rand.Read(data)
			require.NoError(t, err, "fail to read random data")

			// write data to the store
			key := []byte("testkey")
			meta, err := c.Write(key, data, nil, nil, nil)
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
			assert.Equal(t, data, dataRead, "data read from store is not the same as original data")
		})
	}
}

func BenchmarkDirectWriteGRPC(b *testing.B) {
	etcd, err := embedserver.New()
	require.NoError(b, err, "fail to start embebed etcd server")
	defer etcd.Stop()

	servers, serverClean := testGRPCServer(b, 1)
	defer serverClean()

	shards := make([]string, len(servers))
	for i, server := range servers {
		shards[i] = server.Addr()
	}

	conf := config.Config{
		Organization: "testorg",
		Namespace:    "testnamespace",
		Protocol:     "grpc",
		Shards:       shards,
		MetaShards:   []string{etcd.ListenAddr()},
		IYOAppID:     "id",
		IYOSecret:    "secret",
	}

	for _, proto := range []string{"rest", "grpc"} {
		b.Run(proto, func(b *testing.B) {
			conf.Protocol = proto
			c, err := getTestClient(&conf)
			require.NoError(b, err, "fail to create client")

			data := make([]byte, 1024*1024)
			_, err = rand.Read(data)
			require.NoError(b, err, "fail to read random data")

			// write data to the store

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				key := []byte("testkey")
				_, err := c.Write(key, data, nil, nil, nil)
				require.NoError(b, err, "fail to write data to the store")
			}
		})
	}
}
