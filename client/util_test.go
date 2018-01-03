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

package client

import (
	"io/ioutil"
	"net"
	"os"
	"path"
	"testing"

	"github.com/zero-os/0-stor/client/datastor"
	"github.com/zero-os/0-stor/client/metastor/etcd"
	"github.com/zero-os/0-stor/client/metastor/test"
	"github.com/zero-os/0-stor/client/pipeline"
	"github.com/zero-os/0-stor/server/api"
	"github.com/zero-os/0-stor/server/api/grpc"
	"github.com/zero-os/0-stor/server/db/badger"

	"github.com/stretchr/testify/require"
)

const (
	// path to testing public key
	testPubKeyPath = "./../devcert/jwt_pub.pem"
)

type testServer struct {
	api.Server
	addr string
}

func (ts *testServer) Address() string {
	return ts.addr
}

func testGRPCServer(t testing.TB, n int) ([]*testServer, func()) {
	require := require.New(t)

	servers := make([]*testServer, n)
	dirs := make([]string, n)

	for i := 0; i < n; i++ {
		tmpDir, err := ioutil.TempDir("", "0stortest")
		require.NoError(err)
		dirs[i] = tmpDir

		db, err := badger.New(path.Join(tmpDir, "data"), path.Join(tmpDir, "meta"))
		require.NoError(err)

		server, err := grpc.New(db, nil, 4, 0)
		require.NoError(err)

		listener, err := net.Listen("tcp", "localhost:0")
		require.NoError(err, "failed to create listener on /any/ open (local) port")

		go func() {
			err := server.Serve(listener)
			if err != nil {
				panic(err)
			}
		}()

		servers[i] = &testServer{Server: server, addr: listener.Addr().String()}
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

func getTestClient(cfg Config) (*Client, datastor.Cluster, error) {
	// create datastor cluster
	datastorCluster, err := createDataClusterFromConfig(&cfg, false)
	if err != nil {
		return nil, nil, err
	}

	// create data pipeline, using our datastor cluster
	dataPipeline, err := pipeline.NewPipeline(cfg.Pipeline, datastorCluster, -1)
	if err != nil {
		return nil, nil, err
	}

	// if no metadata shards are given, we'll use a memory client
	if len(cfg.MetaStor.Shards) == 0 {
		return NewClient(test.NewClient(), dataPipeline), datastorCluster, nil
	}

	// create metastor client first,
	// and than create our master 0-stor client with all features.
	metastorClient, err := etcd.NewClient(cfg.MetaStor.Shards, nil)
	if err != nil {
		return nil, nil, err
	}
	return NewClient(metastorClient, dataPipeline), datastorCluster, nil
}
