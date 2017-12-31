/*
 * Copyright (C) 2017-2018 GIG Technology NV and Contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package datastor

import (
	"io"
	"io/ioutil"
	"math"
	"strconv"
	"testing"
	"time"

	"github.com/zero-os/0-stor/client/itsyouonline"
	"github.com/zero-os/0-stor/stubs"

	jwtgo "github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/require"
)

const testPrivateKeyPath = "../../devcert/jwt_key.pem"

func TestJwtTokenGetterUsingIYOClient_ExplicitErrors(t *testing.T) {
	require := require.New(t)

	_, err := JWTTokenGetterUsingIYOClient("", nil)
	require.Error(err, "no organization or client given")
	_, err = JWTTokenGetterUsingIYOClient("", new(itsyouonline.Client))
	require.Error(err, "no organization given")
	_, err = JWTTokenGetterUsingIYOClient("foo", nil)
	require.Error(err, "no client given")
}

func TestIYOBasedJWTTokenGetter_GetLabel(t *testing.T) {
	require := require.New(t)

	jwtTokenGetter, err := JWTTokenGetterUsingIYOClient("foo", new(itsyouonline.Client))
	require.NoError(err)

	_, err = jwtTokenGetter.GetLabel("")
	require.Error(err, "no namespace given")

	label, err := jwtTokenGetter.GetLabel("bar")
	require.NoError(err)
	require.Equal("foo_0stor_bar", label)
}

func Test_IYO_JWT_TokenGetter(t *testing.T) {
	require := require.New(t)

	b, err := ioutil.ReadFile(testPrivateKeyPath)
	require.NoError(err)
	key, err := jwtgo.ParseECPrivateKeyFromPEM(b)
	require.NoError(err)

	jwtCreator, err := stubs.NewStubIYOClient("testorg", key)
	require.NoError(err, "failed to create the stub IYO client")

	tg := IYOBasedJWTTokenGetter{client: jwtCreator}
	token, err := tg.GetJWTToken("foo")
	require.NoError(err)
	require.NotEmpty(token)
}

func TestCachedJWTTokenGetter_ExplicitErrors(t *testing.T) {
	require := require.New(t)

	_, err := CachedJWTTokenGetter(nil, -1, -1)
	require.Error(err, "no JWTTokenGetter given")

	_, err = CachedJWTTokenGetter(new(stubJWTTokenGetter), int(math.MaxUint32+1), -1)
	require.Error(err, "invalid bucket count: too big")
}

func TestCachedJWTTokenGetter(t *testing.T) {
	require := require.New(t)

	getter := &stubJWTTokenGetter{TTL: time.Second}
	cachedGetter, err := CachedJWTTokenGetter(getter, -1, -1)
	require.NoError(err)

	getter.GetError = io.EOF

	_, err = cachedGetter.GetJWTToken("foo")
	require.Equal(io.EOF, err, "error while creating JWT token")

	getter.GetError = nil

	token, err := cachedGetter.GetJWTToken("foo")
	require.NoError(err, "no error while creating JWT token")
	require.NotEmpty(token)

	getter.GetError = io.EOF

	token, err = cachedGetter.GetJWTToken("foo")
	require.NoError(err, "using cached token, so no need to create new token, so no error")
	require.NotEmpty(token)

	_, err = cachedGetter.GetJWTToken("bar")
	require.Equal(io.EOF, err, "namespace from other bucket, not cached yet")

	token, err = cachedGetter.GetJWTToken("foo")
	require.NoError(err, "using cached token, so no need to create new token, so no error")
	require.NotEmpty(token)

	time.Sleep(time.Second)

	_, err = cachedGetter.GetJWTToken("foo")
	require.Equal(io.EOF, err, "token expired, need to create new one, error while that happened")

	getter.GetError = nil

	token, err = cachedGetter.GetJWTToken("foo")
	require.NoError(err, "no error while creating JWT token")
	require.NotEmpty(token)

	token, err = cachedGetter.GetJWTToken("bar")
	require.NoError(err, "no error while creating JWT token")
	require.NotEmpty(token)
}

// stubJWTTokenGetter is a stub-version of a JWTTokenGetter,
// developed to help test the JWTTokenGetter cache support
type stubJWTTokenGetter struct {
	// Time To Live in Seconds
	TTL time.Duration
	// GetError defines the optional error to return
	// when retrieving a token
	GetError error
}

// GetJWTToken implements JWTTokenGetter.GetJWTToken
func (jwt *stubJWTTokenGetter) GetJWTToken(namespace string) (string, error) {
	if jwt.GetError != nil {
		return "", jwt.GetError
	}
	exp := time.Now().Add(jwt.TTL).Unix()
	return strconv.FormatInt(exp, 10), nil
}

// GetLabel implements JWTTokenGetter.GetLabel
func (jwt *stubJWTTokenGetter) GetLabel(namespace string) (string, error) {
	return namespace, nil
}

// GetClaimsFromJWTToken implements JWTTokenGetter.GetClaimsFromJWTToken
func (jwt *stubJWTTokenGetter) GetClaimsFromJWTToken(token string) (map[string]interface{}, error) {
	exp, err := strconv.ParseInt(token, 10, 64)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"exp": float64(exp)}, nil
}

func init() {
	jwtTokenCacheMinTTLInSeconds = 0
}
