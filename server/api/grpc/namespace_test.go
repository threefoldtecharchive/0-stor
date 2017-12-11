package grpc

import (
	"context"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zero-os/0-stor/server"
	"github.com/zero-os/0-stor/server/api/grpc/rpctypes"
	pb "github.com/zero-os/0-stor/server/api/grpc/schema"
	"github.com/zero-os/0-stor/server/db"
	"github.com/zero-os/0-stor/server/db/badger"
	"github.com/zero-os/0-stor/server/encoding"
)

func TestNewNamespaceAPIPanics(t *testing.T) {
	require.Panics(t, func() {
		NewNamespaceAPI(nil)
	}, "no db given")
}

func TestGetNamespace(t *testing.T) {
	require := require.New(t)

	api, clean := getTestNamespaceAPI(t)
	defer clean()

	data, err := encoding.EncodeNamespace(server.Namespace{Label: []byte(label)})
	require.NoError(err)
	err = api.db.Set(db.NamespaceKey([]byte(label)), data)
	require.NoError(err)

	req := &pb.GetNamespaceRequest{}

	resp, err := api.GetNamespace(context.Background(), req)
	require.Error(err)
	err = rpctypes.Error(err)
	require.Equal(rpctypes.ErrNilLabel, err)

	resp, err = api.GetNamespace(contextWithLabel(nil, label), req)
	require.NoError(err)

	require.Equal(label, resp.GetLabel())
	require.EqualValues(0, resp.GetNrObjects())
}

func getTestNamespaceAPI(t *testing.T) (*NamespaceAPI, func()) {
	require := require.New(t)

	tmpDir, err := ioutil.TempDir("", "0stortest")
	require.NoError(err)

	db, err := badger.New(path.Join(tmpDir, "data"), path.Join(tmpDir, "meta"))
	if err != nil {
		require.NoError(err)
	}

	clean := func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}

	require.NoError(err)
	api := NewNamespaceAPI(db)

	return api, clean
}
