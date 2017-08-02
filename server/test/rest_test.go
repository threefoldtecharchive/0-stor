package test

import (
	"io/ioutil"
	"path"
	"testing"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/zero-os/0-stor/server/db"
	"github.com/zero-os/0-stor/server/db/badger"
	"github.com/zero-os/0-stor/server/api/rest"
	"os"
	"net/http/httptest"
	"github.com/zero-os/0-stor/client/itsyouonline"
	"log"
)

func getTestAPI(t *testing.T) (string, db.DB, string, *itsyouonline.Client, map[string]itsyouonline.Permission, func()) {
	tmpDir, err := ioutil.TempDir("", "0stortest")
	require.NoError(t, err)

	db, err := badger.New(path.Join(tmpDir, "data"), path.Join(tmpDir, "meta"))
	if err != nil {
		require.NoError(t, err)
	}


	r := mux.NewRouter()
	api := rest.NewNamespaceAPI(db)
	rest.NamespacesInterfaceRoutes(r, api, db)
	srv := httptest.NewServer(r)

	clean := func() {
		os.RemoveAll(tmpDir)
		srv.Close()
	}

	iyoClientID := os.Getenv("iyo_client_id")
	iyoSecret := os.Getenv("iyo_secret")
	iyoOrganization := os.Getenv("iyo_organization")


	if iyoClientID == ""{
		log.Fatal("[iyo] Missing (iyo_client_id) environement variable")

	}

	if iyoSecret == ""{
		log.Fatal("[iyo] Missing (iyo_secret) environement variable")

	}

	if iyoOrganization == ""{
		log.Fatal("[iyo] Missing (iyo_organization) environement variable")

	}

	iyoClient := itsyouonline.NewClient(iyoOrganization, iyoClientID, iyoSecret)

	permissions := make(map[string]itsyouonline.Permission)

	permissions["read"] = itsyouonline.Permission{Read: true,}
	permissions["all"] = itsyouonline.Permission{Read: true,Write:true, Delete:true}
	permissions["write"] = itsyouonline.Permission{Write: true,}
	permissions["delete"] = itsyouonline.Permission{Delete: true,}

	return srv.URL, db, iyoOrganization, iyoClient, permissions, clean
}
