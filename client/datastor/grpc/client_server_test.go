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
	"context"
	"testing"
	"time"

	"github.com/zero-os/0-stor/client/datastor"
	"github.com/zero-os/0-stor/server/api/grpc/rpctypes"

	"github.com/stretchr/testify/require"
)

func TestClientWithServer_API(t *testing.T) {
	require := require.New(t)

	client, _, clean, err := newServerClient()
	require.NoError(err)
	defer clean()
	require.NotNil(client)

	var (
		key = []byte("myKey")
	)

	// all getters should "work",
	// even though the key doesn't exist

	obj, err := client.GetObject(key)
	require.Equal(datastor.ErrKeyNotFound, err)
	require.Nil(obj)

	status, err := client.GetObjectStatus(key)
	require.NoError(err)
	require.Equal(datastor.ObjectStatusMissing, status)

	exists, err := client.ExistObject(key)
	require.NoError(err)
	require.False(exists)

	// even the list object key iterator should work,
	// although it should close immediately

	ch, err := client.ListObjectKeyIterator(context.Background())
	require.NoError(err)
	select {
	case _, open := <-ch:
		require.False(open)
	case <-time.After(time.Millisecond * 500):
	}

	// now let's set something, but miss ome required params,
	// making these calls invalid and have no affect on the database's state

	_, err = client.CreateObject(nil)
	require.Equal(rpctypes.ErrNilData, err)

	// let's ensure our object is still missing

	status, err = client.GetObjectStatus(key)
	require.NoError(err)
	require.Equal(datastor.ObjectStatusMissing, status)

	exists, err = client.ExistObject(key)
	require.NoError(err)
	require.False(exists)

	// now let's set the object for real

	data := []byte("myData")
	key, err = client.CreateObject(data)
	require.NoError(err)
	require.NotNil(key)

	// our getters from earlier should work now

	obj, err = client.GetObject(key)
	require.NoErrorf(err, "Key: %s", key)
	require.NotNil(obj)
	require.Equal(key, obj.Key)
	require.Equal(data, obj.Data)

	status, err = client.GetObjectStatus(key)
	require.NoError(err)
	require.Equal(datastor.ObjectStatusOK, status)

	exists, err = client.ExistObject(key)
	require.NoError(err)
	require.True(exists)

	// The iterator will work now as well

	ch, err = client.ListObjectKeyIterator(context.Background())
	require.NoError(err)
	select {
	case result, open := <-ch:
		require.True(open)
		require.NoError(result.Error)
		require.Equal(key, result.Key)
	case <-time.After(time.Millisecond * 500):
	}
	select {
	case _, open := <-ch:
		require.False(open)
	case <-time.After(time.Millisecond * 500):
	}

	otherData := []byte("some other data")
	// now let's add one more object
	otherKey, err := client.CreateObject(otherData)
	require.NoError(err)
	require.NotNil(otherKey)

	// let's add another one, it's starting to get fun
	whyNotData := []byte("why not data")
	// now let's add one more object
	yetAnotherKey, err := client.CreateObject(whyNotData)
	require.NoError(err)
	require.NotNil(otherKey)

	// now let's list them all, they should all appear!
	objects := map[string]datastor.Object{
		string(key): {
			Key:  key,
			Data: data,
		},
		string(otherKey): {
			Key:  otherKey,
			Data: otherData,
		},
		string(yetAnotherKey): {
			Key:  yetAnotherKey,
			Data: whyNotData,
		},
	}
	// start the iteration
	ch, err = client.ListObjectKeyIterator(context.Background())
	require.NoError(err)
	for resp := range ch {
		require.NoError(resp.Error)
		require.NotNil(resp.Key)

		expObject, ok := objects[string(resp.Key)]
		require.True(ok)
		require.Equal(expObject.Key, resp.Key)
		delete(objects, string(resp.Key))

		obj, err := client.GetObject(expObject.Key)
		require.NoError(err)
		require.NotNil(obj)
		require.Equal(expObject.Key, obj.Key)
		require.Equal(expObject.Data, obj.Data)
	}
	require.Empty(objects, "all objects should have been listed")
}
