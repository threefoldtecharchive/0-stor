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

package client

import (
	"io/ioutil"
	"testing"

	"github.com/zero-os/0-stor/client/itsyouonline"
	"github.com/zero-os/0-stor/stubs"

	jwtgo "github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/require"
)

const testPrivateKeyPath = "../devcert/jwt_key.pem"

func TestJwtTokenGetterFromIYOClient_Panics(t *testing.T) {
	require := require.New(t)

	require.Panics(func() {
		jwtTokenGetterFromIYOClient("", nil)
	}, "no organization or client given")
	require.Panics(func() {
		jwtTokenGetterFromIYOClient("", new(itsyouonline.Client))
	}, "no organization given")
	require.Panics(func() {
		jwtTokenGetterFromIYOClient("foo", nil)
	}, "no client given")
}

func TestIYOJWTTokenGetter_GetLabel(t *testing.T) {
	require := require.New(t)

	jwtTokenGetter := jwtTokenGetterFromIYOClient("foo", new(itsyouonline.Client))
	require.NotNil(jwtTokenGetter)

	_, err := jwtTokenGetter.GetLabel("")
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

	tg := iyoJWTTokenGetter{client: jwtCreator}
	token, err := tg.GetJWTToken("foo")
	require.NoError(err)
	require.NotEmpty(token)
}
