package rest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zero-os/0-stor/store/rest/models"
)

func TestCreateNameSpace(t *testing.T) {
	url, db, clean := getTestAPI(t)
	defer clean()

	body := &bytes.Buffer{}
	ns := models.NamespaceCreate{Label: "mynamespace"}
	err := json.NewEncoder(body).Encode(ns)
	require.NoError(t, err)

	resp, err := http.Post(url+"/namespaces", "application/json", body)
	require.NoError(t, err)

	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	exists, err := db.Exists(ns.Key())
	require.NoError(t, err)
	assert.True(t, exists, "namespace doesn't exists")
}

func TestListNamespace(t *testing.T) {
	url, db, clean := getTestAPI(t)
	defer clean()

	// populate db with some namespace
	ns := models.NamespaceCreate{}
	for _, label := range []string{"namespace1", "namespace2", "namespace3"} {
		ns.Label = label
		b, err := ns.Encode()
		require.NoError(t, err)

		err = db.Set(ns.Key(), b)
		require.NoError(t, err)
	}

	resp, err := http.Get(url + "/namespaces")
	require.NoError(t, err)

	require.Equal(t, http.StatusOK, resp.StatusCode)
	result := []models.Namespace{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Equal(t, 3, len(result))
}

func TestGetNameSpace(t *testing.T) {
	url, db, clean := getTestAPI(t)
	defer clean()

	// populate db with some namespace
	ns := models.Namespace{
		NamespaceCreate: models.NamespaceCreate{Label: "namespace1"},
		SpaceAvailable:  100,
		SpaceUsed:       50,
	}
	b, err := ns.Encode()
	require.NoError(t, err)
	err = db.Set(ns.Key(), b)
	require.NoError(t, err)

	resp, err := http.Get(url + "/namespaces/namespace1")
	require.Equal(t, http.StatusOK, resp.StatusCode)

	result := models.Namespace{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err, "error decoding response")

	assert.EqualValues(t, ns, result)
}

func TestDeleteNameSpace(t *testing.T) {
	// TODO
	t.SkipNow()
}

func TestStatNamespace(t *testing.T) {
	// TODO
	t.SkipNow()
}
