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

package grpc

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/0-stor/client/datastor/pipeline"
	"github.com/threefoldtech/0-stor/client/datastor/pipeline/storage"
	"github.com/threefoldtech/0-stor/client/datastor/zerodb"
	zdbtest "github.com/threefoldtech/0-stor/client/datastor/zerodb/test"
	"github.com/threefoldtech/0-stor/client/metastor"
	"github.com/threefoldtech/0-stor/client/metastor/db/badger"
)

const testLabel = "testLabel"

func newTestDaemon(t *testing.T) *Daemon {
	dataCluster, cleanup, err := newServerCluster(1)
	require.NoError(t, err)

	chunkStorage, err := storage.NewRandomChunkStorage(dataCluster)
	if err != nil {
		cleanup()
		t.Fatal(err)
	}
	dataPipeline := pipeline.NewSingleObjectPipeline(chunkStorage,
		pipeline.DefaultProcessorConstructor, pipeline.DefaultHasherConstructor)

	tmpDir, err := ioutil.TempDir("", "0-stor-test-daemon")
	require.NoError(t, err)
	dataDir := path.Join(tmpDir, "data")
	metaDir := path.Join(tmpDir, "meta")
	db, err := badger.New(dataDir, metaDir)
	if err != nil {
		cleanup()
		t.Fatal(err)
	}
	metaClient, err := metastor.NewClientFromConfig([]byte("namespace"), metastor.Config{Database: db})
	if err != nil {
		cleanup()
		t.Fatal(err)
	}
	cleanupB := func() {
		err := metaClient.Close()
		if err != nil {
			t.Fatal(err)
		}
		err = os.RemoveAll(tmpDir)
		if err != nil {
			t.Fatal(err)
		}
		cleanup()
	}

	daemon, err := New(Config{
		Pipeline:   dataPipeline,
		MetaClient: metaClient,
	})
	if err != nil {
		cleanupB()
		t.Fatal(err)
	}
	daemon.closer = &testCloser{func() error {
		cleanupB()
		return nil
	}}
	return daemon
}

type testCloser struct {
	cb func() error
}

func (tc *testCloser) Close() error {
	return tc.cb()
}

// create cluster with `count` number of server
func newServerCluster(count int) (clu *zerodb.Cluster, cleanup func(), err error) {
	var (
		addresses []string
		cleanups  []func()
		addr      string
	)

	const (
		namespace = "ns"
		passwd    = "passwd"
	)

	for i := 0; i < count; i++ {
		addr, cleanup, err = zdbtest.NewInMem0DBServer(namespace)
		if err != nil {
			return
		}
		cleanups = append(cleanups, cleanup)
		addresses = append(addresses, addr)
	}

	clu, err = zerodb.NewCluster(addresses, passwd, namespace, nil)
	if err != nil {
		return
	}

	cleanup = func() {
		clu.Close()
		for _, cleanup := range cleanups {
			cleanup()
		}
	}
	return
}

// creates server and client
func newServerClient(passwd, namespace string) (cli *zerodb.Client, addr string, cleanup func(), err error) {
	var serverCleanup func()

	addr, serverCleanup, err = zdbtest.NewInMem0DBServer(namespace)
	if err != nil {
		return
	}
	cli, err = zerodb.NewClient(addr, passwd, namespace)
	if err != nil {
		return
	}

	cleanup = func() {
		serverCleanup()
		cli.Close()
	}
	return
}
