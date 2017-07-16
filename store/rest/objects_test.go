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

func TestCreateObject(t *testing.T) {
	url, db, clean := getTestAPI(t)
	defer clean()

	// create namespace
	body := &bytes.Buffer{}
	ns := models.NamespaceCreate{Label: "mynamespace"}
	err := json.NewEncoder(body).Encode(ns)
	require.NoError(t, err)

	resp, err := http.Post(url+"/namespaces", "application/json", body)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// Trying to create invalid object.
	body = &bytes.Buffer{}
	obj := models.Object{
		Data: "hello world",
		Id:   "myobject",
	}
	err = json.NewEncoder(body).Encode(obj)
	require.NoError(t, err)
	resp, err = http.Post(url+"/namespaces/mynamespace/objects", "application/json", body)
	require.NoError(t, err)

	// TODO, more test on the content of the obj
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	f, err := obj.ToFile("mynamespace")
	require.Error(t, err, "File contents < 32 bytes. There should be an error")

	// Trying to create valid object.
	body = &bytes.Buffer{}
	obj = models.Object{
		Data: "********************************abcdef",
		Id:   "myobject",
	}
	err = json.NewEncoder(body).Encode(obj)
	require.NoError(t, err)
	resp, err = http.Post(url+"/namespaces/mynamespace/objects", "application/json", body)
	require.NoError(t, err)

	// TODO, more test on the content of the obj
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	f, err = obj.ToFile("mynamespace")
	require.NoError(t, err)
	exists, err := db.Exists(f.Key())
	require.NoError(t, err)
	assert.True(t, exists)

}
