package rest

//
// import (
// 	"bytes"
// 	"io/ioutil"
// 	"net/http"
// 	"net/http/httptest"
// 	"os"
// 	"testing"
//
// 	log "github.com/Sirupsen/logrus"
// 	"github.com/gorilla/mux"
// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/require"
// 	"github.com/zero-os/0-stor/store/config"
// )
//
// var router *mux.Router
//
// func SetUp(t *testing.T) {
// 	// clean
//
// 	cfg := &config.Settings{}
// 	tmp, err := ioutil.TempDir("", "0stortest")
// 	require.NoError(t, err)
// 	cfg.DB.Dirs.Data = data
//
// 	tmp, err := ioutil.TempDir("", "0stortest")
// 	require.NoError(t, err)
// 	cfg.DB.Dirs.Meta = tmp
//
// 	db, err = NewBadger(config.Dirs.Meta, config.Dirs.Data)
// 	if err != nil {
// 		log.Errorln(err.Error())
// 	}
//
// 	api := NamespacesAPI{db: db, config: config}
//
// 	router = mux.NewRouter()
// 	router.HandleFunc("/namespaces/", api.Createnamespace).Methods("POST")
// 	router.HandleFunc("/namespaces/{nsid}", api.Getnsid).Methods("GET")
//
// }
//
// func TearDown() {
// 	os.RemoveAll(cfg.DB.Dirs.Data)
// 	os.RemoveAll(cfg.DB.Dirs.Meta)
// }
//
// func TestNamespaceValidation(t *testing.T) {
// 	tt := []struct {
// 		input NamespaceCreate
// 		valid bool
// 	}{
// 		{
// 			input: NamespaceCreate{Label: "valid"},
// 			valid: true,
// 		},
// 		{
// 			input: NamespaceCreate{Label: "with:colon"},
// 			valid: false,
// 		},
// 		{
// 			input: NamespaceCreate{Label: "with_underscore"},
// 			valid: false,
// 		},
// 	}
//
// 	for _, test := range tt {
// 		t.Run(test.input.Label, func(t *testing.T) {
// 			err := test.input.Validate()
// 			assert.Equal(t, test.valid, err == nil)
// 		})
// 	}
// }
//
// func TestCreateAndGetNamespace(t *testing.T) {
// 	// Get non existing namespace (404) expected
// 	req, err := http.NewRequest("GET", "/namespaces/namespace1", nil)
// 	require.Nil(t, err)
//
// 	rr := httptest.NewRecorder()
// 	router.ServeHTTP(rr, req)
//
// 	assert.Equal(t, rr.Code, 404)
//
// 	payload := []byte(`{"name":"test product","price":11.22}`)
// 	req, err = http.NewRequest("POST", "/namespaces/", bytes.NewBuffer(payload))
// 	require.Nil(t, err)
//
// 	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
// 	rr = httptest.NewRecorder()
// 	router.ServeHTTP(rr, req)
//
// 	assert.Equal(t, rr.Code, 400)
//
// 	// Invalid name (400 Bad request)
//
// 	payload = []byte(`{
// 	  "label": "namespace:1",
// 	  "acl": [
// 	    {
// 	      "id": "normalUser",
// 	      "acl": {
// 		"read": true,
// 		"write": true,
// 		"delete": false,
// 		"admin": false
// 	      }
// 	    },
// 	    {
// 	      "id": "admin",
// 	      "acl": {
// 		"read": true,
// 		"write": true,
// 		"delete": true,
// 		"admin": true
// 	      }
// 	    }
// 	  ]
// 	}`)
//
// 	req, err = http.NewRequest("POST", "/namespaces/", bytes.NewBuffer(payload))
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
// 	rr = httptest.NewRecorder()
// 	router.ServeHTTP(rr, req)
//
// 	assert.Equal(t, rr.Code, 400)
//
// 	// success
//
// 	payloadJson := `{
// 	  "label": "namespace1",
// 	  "acl": [
// 	    {
// 	      "id": "normalUser",
// 	      "acl": {
// 		"read": true,
// 		"write": true,
// 		"delete": false,
// 		"admin": false
// 	      }
// 	    },
// 	    {
// 	      "id": "admin",
// 	      "acl": {
// 		"read": true,
// 		"write": true,
// 		"delete": true,
// 		"admin": true
// 	      }
// 	    }
// 	  ]
// 	}`
// 	payload = []byte(payloadJson)
//
// 	req, err = http.NewRequest("POST", "/namespaces/", bytes.NewBuffer(payload))
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
// 	rr = httptest.NewRecorder()
// 	router.ServeHTTP(rr, req)
// 	assert.Equal(t, rr.Code, 201)
//
// 	// conflict if namespace exists (using old payload)
//
// 	req, err = http.NewRequest("POST", "/namespaces/", bytes.NewBuffer(payload))
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
// 	rr = httptest.NewRecorder()
// 	router.ServeHTTP(rr, req)
// 	assert.Equal(t, rr.Code, 409)
// 	assert.Equal(t, rr.Body.String(), "Namespace already exists\n")
//
// 	req2, err2 := http.NewRequest("GET", "/namespaces/namespace1", nil)
// 	require.Nil(t, err2)
//
// 	rr2 := httptest.NewRecorder()
// 	router.ServeHTTP(rr2, req2)
//
// 	assert.Equal(t, rr2.Code, 200)
// 	log.Println(rr2.Body.String())
// }
//
// func TestMain(m *testing.M) {
// 	SetUp()
// 	code := m.Run()
// 	TearDown()
// 	os.Exit(code)
// }
