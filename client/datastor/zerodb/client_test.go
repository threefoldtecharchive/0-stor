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

package zerodb

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/threefoldtech/0-stor/client/datastor"
	zdbtest "github.com/threefoldtech/0-stor/client/datastor/zerodb/test"
)

func TestRoundTrip(t *testing.T) {
	var (
		namespace = "ns"
		data      = []byte("data")
	)
	require := require.New(t)

	// create server
	_,addr, cleanup, err := zdbtest.NewInMem0DBServer(namespace)
	require.NoError(err)
	defer cleanup()

	// create client
	c, err := NewClient(addr, "mypasswd", namespace)
	require.NoError(err)

	// create object
	key, err := c.CreateObject(data)
	require.NoError(err)

	// exist object
	exists, err := c.ExistObject(key)
	require.NoError(err)
	require.True(exists)

	// status
	status, err := c.GetObjectStatus(key)
	require.NoError(err)
	require.Equal(datastor.ObjectStatusOK, status)

	// get object
	obj, err := c.GetObject(key)
	require.NoError(err)
	require.Equal(key, obj.Key)
	require.Equal(data, obj.Data)

	nsObj, err := c.GetNamespace()
	require.NoError(err)
	require.NotNil(nsObj)
	require.Equal(namespace, nsObj.Label)
	require.Equal(int64(1), nsObj.NrObjects)

	// delete object
	err = c.DeleteObject(key)
	require.NoError(err)

	// check that object not exist anymore after deletion
	exists, err = c.ExistObject(key)
	require.NoError(err)
	require.False(exists)

	// check status
	status, err = c.GetObjectStatus(key)
	require.NoError(err)
	require.Equal(datastor.ObjectStatusMissing, status)
}

func TestNewClientPanics(t *testing.T) {
	require := require.New(t)

	client, err := NewClient("", "", "")
	require.Error(err, "no address given")
	require.Nil(client)

	client, err = NewClient("foo", "", "")
	require.Error(err, "no namespace given")
	require.Nil(client)
}
