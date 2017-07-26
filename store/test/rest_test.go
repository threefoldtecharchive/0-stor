package test

import (
	"io/ioutil"
	"path"
	"testing"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/zero-os/0-stor/store/db"
	"github.com/zero-os/0-stor/store/db/badger"
	"github.com/zero-os/0-stor/store/rest"
	"os"
	"net/http/httptest"
)

func getTestAPI(t *testing.T) (string, db.DB, func()) {
	tmpDir, err := ioutil.TempDir("", "0stortest")
	require.NoError(t, err)

	db, err := badger.New(path.Join(tmpDir, "data"), path.Join(tmpDir, "meta"))
	if err != nil {
		require.NoError(t, err)
	}


	r := mux.NewRouter()
	api := rest.NewNamespaceAPI(db)
	rest.NamespacesInterfaceRoutes(r, api)
	srv := httptest.NewServer(r)

	clean := func() {
		os.RemoveAll(tmpDir)
		srv.Close()
	}

	return srv.URL, db, clean
}
