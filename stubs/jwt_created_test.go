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

package stubs

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/assert"
	"github.com/zero-os/0-stor/client/itsyouonline"
)

func TestCreateJWT(t *testing.T) {
	pubKey, err := ioutil.ReadFile("../devcert/jwt_pub.pem")
	require.NoError(t, err)

	b, err := ioutil.ReadFile("../devcert/jwt_key.pem")
	require.NoError(t, err)

	key, err := jwt.ParseECPrivateKeyFromPEM(b)
	require.NoError(t, err)

	iyoCl, err := NewStubIYOClient("testorg", key)
	assert.NoError(t, err)

	tokenString, err := iyoCl.CreateJWT("testns", itsyouonline.Permission{
		Admin:  true,
		Read:   true,
		Write:  true,
		Delete: true,
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, tokenString)

	token, _ := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return pubKey, nil
	})

	claims, ok := token.Claims.(jwt.MapClaims)
	require.True(t, ok, "bad claims format")

	var scopes []string
	for _, v := range claims["scope"].([]interface{}) {
		scopes = append(scopes, v.(string))
	}
	assert.Contains(t, scopes, "user:memberof:testorg.0stor.testns.read")
	assert.Contains(t, scopes, "user:memberof:testorg.0stor.testns.write")
	assert.Contains(t, scopes, "user:memberof:testorg.0stor.testns.delete")
	assert.Contains(t, scopes, "user:memberof:testorg.0stor.testns")
}
