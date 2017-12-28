package client

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

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

func getTestClient(cfg Config) (*Client, datastor.Cluster, error) {
	var (
		err             error
		datastorCluster datastor.Cluster
	)
	// create datastor cluster
	if cfg.IYO != (itsyouonline.Config{}) {
		var client *itsyouonline.Client
		client, err = itsyouonline.NewClient(cfg.IYO)
		if err == nil {
			tokenGetter := jwtTokenGetterFromIYOClient(
				cfg.IYO.Organization, client)
			datastorCluster, err = storgrpc.NewCluster(cfg.DataStor.Shards, cfg.Namespace, tokenGetter)
		}
	} else {
		datastorCluster, err = storgrpc.NewCluster(cfg.DataStor.Shards, cfg.Namespace, nil)
	}
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
		return NewClient(memory.NewClient(), dataPipeline), datastorCluster, nil
	}

	// create metastor client first,
	// and than create our master 0-stor client with all features.
	metastorClient, err := etcd.NewClient(cfg.MetaStor.Shards)
	if err != nil {
		return nil, nil, err
	}
	return NewClient(metastorClient, dataPipeline), datastorCluster, nil
}
