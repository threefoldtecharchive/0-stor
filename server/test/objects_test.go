package test

import (
	"testing"
	"bytes"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/assert"
	"github.com/zero-os/0-stor/server/api/rest"
	"net/http"
	"github.com/zero-os/0-stor/server/db"
	"fmt"
	"sort"
)

func TestCreateObject(t *testing.T) {
	url, db, clean := getTestAPI(t)
	defer clean()

	// Trying to create invalid object. (size > 1mB)
	//@TODO: FIXME this gives an error in Httpserver
	//2017/07/17 17:29:37 httptest.Server blocked in Close after 5 seconds, waiting for connections:
	//*net.TCPConn 0xc424fac0b8 127.0.0.1:34030 in state active


	//body = &bytes.Buffer{}
	//data := make([]byte, 1025*1024)
	//obj := models.Object{
	//	Data: string(data[:]),
	//	Id:   "myobject",
	//}
	//err = json.NewEncoder(body).Encode(obj)
	//require.NoError(t, err)
	//resp, err = http.Post(url+"/namespaces/mynamespace/objects", "application/json", body)
	//require.NoError(t, err)

	// Trying to create valid object.
	body := &bytes.Buffer{}
	obj := rest.Object{
		Data: "********************************abcdef",
		Id:   "myobject",
		ReferenceList:[]rest.ReferenceID{
			"ref1",
		},
	}
	err := json.NewEncoder(body).Encode(obj)
	require.NoError(t, err)
	resp, err := http.Post(url+"/namespaces/mynamespace/objects", "application/json", body)
	require.NoError(t, err)

	require.Equal(t, http.StatusCreated, resp.StatusCode)
	exists, err := db.Exists([]byte("mynamespace:myobject"))
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestGetObject(t *testing.T){
	url, database, clean := getTestAPI(t)
	defer clean()

	// create an object.

	obj := db.Object{
		Data: []byte("********************************abcdef"),
	}

	b, err := obj.Encode()
	require.NoError(t, err)

	err = database.Set([]byte("mynamespace:myobject"), b)

	resp, err := http.Get(url+"/namespaces/mynamespace/objects/myobject")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	result := rest.Object{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	require.Equal(t, result.Id, "myobject")
	require.Equal(t, result.Data, string(obj.Data))

	resp, err = http.Get(url+"/namespaces/mynamespace/objects/myobject2")
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestListObjects(t *testing.T){
	url, database, clean := getTestAPI(t)
	defer clean()

	objects := []string{"obj1", "obj2", "obj3"}
	for _, label := range objects {
		key := fmt.Sprintf("%s:%s", "namespace", label)

		obj := db.Object{
			Data: []byte("********************************abcdef"),
		}

		b, err := obj.Encode()
		require.NoError(t, err)

		err = database.Set([]byte(key), b)
	}


	resp, err := http.Get(url+"/namespaces/namespace/objects")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	result := []string{}
	err = json.NewDecoder(resp.Body).Decode(&result)

	require.Equal(t, len(result), 3)
	sort.Strings(result)
	require.Equal(t, result, objects)
}

func TestDeleteObject(t *testing.T){
	url, database, clean := getTestAPI(t)
	defer clean()

	// create an object.

	obj := db.Object{
		Data: []byte("********************************abcdef"),
	}

	b, err := obj.Encode()
	require.NoError(t, err)

	err = database.Set([]byte("mynamespace:myobject"), b)

	resp, err := http.Get(url+"/namespaces/mynamespace/objects/myobject")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	req, err := http.NewRequest("DELETE", url+"/namespaces/mynamespace/objects/myobject",  &bytes.Buffer{})
	req.Header.Add("Content-Type", "application/json")
	require.NoError(t, err)
	cli := http.DefaultClient

	resp, err = cli.Do(req)

	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	// No actual delete yet

	//exists, err := database.Exists([]byte("mynamespace:myobject"))
	//require.NoError(t, err)
	//require.False(t, exists)
}