package client

import (
	"io/ioutil"
	"testing"

	jwtgo "github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/require"
	"github.com/zero-os/0-stor/stubs"
)

const testPrivateKeyPath = "../devcert/jwt_key.pem"

func Test_IYO_JWT_TokenGetter(t *testing.T) {
	require := require.New(t)

	b, err := ioutil.ReadFile(testPrivateKeyPath)
	require.NoError(err)
	key, err := jwtgo.ParseECPrivateKeyFromPEM(b)
	require.NoError(err)

	jwtCreator, err := stubs.NewStubIYOClient("testorg", key)
	require.NoError(err, "failed to create the stub IYO client")

	tg := iyoJWTTokenGetter{client: jwtCreator}
	token, err := tg.GetJWTToken("foo")
	require.NoError(err)
	require.NotEmpty(token)
}
