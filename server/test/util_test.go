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

func TestMain(m *testing.M) {
	log.SetLevel(log.DebugLevel)
	os.Exit(m.Run())
}

func getTestGRPCServer(t *testing.T) (server.StoreServer, stubs.IYOClient, string, func()) {
	tmpDir, err := ioutil.TempDir("", "0stortest")
	require.NoError(t, err)

	server, err := server.NewGRPC(path.Join(tmpDir, "data"), path.Join(tmpDir, "meta"))
	require.NoError(t, err)

	_, err = server.Listen("localhost:0")
	require.NoError(t, err, "server failed to start listening")

	jwtCreater, organization := getIYOClient(t)

	clean := func() {
		fmt.Sprintln("clean called")
		server.Close()
		os.RemoveAll(tmpDir)
	}

	return server, jwtCreater, organization, clean
}

func getIYOClient(t testing.TB) (stubs.IYOClient, string) {
	pubKey, err := ioutil.ReadFile("../../devcert/jwt_pub.pem")
	require.NoError(t, err)
	jwt.SetJWTPublicKey(string(pubKey))

	b, err := ioutil.ReadFile("../../devcert/jwt_key.pem")
	require.NoError(t, err)

	key, err := jwtgo.ParseECPrivateKeyFromPEM(b)
	require.NoError(t, err)

	organization := "testorg"
	jwtCreater, err := stubs.NewStubIYOClient(organization, key)
	require.NoError(t, err, "failt to create MockJWTCreater")

	return jwtCreater, organization
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
