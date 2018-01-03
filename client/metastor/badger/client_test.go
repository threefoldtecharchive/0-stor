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

package badger

import (
	"errors"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/zero-os/0-stor/client/metastor"
	"github.com/zero-os/0-stor/client/metastor/encoding"
	"github.com/zero-os/0-stor/client/metastor/encoding/proto"
	"github.com/zero-os/0-stor/client/metastor/test"

	badgerdb "github.com/dgraph-io/badger"
	"github.com/stretchr/testify/require"
)

func makeTestClient(t *testing.T) (*Client, func()) {
	tmpDir, err := ioutil.TempDir("", "0-stor-test")
	require.NoError(t, err)

	client, err := NewClient(path.Join(tmpDir, "data"), path.Join(tmpDir, "meta"), nil)
	require.NoError(t, err)
	require.NotNil(t, client)
	cleanup := func() {
		client.Close()
		os.RemoveAll(tmpDir)
	}
	return client, cleanup
}

func TestClientExplicitPanics(t *testing.T) {
	require := require.New(t)

	// explicit panics to guarantee all
	// input parameters are given when creating a client
	require.Panics(func() {
		NewClient("", "meta", &encoding.MarshalFuncPair{
			Marshal:   proto.MarshalMetadata,
			Unmarshal: proto.UnmarshalMetadata,
		})
	}, "no data dir given")
	require.Panics(func() {
		NewClient("data", "", &encoding.MarshalFuncPair{
			Marshal:   proto.MarshalMetadata,
			Unmarshal: proto.UnmarshalMetadata,
		})
	}, "no meta dir given")
	require.Panics(func() {
		NewClient("", "", &encoding.MarshalFuncPair{
			Marshal:   proto.MarshalMetadata,
			Unmarshal: proto.UnmarshalMetadata,
		})
	}, "no data or meta dir given")
	require.Panics(func() {
		NewClient("data", "meta", &encoding.MarshalFuncPair{
			Marshal:   nil,
			Unmarshal: proto.UnmarshalMetadata,
		})
	}, "no marshal func given")
	require.Panics(func() {
		NewClient("data", "meta", &encoding.MarshalFuncPair{
			Marshal:   proto.MarshalMetadata,
			Unmarshal: nil,
		})
	}, "no unmarshal func given")
	require.Panics(func() {
		NewClient("data", "meta", &encoding.MarshalFuncPair{
			Marshal:   nil,
			Unmarshal: nil,
		})
	}, "no (un)marshal funcs given")
	require.Panics(func() {
		NewClient("", "", nil)
	}, "nothing given")
}

func TestRoundTrip(t *testing.T) {
	c, cleanup := makeTestClient(t)
	defer cleanup()

	test.RoundTrip(t, c)
}

func TestClientNilKeys(t *testing.T) {
	c, cleanup := makeTestClient(t)
	defer cleanup()

	test.ClientNilKeys(t, c)
}

func TestInvalidMetadataObject(t *testing.T) {
	require := require.New(t)

	c, cleanup := makeTestClient(t)
	defer cleanup()

	// store an invalid value
	err := c.db.Update(func(txn *badgerdb.Txn) error {
		return txn.Set([]byte("foo"), []byte("bar"))
	})
	require.NoError(err)

	// now try to fetch it, and ensure that we idd get an error
	_, err = c.GetMetadata([]byte("foo"))
	require.Error(err)
	require.NotEqual(metastor.ErrNotFound, err)

	// also ensure we get an error in case the marshaler returns an error
	err = c.SetMetadata(metastor.Metadata{Key: []byte("whatever-forever")})
	require.NoError(err)
	myEncodingErr := errors.New("encoding error: pwned")
	c.marshal = func(md metastor.Metadata) ([]byte, error) {
		return nil, myEncodingErr
	}
	err = c.SetMetadata(metastor.Metadata{Key: []byte("whatever-forever")})
	require.Equal(myEncodingErr, err)
}

func TestClientUpdate(t *testing.T) {
	c, cleanup := makeTestClient(t)
	defer cleanup()

	test.ClientUpdate(t, c)
}

func TestClientUpdateAsync(t *testing.T) {
	c, cleanup := makeTestClient(t)
	defer cleanup()

	test.ClientUpdateAsync(t, c)
}
