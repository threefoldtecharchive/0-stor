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

package grpc

import (
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"net"
	"testing"

	"github.com/zero-os/0-stor/server"
	dbp "github.com/zero-os/0-stor/server/db"
	"github.com/zero-os/0-stor/server/db/memory"
	"github.com/zero-os/0-stor/server/encoding"
	"github.com/zero-os/0-stor/server/jwt"
	"github.com/zero-os/0-stor/stubs"

	jwtgo "github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/require"
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

type testServer struct {
	*Server
	addr string
}

func (ts *testServer) Address() string {
	return ts.addr
}

func getTestGRPCServer(t *testing.T, organization string) (*testServer, stubs.IYOClient, func()) {
	listener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)

	var client stubs.IYOClient
	var verifier jwt.TokenVerifier
	if organization != "" {
		client = getIYOClient(t, organization)
		var err error
		verifier, err = getTestVerifier(testPubKeyPath)
		require.NoError(t, err)
	} else {
		verifier = jwt.NopVerifier{}
	}

	server, err := New(memory.New(), verifier, 4, 0)
	require.NoError(t, err)

	go func() {
		err := server.Serve(listener)
		if err != nil {
			panic(err)
		}
	}()

	clean := func() {
		fmt.Sprintln("clean called")
		server.Close()
	}
	return &testServer{server, listener.Addr().String()}, client, clean
}

func getTestVerifier(pubKeyPath string) (*jwt.Verifier, error) {
	pubKey, err := ioutil.ReadFile(pubKeyPath)
	if err != nil {
		return nil, err
	}
	return jwt.NewVerifier(string(pubKey))
}

func getIYOClient(t testing.TB, organization string) stubs.IYOClient {
	b, err := ioutil.ReadFile(testPrivKeyPath)
	require.NoError(t, err)

	key, err := jwtgo.ParseECPrivateKeyFromPEM(b)
	require.NoError(t, err)

	jwtCreator, err := stubs.NewStubIYOClient(organization, key)
	require.NoError(t, err, "failed to create the stub IYO client")

	return jwtCreator
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
