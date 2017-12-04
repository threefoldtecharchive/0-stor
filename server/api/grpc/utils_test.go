package grpc

import (
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"

	jwtgo "github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/require"
	"github.com/zero-os/0-stor/server"
	dbp "github.com/zero-os/0-stor/server/db"
	"github.com/zero-os/0-stor/server/db/badger"
	"github.com/zero-os/0-stor/server/encoding"
	"github.com/zero-os/0-stor/stubs"
)

const (
	// path to testing public key
	testPubKeyPath = "../../../devcert/jwt_pub.pem"

	// path to testing private key
	testPrivKeyPath = "../../../devcert/jwt_key.pem"
)

const (
	// test organization
	organization = "testorg"

	// test namespace
	namespace = "testnamespace"

	// test label (full iyo namespacing)
	label = "testorg_0stor_testnamespace"
)

func getTestGRPCServer(t *testing.T, organization string) (*Server, stubs.IYOClient, func()) {
	tmpDir, err := ioutil.TempDir("", "0stortest")
	require.NoError(t, err)

	db, err := badger.New(path.Join(tmpDir, "data"), path.Join(tmpDir, "meta"))
	require.NoError(t, err)
	//verifier, err := getTestVerifier(testPubKeyPath)
	//require.NoError(t, err)
	server, err := New(db, nil, 4, 0)
	require.NoError(t, err)

	go func() {
		err := server.Listen("localhost:0")
		require.NoError(t, err)
	}()
	require.NoError(t, err, "server failed to start listening")

	jwtCreator, organization := getIYOClient(t, organization)

	clean := func() {
		fmt.Sprintln("clean called")
		server.Close()
		os.RemoveAll(tmpDir)
	}

	return server, jwtCreator, clean
}

// returns a jwt verifier from provided public key file
/*func getTestVerifier(pubKeyPath string) (*jwt.Verifier, error) {
	pubKey, err := ioutil.ReadFile(pubKeyPath)
	if err != nil {
		return nil, err
	}

	return jwt.NewVerifier(string(pubKey))
}*/

func getIYOClient(t testing.TB, organization string) (stubs.IYOClient, string) {
	b, err := ioutil.ReadFile(testPrivKeyPath)
	require.NoError(t, err)

	key, err := jwtgo.ParseECPrivateKeyFromPEM(b)
	require.NoError(t, err)

	jwtCreator, err := stubs.NewStubIYOClient(organization, key)
	require.NoError(t, err, "failed to create the stub IYO client")

	return jwtCreator, organization
}

// populateDB populates a db with 10 entries that have keys `testkey0` - `testkey9`
func populateDB(t *testing.T, label string, db dbp.DB) map[string][]byte {
	/*
		ns := server.Namespace{Label: []byte(label)}
		nsData, err := encoding.EncodeNamespace(ns)
		require.NoError(t, err)
		require.NotNil(t, nsData)
		err = db.Set(dbp.NamespaceKey([]byte(label)), nsData)
		require.NoError(t, err)
	*/

	bufList := make(map[string][]byte, 10)

	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("testkey%d", i)
		bufList[key] = make([]byte, 1024*1024)

		_, err := rand.Read(bufList[key])
		require.NoError(t, err)

		refList := []string{
			"user1", "user2",
		}

		data, err := encoding.EncodeObject(server.Object{Data: bufList[key]})
		require.NoError(t, err)
		require.NotNil(t, data)
		err = db.Set(dbp.DataKey([]byte(label), []byte(key)), data)
		require.NoError(t, err)

		refListData, err := encoding.EncodeReferenceList(refList)
		require.NoError(t, err)
		require.NotNil(t, refListData)
		err = db.Set(dbp.ReferenceListKey([]byte(label), []byte(key)), refListData)
		require.NoError(t, err)
	}

	return bufList
}
