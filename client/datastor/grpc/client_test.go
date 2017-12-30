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
	"errors"
	"math"
	"testing"
	"time"

	"github.com/zero-os/0-stor/client/datastor"
	"github.com/zero-os/0-stor/server/api/grpc/rpctypes"
	pb "github.com/zero-os/0-stor/server/api/grpc/schema"

	"github.com/stretchr/testify/require"
)

func TestNewClientPanics(t *testing.T) {
	require := require.New(t)

	client, err := NewClient("", "", nil)
	require.Error(err, "no address given")
	require.Nil(client)

	client, err = NewClient("foo", "", nil)
	require.Error(err, "no label given")
	require.Nil(client)
}

func TestClientCreateObject(t *testing.T) {
	require := require.New(t)

	var os stubObjectService
	client := Client{objService: &os}
	client.contextConstructor = defaultContextConstructor("foo")

	_, err := client.CreateObject(nil)
	require.NoError(err, "server returns no error -> no error")

	os.err = rpctypes.ErrGRPCNilLabel
	_, err = client.CreateObject(nil)
	require.Equal(rpctypes.ErrNilLabel, err, "server returns GRPC error -> client error")

	errFoo := errors.New("fooErr")
	os.err = errFoo
	_, err = client.CreateObject(nil)
	require.Equal(errFoo, err, "any other error -> client error")
}

func TestClientGetObject(t *testing.T) {
	require := require.New(t)

	var os stubObjectService
	client := Client{objService: &os}
	client.contextConstructor = defaultContextConstructor("foo")

	obj, err := client.GetObject(nil)
	require.Equal(datastor.ErrMissingData, err)
	require.Nil(obj)

	os.data = []byte("foo")
	obj, err = client.GetObject(nil)
	require.NoError(err)
	require.NotNil(obj)
	require.Nil(obj.Key)
	require.Equal([]byte("foo"), obj.Data)

	os.err = rpctypes.ErrGRPCNilLabel
	obj, err = client.GetObject(nil)
	require.Equal(rpctypes.ErrNilLabel, err, "server returns GRPC error -> client error")
	require.Nil(obj)

	errFoo := errors.New("fooErr")
	os.err = errFoo
	obj, err = client.GetObject(nil)
	require.Equal(errFoo, err, "any other error -> client error")
	require.Nil(obj)
}

func TestClientDeleteObject(t *testing.T) {
	require := require.New(t)

	var os stubObjectService
	client := Client{objService: &os}
	client.contextConstructor = defaultContextConstructor("foo")

	err := client.DeleteObject(nil)
	require.NoError(err, "server returns no error -> no error")

	os.err = rpctypes.ErrGRPCNilLabel
	err = client.DeleteObject(nil)
	require.Equal(rpctypes.ErrNilLabel, err, "server returns GRPC error -> client error")

	errFoo := errors.New("fooErr")
	os.err = errFoo
	err = client.DeleteObject(nil)
	require.Equal(errFoo, err, "any other error -> client error")
}

func TestClientGetObjectStatus(t *testing.T) {
	require := require.New(t)

	var os stubObjectService
	client := Client{objService: &os}
	client.contextConstructor = defaultContextConstructor("foo")

	status, err := client.GetObjectStatus(nil)
	require.NoError(err)
	require.Equal(datastor.ObjectStatusMissing, status)

	os.status = math.MaxInt32
	status, err = client.GetObjectStatus(nil)
	require.Equal(datastor.ErrInvalidStatus, err)
	require.Equal(datastor.ObjectStatus(0), status)

	os.status = pb.ObjectStatusCorrupted
	status, err = client.GetObjectStatus(nil)
	require.NoError(err)
	require.Equal(datastor.ObjectStatusCorrupted, status)

	os.err = rpctypes.ErrGRPCNilLabel
	status, err = client.GetObjectStatus(nil)
	require.Equal(rpctypes.ErrNilLabel, err, "server returns GRPC error -> client error")
	require.Equal(datastor.ObjectStatus(0), status)

	errFoo := errors.New("fooErr")
	os.err = errFoo
	status, err = client.GetObjectStatus(nil)
	require.Equal(errFoo, err, "any other error -> client error")
	require.Equal(datastor.ObjectStatus(0), status)
}

func TestClientExistObject(t *testing.T) {
	require := require.New(t)

	var os stubObjectService
	client := Client{objService: &os}
	client.contextConstructor = defaultContextConstructor("foo")

	exists, err := client.ExistObject(nil)
	require.NoError(err)
	require.False(exists)

	os.status = math.MaxInt32
	exists, err = client.ExistObject(nil)
	require.Equal(datastor.ErrInvalidStatus, err)
	require.False(exists)

	os.status = pb.ObjectStatusCorrupted
	exists, err = client.ExistObject(nil)
	require.Equal(datastor.ErrObjectCorrupted, err)
	require.False(exists)

	os.status = pb.ObjectStatusOK
	exists, err = client.ExistObject(nil)
	require.NoError(err)
	require.True(exists)

	os.status = pb.ObjectStatusMissing
	exists, err = client.ExistObject(nil)
	require.NoError(err)
	require.False(exists)

	os.err = rpctypes.ErrGRPCNilLabel
	exists, err = client.ExistObject(nil)
	require.Equal(rpctypes.ErrNilLabel, err, "server returns GRPC error -> client error")
	require.False(exists)

	errFoo := errors.New("fooErr")
	os.err = errFoo
	exists, err = client.ExistObject(nil)
	require.Equal(errFoo, err, "any other error -> client error")
	require.False(exists)
}

func TestClientGetNamespace(t *testing.T) {
	require := require.New(t)

	var nss stubNamespaceService
	client := Client{namespaceService: &nss, label: "myLabel"}
	client.contextConstructor = defaultContextConstructor("foo")

	ns, err := client.GetNamespace()
	require.Equal(datastor.ErrInvalidLabel, err, "returned label should be equal to client's label")
	require.Nil(ns)

	nss.label = "myLabel"
	ns, err = client.GetNamespace()
	require.NoError(err, "returned label should be equal to client's label")
	require.NotNil(ns)
	require.Equal("myLabel", ns.Label)
	require.Equal(int64(0), ns.ReadRequestPerHour)
	require.Equal(int64(0), ns.WriteRequestPerHour)
	require.Equal(int64(0), ns.NrObjects)

	nss.err = rpctypes.ErrGRPCNilLabel
	ns, err = client.GetNamespace()
	require.Equal(rpctypes.ErrNilLabel, err, "server errors should be caught by client")
	require.Nil(ns)

	nss.nrOfObjects = 42
	nss.readRPH = 1023
	nss.writeRPH = math.MaxInt64
	nss.err = nil
	ns, err = client.GetNamespace()
	require.NoError(err)
	require.NotNil(ns)
	require.Equal("myLabel", ns.Label)
	require.Equal(int64(1023), ns.ReadRequestPerHour)
	require.Equal(int64(math.MaxInt64), ns.WriteRequestPerHour)
	require.Equal(int64(42), ns.NrObjects)

	client.label = "foo"
	ns, err = client.GetNamespace()
	require.Equal(datastor.ErrInvalidLabel, err, "returned label should be equal to client's label")
	require.Nil(ns)
}

func TestClientListObjectKeys(t *testing.T) {
	require := require.New(t)

	var os stubObjectService
	client := Client{objService: &os}
	client.contextConstructor = defaultContextConstructor("foo")

	require.Panics(func() {
		client.ListObjectKeyIterator(nil)
	}, "no context given")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch, err := client.ListObjectKeyIterator(ctx)
	require.NoError(err, "server returns no error -> no error")
	require.NotNil(ch)
	select {
	case resp, open := <-ch:
		require.True(open)
		require.Equal(datastor.ErrMissingKey, resp.Error)
		require.Nil(resp.Key)
	case <-time.After(time.Millisecond * 500):
	}

	os.err = rpctypes.ErrGRPCNilLabel
	ch, err = client.ListObjectKeyIterator(context.Background())
	require.Equal(rpctypes.ErrNilLabel, err)
	require.Nil(ch)

	os.err = nil
	os.streamErr = datastor.ErrMissingKey
	ch, err = client.ListObjectKeyIterator(context.Background())
	require.NoError(err, "channel should be created")
	require.NotNil(ch)
	select {
	case resp, open := <-ch:
		require.True(open)
		require.Equal(datastor.ErrMissingKey, resp.Error)
		require.Nil(resp.Key)
	case <-time.After(time.Millisecond * 500):
		t.Fatal("nothing received from channel while it was expected")
	}
	select {
	case _, open := <-ch:
		require.False(open)
	case <-time.After(time.Millisecond * 500):
	}

	os.streamErr = nil
	os.key = []byte("foo")
	ch, err = client.ListObjectKeyIterator(context.Background())
	require.NoError(err, "channel should be created")
	require.NotNil(ch)
	select {
	case resp, open := <-ch:
		require.True(open)
		require.NoError(resp.Error)
		require.Equal([]byte("foo"), resp.Key)
	case <-time.After(time.Millisecond * 500):
		t.Fatal("nothing received from channel while it was expected")
	}
	select {
	case _, open := <-ch:
		require.False(open)
	case <-time.After(time.Millisecond * 500):
	}
}

func TestClientClosePanic(t *testing.T) {
	client := new(Client)
	require.Panics(t, func() {
		client.Close()
	}, "nil pointer dereference, because connection property is not set")
}
