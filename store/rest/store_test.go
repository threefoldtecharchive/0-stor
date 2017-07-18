package rest

import (
	"testing"
	"bytes"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"net/http"
	"github.com/zero-os/0-stor/store/rest/models"
)

func TestUpdateStoreStats(t *testing.T) {
	url, db, clean := getTestAPI(t, map[string]MiddlewareEntry{})
	defer clean()

	// create store stats
	ss := models.StoreStat{}
	b, err := ss.Encode()
	require.NoError(t, err)

	err = db.Set(ss.Key(), b)
	require.NoError(t, err)

	// Now we're in business

	body := &bytes.Buffer{}

	statsReq := models.StoreStatRequest{SizeAvailable: 10.0}

	err = json.NewEncoder(body).Encode(statsReq)
	require.NoError(t, err)

	resp, err := http.Post(url+"/namespaces/stats", "application/json", body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	result := models.StoreStatRequest{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	require.Equal(t, result.SizeAvailable, 10.0)

	dbResult :=  models.StoreStat{}
	b, err = db.Get(models.StoreStat{}.Key())
	require.NoError(t, err)
	err = dbResult.Decode(b)
	require.NoError(t, err)

	require.Equal(t, dbResult.SizeAvailable, 10.0)
	require.Equal(t, dbResult.SizeUsed, 0.0)

	dbResult.SizeUsed = 10.0
	b, err = dbResult.Encode()
	require.NoError(t, err)
	err = db.Set(dbResult.Key(), b)
	require.NoError(t, err)

	// Trying to set available size < used size (not applicable)

	statsReq = models.StoreStatRequest{SizeAvailable: 9.0}

	err = json.NewEncoder(body).Encode(statsReq)
	require.NoError(t, err)

	resp, err = http.Post(url+"/namespaces/stats", "application/json", body)
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	dbResult =  models.StoreStat{}
	b, err = db.Get(models.StoreStat{}.Key())
	require.NoError(t, err)
	err = dbResult.Decode(b)
	require.NoError(t, err)

	require.Equal(t, dbResult.SizeAvailable, 10.0)
	require.Equal(t, dbResult.SizeUsed, 10.0)
}

func TestGetStoreStats(t *testing.T) {
	url, db, clean := getTestAPI(t, map[string]MiddlewareEntry{})
	defer clean()

	// create store stats
	ss := models.StoreStat{}
	b, err := ss.Encode()
	require.NoError(t, err)

	err = db.Set(ss.Key(), b)
	require.NoError(t, err)

	// Now we're in business

	resp, err := http.Get(url+"/namespaces/stats")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	result := models.StoreStat{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	require.Equal(t, result.SizeAvailable, 0.0)
	require.Equal(t, result.SizeUsed, 0.0)

	obj :=  models.StoreStat{
		SizeUsed: 20,
		StoreStatRequest: models.StoreStatRequest{
			SizeAvailable: 20,
		},
	}

	b, err = obj.Encode()
	require.NoError(t, err)
	err = db.Set(obj.Key(), b)
	require.NoError(t, err)

	resp, err = http.Get(url+"/namespaces/stats")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	result = models.StoreStat{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	require.Equal(t, result.SizeAvailable, 20.0)
	require.Equal(t, result.SizeUsed, 20.0)
}