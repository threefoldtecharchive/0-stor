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

package bencher

import (
	"io/ioutil"
	"net"
	"os"
	"path"
	"testing"

	"github.com/zero-os/0-stor/client"
	"github.com/zero-os/0-stor/client/datastor/pipeline"
	"github.com/zero-os/0-stor/client/processing"
	"github.com/zero-os/0-stor/server/api"
	"github.com/zero-os/0-stor/server/api/grpc"
	"github.com/zero-os/0-stor/server/db/badger"

	"github.com/stretchr/testify/require"
)

// newTestZstorServers returns n amount of zstor test servers
// also returns a function to clean up the servers
func newTestZstorServers(t testing.TB, n int) ([]*testZstorServer, func()) {
	require := require.New(t)

	servers := make([]*testZstorServer, n)
	dirs := make([]string, n)

	for i := 0; i < n; i++ {
		tmpDir, err := ioutil.TempDir("", "0stortest")
		require.NoError(err)
		dirs[i] = tmpDir

		db, err := badger.New(path.Join(tmpDir, "data"), path.Join(tmpDir, "meta"))
		require.NoError(err)

		server, err := grpc.New(db, grpc.ServerConfig{MaxMsgSize: 4})
		require.NoError(err)

		listener, err := net.Listen("tcp", "localhost:0")
		require.NoError(err, "failed to create listener on /any/ open (local) port")

		go func() {
			err := server.Serve(listener)
			if err != nil {
				panic(err)
			}
		}()

		servers[i] = &testZstorServer{Server: server, addr: listener.Addr().String()}
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

type testZstorServer struct {
	api.Server
	addr string
}

func (ts *testZstorServer) Address() string {
	return ts.addr
}

// newDefaultZstorConfig returns a default zstor client config used for testing
// with provided data shards, meta shards and blocksize
// if meta shards is nil, an in memory meta server will be used (recommended for testing)
func newDefaultZstorConfig(dataShards []string, metaShards []string, blockSize int) client.Config {
	return client.Config{
		Namespace: "namespace1",
		DataStor: client.DataStorConfig{
			Shards: dataShards,
			Pipeline: pipeline.Config{
				BlockSize: blockSize,
				Compression: pipeline.CompressionConfig{
					Mode: processing.CompressionModeDefault,
				},
				Distribution: pipeline.ObjectDistributionConfig{
					DataShardCount:   3,
					ParityShardCount: 1,
				},
			},
		},
		MetaStor: client.MetaStorConfig{
			Database: client.MetaStorETCDConfig{
				Endpoints: metaShards,
			},
		},
	}
}
