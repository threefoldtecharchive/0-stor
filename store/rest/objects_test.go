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
	url, db, clean := getTestAPI(t, map[string]MiddlewareEntry{})
	defer clean()

	// create namespace
	body := &bytes.Buffer{}
	ns := models.NamespaceCreate{Label: "mynamespace"}
	err := json.NewEncoder(body).Encode(ns)
	require.NoError(t, err)

	resp, err := http.Post(url+"/namespaces", "application/json", body)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

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
	//
	//// TODO, more test on the content of the obj
	//require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	//f, err := obj.ToFile("mynamespace")
	//require.Error(t, err, "File contents < 32 bytes. There should be an error")

	// Trying to create valid object.
	body = &bytes.Buffer{}
	obj := models.Object{
		Data: "********************************abcdef",
		Id:   "myobject",
	}
	err = json.NewEncoder(body).Encode(obj)
	require.NoError(t, err)
	resp, err = http.Post(url+"/namespaces/mynamespace/objects", "application/json", body)
	require.NoError(t, err)

	require.Equal(t, http.StatusCreated, resp.StatusCode)
	f, err := obj.ToFile("mynamespace")
	require.NoError(t, err)
	exists, err := db.Exists(f.Key())
	require.NoError(t, err)
	assert.True(t, exists)

	// Try to add same file id, different contents, we should get conflict
	body = &bytes.Buffer{}
	obj = models.Object{
		Data: "********%%%%%******abcdef",
		Id:   "myobject",
	}
	err = json.NewEncoder(body).Encode(obj)
	require.NoError(t, err)
	resp, err = http.Post(url+"/namespaces/mynamespace/objects", "application/json", body)
	require.NoError(t, err)

	require.Equal(t, http.StatusConflict, resp.StatusCode)
	f, err = obj.ToFile("mynamespace")
	require.NoError(t, err)
	exists, err = db.Exists(f.Key())
	require.NoError(t, err)
	assert.True(t, exists)

	//


	// Trying to add same file, reference is incremented by 1
	body = &bytes.Buffer{}
	obj = models.Object{
		Data: "********************************abcdef",
		Id:   "myobject",
	}
	err = json.NewEncoder(body).Encode(obj)
	require.NoError(t, err)
	resp, err = http.Post(url+"/namespaces/mynamespace/objects", "application/json", body)
	require.NoError(t, err)

	require.Equal(t, http.StatusCreated, resp.StatusCode)
	f, err = obj.ToFile("mynamespace")
	require.NoError(t, err)
	exists, err = db.Exists(f.Key())
	require.NoError(t, err)
	assert.True(t, exists)

	b, err := db.Get(f.Key())
	require.NoError(t, err)

	f = &models.File{}
	err = f.Decode(b)
	require.NoError(t, err)

	require.Equal(t, f.Reference, byte(2))

	// reference can't exceed 255

	for i:=1; i<260; i++{
		body = &bytes.Buffer{}
		obj = models.Object{
			Data: "********************************abcdef",
			Id:   "myobject",
		}
		err = json.NewEncoder(body).Encode(obj)
		require.NoError(t, err)
		resp, err = http.Post(url+"/namespaces/mynamespace/objects", "application/json", body)
		require.NoError(t, err)

		require.Equal(t, http.StatusCreated, resp.StatusCode)
	}

	f, err = obj.ToFile("mynamespace")
	require.NoError(t, err)
	exists, err = db.Exists(f.Key())
	require.NoError(t, err)
	assert.True(t, exists)

	b, err = db.Get(f.Key())
	require.NoError(t, err)

	f = &models.File{}
	err = f.Decode(b)
	require.NoError(t, err)

	require.Equal(t, f.Reference, byte(255))
}

func TestGetObject(t *testing.T){
	url, db, clean := getTestAPI(t, map[string]MiddlewareEntry{})
	defer clean()

	// create namespace
	body := &bytes.Buffer{}
	ns := models.NamespaceCreate{Label: "mynamespace"}
	err := json.NewEncoder(body).Encode(ns)
	require.NoError(t, err)

	resp, err := http.Post(url+"/namespaces", "application/json", body)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// create an object.

	obj := models.Object{
		Data: "********************************abcdef",
		Id:   "myobject",
	}

	f , err := obj.ToFile("mynamespace")
	require.NoError(t, err)

	b, err := f.Encode()
	require.NoError(t, err)

	err = db.Set("mynamespace:myobject", b)

	body = &bytes.Buffer{}
	resp, err = http.Get(url+"/namespaces/mynamespace/objects/myobject")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	result := models.Object{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	require.Equal(t, result.Id, "myobject")
	require.Equal(t, result.Data, obj.Data)

	resp, err = http.Get(url+"/namespaces/mynamespace/objects/myobject2")
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}


func TestHeadObject(t *testing.T){
	url, db, clean := getTestAPI(t, map[string]MiddlewareEntry{})
	defer clean()

	// create namespace
	body := &bytes.Buffer{}
	ns := models.NamespaceCreate{Label: "mynamespace"}
	err := json.NewEncoder(body).Encode(ns)
	require.NoError(t, err)

	resp, err := http.Post(url+"/namespaces", "application/json", body)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// create an object.

	obj := models.Object{
		Data: "********************************abcdef",
		Id:   "myobject",
	}

	f , err := obj.ToFile("mynamespace")
	require.NoError(t, err)

	b, err := f.Encode()
	require.NoError(t, err)

	err = db.Set("mynamespace:myobject", b)

	body = &bytes.Buffer{}
	resp, err = http.Head(url+"/namespaces/mynamespace/objects/myobject")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	resp, err = http.Get(url+"/namespaces/mynamespace/objects/myobject2")
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestListObjects(t *testing.T){
	url, db, clean := getTestAPI(t, map[string]MiddlewareEntry{})
	defer clean()

	ns := models.NamespaceCreate{}
	for _, label := range []string{"namespace1", "namespace2", "namespace3"} {
		ns.Label = label
		b, err := ns.Encode()
		require.NoError(t, err)

		err = db.Set(ns.Key(), b)
		require.NoError(t, err)
	}

	// create some objects in namespace1
	for _, label := range []string{"obj1", "obj2", "obj3"} {
		f := models.File{
			Namespace: "namespace1",
			Payload: "abcd",
			Id:   label,
			Reference: 1,
		}

		b, err := f.Encode()
		require.NoError(t, err)

		err = db.Set(f.Key(), b)
		require.NoError(t, err)
	}

	resp, err := http.Get(url+"/namespaces/namespace1/objects")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	result := []models.Object{}
	err = json.NewDecoder(resp.Body).Decode(&result)

	require.Equal(t, len(result), 3)

	for _, obj := range result{
		if obj.Id == "obj1"{
			require.Equal(t, obj.Data, "abcd")
		}else if obj.Id == "obj2"{
			require.Equal(t, obj.Data, "abcd")
		}else if obj.Id == "obj3"{
			require.Equal(t, obj.Data, "abcd")
		}else{
			assert.Fail(t, "Invalid object name")
		}
	}

	resp, err = http.Get(url+"/namespaces/namespace2/objects")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	result = []models.Object{}
	err = json.NewDecoder(resp.Body).Decode(&result)

	require.Equal(t, len(result), 0)

	resp, err = http.Get(url+"/namespaces/namespace3/objects")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	result = []models.Object{}
	err = json.NewDecoder(resp.Body).Decode(&result)

	require.Equal(t, len(result), 0)
}

func TestDeleteObject(t *testing.T){
	url, db, clean := getTestAPI(t, map[string]MiddlewareEntry{})
	defer clean()

	ns := models.NamespaceCreate{Label: "namespace1"}
	b, err := ns.Encode()
	require.NoError(t, err)

	err = db.Set(ns.Key(), b)
	require.NoError(t, err)

	// create object

	f := models.File{
		Namespace: "namespace1",
		Payload: "Hello world!",
		Id:   "object1",
		Reference: 2,
	}

	b, err = f.Encode()
	require.NoError(t, err)

	err = db.Set(f.Key(), b)
	require.NoError(t, err)

	req, err := http.NewRequest("DELETE", url+"/namespaces/namespace1/objects/object1",  &bytes.Buffer{})
	req.Header.Add("Content-Type", "application/json")
	require.NoError(t, err)
	cli := http.DefaultClient

	resp, err := cli.Do(req)

	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	v, err := db.Get(f.Key())
	require.NoError(t, err)

	f = models.File{}

	err = f.Decode(v)
	require.NoError(t, err)

	require.Equal(t, f.Reference, byte(1))
	require.Equal(t, f.Namespace, "namespace1")
	require.Equal(t, f.Id, "object1")

	// Now actually delete file

	req, err = http.NewRequest("DELETE", url+"/namespaces/namespace1/objects/object1",  &bytes.Buffer{})
	req.Header.Add("Content-Type", "application/json")
	require.NoError(t, err)
	cli = http.DefaultClient

	resp, err = cli.Do(req)

	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	exists, err := db.Exists(f.Key())
	require.NoError(t, err)
	require.False(t, exists)

	// Now 404
	req, err = http.NewRequest("DELETE", url+"/namespaces/namespace1/objects/object1",  &bytes.Buffer{})
	req.Header.Add("Content-Type", "application/json")
	require.NoError(t, err)
	cli = http.DefaultClient

	resp, err = cli.Do(req)

	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}