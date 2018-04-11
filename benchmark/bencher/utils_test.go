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
	"testing"

	"github.com/zero-os/0-stor/client"
	"github.com/zero-os/0-stor/client/datastor/pipeline"
	zdbtest "github.com/zero-os/0-stor/client/datastor/zerodb/test"
	"github.com/zero-os/0-stor/client/processing"

	"github.com/stretchr/testify/require"
)

// newTestZstorServers returns n amount of zstor test servers
// also returns a function to clean up the servers
func newTestZstorServers(t testing.TB, n int) (servers []*testZstorServer, cleanups func()) {
	require := require.New(t)

	var (
		namespace    = "ns"
		cleanupFuncs []func()
	)

	for i := 0; i < n; i++ {
		addr, cleanup, err := zdbtest.NewInMem0DBServer(namespace)
		require.NoError(err)
		cleanupFuncs = append(cleanupFuncs, cleanup)
		servers = append(servers, &testZstorServer{
			addr: addr,
		})
	}

	cleanups = func() {
		for _, cleanup := range cleanupFuncs {
			cleanup()
		}
	}
	return
}

type testZstorServer struct {
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
