package grpc

import (
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"testing"

	jwtgo "github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/require"
	"github.com/zero-os/0-stor/server"
	dbp "github.com/zero-os/0-stor/server/db"
	"github.com/zero-os/0-stor/server/db/memory"
	"github.com/zero-os/0-stor/server/encoding"
	"github.com/zero-os/0-stor/server/jwt"
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
	var client stubs.IYOClient
	var verifier jwt.TokenVerifier
	if organization != "" {
		client, organization = getIYOClient(t, organization)
		var err error
		verifier, err = getTestVerifier(testPubKeyPath)
		require.NoError(t, err)
	} else {
		verifier = jwt.NopVerifier{}
	}

	server, err := New(memory.New(), verifier, 4, 0)
	require.NoError(t, err)

	go func() {
		err := server.Listen("localhost:0")
		require.NoError(t, err)
	}()
	require.NoError(t, err, "server failed to start listening")

	clean := func() {
		fmt.Sprintln("clean called")
		server.Close()
	}
	return server, client, clean
}

func getTestVerifier(pubKeyPath string) (*jwt.Verifier, error) {
	pubKey, err := ioutil.ReadFile(pubKeyPath)
	if err != nil {
		return nil, err
	}
	return jwt.NewVerifier(string(pubKey))
}

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
	bufList := make(map[string][]byte, 10)

	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("testkey%d", i)
		bufList[key] = make([]byte, 32)

		_, err := rand.Read(bufList[key])
		require.NoError(t, err)

		data, err := encoding.EncodeObject(server.Object{Data: bufList[key]})
		require.NoError(t, err)
		require.NotNil(t, data)
		err = db.Set(dbp.DataKey([]byte(label), []byte(key)), data)
		require.NoError(t, err)
	}

	return bufList
}
