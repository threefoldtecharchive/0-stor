package rest

import (
	"encoding/json"
	"net/http"
	"testing"

	units "github.com/docker/go-units"
	"github.com/stretchr/testify/require"
	"github.com/zero-os/0-stor/store/rest/models"
)

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

	resp, err := http.Get(url + "/stats")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	result := models.StoreStat{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	require.EqualValues(t, 0, result.SizeAvailable)
	require.EqualValues(t, 0, result.SizeUsed)

	// stats are saved in bytes
	// but returned in MiB
	obj := models.StoreStat{
		SizeUsed:      20 * units.MiB,
		SizeAvailable: 20 * units.MiB,
	}

	b, err = obj.Encode()
	require.NoError(t, err)
	err = db.Set(obj.Key(), b)
	require.NoError(t, err)

	resp, err = http.Get(url + "/stats")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	result = models.StoreStat{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	require.EqualValues(t, 20, result.SizeAvailable)
	require.EqualValues(t, 20, result.SizeUsed)
}
