package rest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zero-os/0-stor/store/rest/models"
	"time"
)

func TestCreateNameSpace(t *testing.T) {
	url, db, clean := getTestAPI(t, map[string]MiddlewareEntry{})
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
	url, db, clean := getTestAPI(t, map[string]MiddlewareEntry{})
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
	url, db, clean := getTestAPI(t, map[string]MiddlewareEntry{})
	defer clean()

	// populate db with some namespace
	ns := models.Namespace{
		NamespaceCreate: models.NamespaceCreate{Label: "namespace1"},
		SpaceAvailable:  0,
		SpaceUsed:       0,
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
	url, db, clean := getTestAPI(t, map[string]MiddlewareEntry{})
	defer clean()

	body := &bytes.Buffer{}
	ns := models.NamespaceCreate{Label: "mynamespace"}
	err := json.NewEncoder(body).Encode(ns)
	require.NoError(t, err)

	// create namespace!
	_, err = http.Post(url+"/namespaces", "application/json", body)
	require.NoError(t, err)

	//Sleep to allow creation of namespace stats
	time.Sleep(time.Second)

	// Add some objects
	for _, label := range []string{"obj1", "obj2", "obj3"} {
		f := models.File{}
		f.Namespace = "mynamespace"
		f.Id = label
		f.Payload = "hello world!"
		f.CRC = "crcrccccccccc"
		f.Tags = []models.Tag{}
		b, err := f.Encode()
		require.NoError(t, err)
		err = db.Set(f.Key(), b)
		require.NoError(t, err)
	}

	// all files exist
	for _, label := range []string{"obj1", "obj2", "obj3"} {
		f := models.File{}
		f.Namespace = "mynamespace"
		f.Id = label
		exists, err := db.Exists(f.Key())
		require.NoError(t, err)
		assert.True(t, exists, "file does not exist")
	}

	req, err := http.NewRequest("DELETE", url+"/namespaces/mynamespace",  &bytes.Buffer{})
	req.Header.Add("Content-Type", "application/json")
	require.NoError(t, err)
	cli := http.DefaultClient

	resp, err := cli.Do(req)

	//Sleep to allow deletion of related collections
	time.Sleep(time.Second)

	assert.Equal(t, resp.StatusCode, http.StatusNoContent)
	exists, err := db.Exists(ns.Key())
	require.NoError(t, err)
	assert.False(t, exists, "namespace exists")

	nss := models.NamespaceStats{
		Namespace: "mynamespace",
	}

	exists, err = db.Exists(nss.Key())
	require.NoError(t, err)
	assert.False(t, exists, "namespace stats exists")

	// no files exist
	for _, label := range []string{"obj1", "obj2", "obj3"} {
		f := models.File{}
		f.Namespace = "mynamespace"
		f.Id = label
		exists, err = db.Exists(f.Key())
		require.NoError(t, err)
		assert.False(t, exists, "file exists")
	}

}

func TestStatNamespace(t *testing.T) {
	url, db, clean := getTestAPI(t, map[string]MiddlewareEntry{})
	defer clean()

	body := &bytes.Buffer{}
	ns := models.NamespaceCreate{Label: "mynamespace"}
	err := json.NewEncoder(body).Encode(ns)
	require.NoError(t, err)

	_, err = http.Post(url+"/namespaces", "application/json", body)
	require.NoError(t, err)

	//Sleep to allow creation of namespace stats
	time.Sleep(time.Second)

	expected := models.NamespaceStats{
		Namespace: "mynamespace",
		TotalSizeReserved:0,
		NrRequests:0,
	}

	result := models.NamespaceStats{}

	b, err := db.Get(expected.Key())
	require.NoError(t, err)
	err = result.Decode(b)
	require.NoError(t, err)


	expected.Created = result.Created
	expected.Namespace = ""

	assert.EqualValues(t, expected, result)
}


func TestUpdateACLs(t *testing.T){
	url, db, clean := getTestAPI(t, map[string]MiddlewareEntry{})
	defer clean()

	ns := models.NamespaceCreate{
		Label: "namespace1",
		Acl: []models.ACL{
			models.ACL{
				Acl: models.ACLEntry{
					Delete: true,
					Read:true,
					Write: true,
					Admin: true,
				},
				Id: "hamdy",
			},

			models.ACL{
				Acl: models.ACLEntry{
					Delete: true,
					Read:true,
					Write: true,
					Admin: true,
				},
				Id: "zaibon",
			},

		},
	}

	b, err := ns.Encode()
	require.NoError(t, err)
	err = db.Set(ns.Key(), b)
	require.NoError(t, err)

	b, err = db.Get(ns.Key())
	require.NoError(t, err)
	err = ns.Decode(b)
	require.NoError(t, err)
	assert.Equal(t, len(ns.Acl), 2)

	body := &bytes.Buffer{}

	acl := models.ACL{
		Id: "hamdy",
		Acl: models.ACLEntry{
			Read: true,
			Write: false,
			Delete: false,
			Admin: false,
		},
	}

	err = json.NewEncoder(body).Encode(acl)
	require.NoError(t, err)

	resp, err := http.Post(url+"/namespaces/namespace1/acl", "application/json", body)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	b, err = db.Get(ns.Key())
	require.NoError(t, err)
	err = ns.Decode(b)
	require.NoError(t, err)
	assert.Equal(t, len(ns.Acl), 2)

	for _, v := range ns.Acl{
		if v.Id == "hamdy"{
			assert.False(t, v.Acl.Delete)
			assert.False(t, v.Acl.Write)
			assert.False(t, v.Acl.Admin)
			assert.True(t, v.Acl.Read)


		}else if v.Id == "zaibon" {
			assert.True(t, v.Acl.Read)
			assert.True(t, v.Acl.Write)
			assert.True(t, v.Acl.Admin)
			assert.True(t, v.Acl.Delete)
		}else{
			assert.Fail(t, "Invalid Id")
		}
	}
}

