package main

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"net/http"
	log "github.com/Sirupsen/logrus"
	"net/http/httptest"
	"github.com/gorilla/mux"

	"bytes"
	"os"
)


var config *settings
var db *Badger
var router *mux.Router


func SetUp(){
	// clean

	config = &settings{}
	config.Dirs.Data = "/tmp/data"
	config.Dirs.Meta = "/tmp/meta"

	err := os.RemoveAll(config.Dirs.Data)

	if err != nil{
		log.Errorln(err.Error())
	}

	err = os.RemoveAll(config.Dirs.Meta)

	if err != nil{
		log.Errorln(err.Error())
	}


	router =  mux.NewRouter()

	badger, err := NewBadger(config.Dirs.Meta, config.Dirs.Data)

	if err != nil{
		log.Errorln(err.Error())
	}

	db = badger

	api := NamespacesAPI{db:db, config: config}

	router.HandleFunc("/namespaces/", api.Createnamespace).Methods("POST")
}



func TestNamespaceValidation(t *testing.T) {
	tt := []struct {
		input NamespaceCreate
		valid bool
	}{
		{
			input: NamespaceCreate{Label: "valid"},
			valid: true,
		},
		{
			input: NamespaceCreate{Label: "with:colon"},
			valid: false,
		},
		{
			input: NamespaceCreate{Label: "with_underscore"},
			valid: false,
		},
	}

	for _, test := range tt {
		t.Run(test.input.Label, func(t *testing.T) {
			err := test.input.Validate()
			assert.Equal(t, test.valid, err == nil)
		})
	}
}

func TestCreatenamespace(t *testing.T) {
	payload := []byte(`{"name":"test product","price":11.22}`)
	req, err := http.NewRequest("POST", "/namespaces/", bytes.NewBuffer(payload))
	if err != nil {
		t.Fatal(err)
	}
	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, rr.Code, 400)

	// Invalid name (400 Bad request)

	payload = []byte(`{
	  "label": "namespace:1",
	  "acl": [
	    {
	      "id": "normalUser",
	      "acl": {
		"read": true,
		"write": true,
		"delete": false,
		"admin": false
	      }
	    },
	    {
	      "id": "admin",
	      "acl": {
		"read": true,
		"write": true,
		"delete": true,
		"admin": true
	      }
	    }
	  ]
	}`)

	req, err = http.NewRequest("POST", "/namespaces/", bytes.NewBuffer(payload))
	if err != nil {
		t.Fatal(err)
	}
	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, rr.Code, 400)

	// success

	payload = []byte(`{
	  "label": "namespace1",
	  "acl": [
	    {
	      "id": "normalUser",
	      "acl": {
		"read": true,
		"write": true,
		"delete": false,
		"admin": false
	      }
	    },
	    {
	      "id": "admin",
	      "acl": {
		"read": true,
		"write": true,
		"delete": true,
		"admin": true
	      }
	    }
	  ]
	}`)

	req, err = http.NewRequest("POST", "/namespaces/", bytes.NewBuffer(payload))
	if err != nil {
		t.Fatal(err)
	}
	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	assert.Equal(t, rr.Code, 201)

	// conflict if namespace exists
	payload = []byte(`{
	  "label": "namespace1",
	  "acl": [
	    {
	      "id": "normalUser",
	      "acl": {
		"read": true,
		"write": true,
		"delete": false,
		"admin": false
	      }
	    },
	    {
	      "id": "admin",
	      "acl": {
		"read": true,
		"write": true,
		"delete": true,
		"admin": true
	      }
	    }
	  ]
	}`)

	req, err = http.NewRequest("POST", "/namespaces/", bytes.NewBuffer(payload))
	if err != nil {
		t.Fatal(err)
	}
	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	assert.Equal(t, rr.Code, 409)
	assert.Equal(t, rr.Body.String(), "Namespace already exists\n")
}

func TestMain(m *testing.M) {
	SetUp()
	code := m.Run()
	os.Exit(code)
}
