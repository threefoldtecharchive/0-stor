package rest

import (
	"io/ioutil"
	"net/http/httptest"
	"os"
	"path"
	"testing"

	"github.com/gorilla/mux"

	"github.com/stretchr/testify/require"
	"github.com/zero-os/0-stor/store/config"
	"github.com/zero-os/0-stor/store/db"
	"github.com/zero-os/0-stor/store/db/badger"
	"github.com/justinas/alice"
)

func getTestAPI(t *testing.T, middlewares map[string]MiddlewareEntry) (string, db.DB, func()) {
	tmpDir, err := ioutil.TempDir("", "0stortest")
	require.NoError(t, err)

	db, err := badger.New(path.Join(tmpDir, "data"), path.Join(tmpDir, "meta"))
	if err != nil {
		require.NoError(t, err)
	}

	api := NamespacesAPI{db: db, config: config.Settings{}}

	r := mux.NewRouter()
	routes  := new(HttpRoutes).GetRoutes(api)

	for i, e := range routes{
		if entry, ok := middlewares[e.Path]; ok { // override provided paths
			routes[i].Middlewares = entry.Middlewares
		}else{ // By default in testing, no middlewares
			routes[i].Middlewares = []alice.Constructor{}
		}
	}

	NamespacesInterfaceRoutes(r, routes)
	srv := httptest.NewServer(r)

	clean := func() {
		os.RemoveAll(tmpDir)
		srv.Close()
	}
	return srv.URL, db, clean
}
