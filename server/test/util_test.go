package test

import (
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"

	log "github.com/Sirupsen/logrus"
	jwtgo "github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/require"
	"github.com/zero-os/0-stor/server"
	"github.com/zero-os/0-stor/server/db"
	"github.com/zero-os/0-stor/server/jwt"
	"github.com/zero-os/0-stor/server/manager"
	"github.com/zero-os/0-stor/stubs"
)

const (
	// path to testing public key
	testPubKeyPath = "../../devcert/jwt_pub.pem"

	// path to testing private key
	testPrivKeyPath = "../../devcert/jwt_key.pem"
)

func TestMain(m *testing.M) {
	log.SetLevel(log.DebugLevel)
	os.Exit(m.Run())
}

func getTestGRPCServer(t *testing.T, organization string) (server.StoreServer, stubs.IYOClient, func()) {
	tmpDir, err := ioutil.TempDir("", "0stortest")
	require.NoError(t, err)

	verifier, err := getTestVerifier(testPubKeyPath)

	server, err := server.New(path.Join(tmpDir, "data"), path.Join(tmpDir, "meta"), verifier, 4)
	require.NoError(t, err)

	_, err = server.Listen("localhost:0")
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

func populateDB(t *testing.T, namespace string, db db.DB) map[string][]byte {
	nsMgr := manager.NewNamespaceManager(db)
	objMgr := manager.NewObjectManager(namespace, db)
	err := nsMgr.Create(namespace)
	require.NoError(t, err)

	bufList := make(map[string][]byte, 10)

	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("testkey%d", i)
		bufList[key] = make([]byte, 1024*1024)

		_, err = rand.Read(bufList[key])
		require.NoError(t, err)

		refList := []string{
			"user1", "user2",
		}

		err = objMgr.Set([]byte(key), bufList[key], refList)
		require.NoError(t, err)
	}

	return bufList
}
