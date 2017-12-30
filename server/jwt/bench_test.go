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

package jwt

import (
	"io/ioutil"
	"testing"

	"github.com/zero-os/0-stor/client/itsyouonline"
	"github.com/zero-os/0-stor/stubs"

	jwtgo "github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/require"
)

func BenchmarkJWTCache(b *testing.B) {
	require := require.New(b)

	token := getTokenBench(require)
	v, err := getTestVerifier(true)
	require.NoError(err, "failed to create jwt verifier")
	verifier := v.(*Verifier)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_, err := verifier.getScopes(token)
		require.NoError(err, "getScopes failed")
	}
}

func BenchmarkJWTWithoutCache(b *testing.B) {
	require := require.New(b)

	token := getTokenBench(require)
	v, err := getTestVerifier(true)
	require.NoError(err, "failed to create jwt verifier")
	verifier := v.(*Verifier)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_, _, err := verifier.checkJWTGetScopes(token)
		if err != nil {
			require.NoError(err, "getScopes failed")
		}
	}
}

func getTokenBench(require *require.Assertions) string {
	b, err := ioutil.ReadFile("../../devcert/jwt_key.pem")
	require.NoError(err)

	key, err := jwtgo.ParseECPrivateKeyFromPEM(b)
	require.NoError(err)

	iyoCl, err := stubs.NewStubIYOClient("testorg", key)
	require.NoError(err)

	token, err := iyoCl.CreateJWT("mynamespace", itsyouonline.Permission{
		Read:   true,
		Write:  true,
		Delete: true,
	})
	require.NoError(err)

	return token
}
