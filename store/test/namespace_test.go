package test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zero-os/0-stor/store/rest"
)


func TestGetNameSpace(t *testing.T) {
	url, db, clean := getTestAPI(t)
	defer clean()

	exists, err := db.Exists([]byte("namespace1"))
	require.NoError(t, err)
	assert.False(t, exists)

	resp, err := http.Get(url + "/namespaces/namespace1")
	require.Equal(t, http.StatusOK, resp.StatusCode)

	exists, err = db.Exists([]byte("namespace1"))
	require.NoError(t, err)
	assert.True(t, exists)


	result := rest.Namespace{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err, "error decoding response")

	assert.Equal(t, result.Label, "namespace1")
	assert.Equal(t, result.Stats.NrObjects, int64(0))
}