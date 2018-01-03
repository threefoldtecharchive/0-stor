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

package etcd

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/zero-os/0-stor/client/metastor"
	"github.com/zero-os/0-stor/client/metastor/encoding"
	"github.com/zero-os/0-stor/client/metastor/encoding/proto"
	"github.com/zero-os/0-stor/client/metastor/test"

	"github.com/stretchr/testify/require"
)

func TestRoundTrip(t *testing.T) {
	etcd, err := NewEmbeddedServer()
	require.NoError(t, err)

	c, err := NewClient([]string{etcd.ListenAddr()}, nil)
	require.NoError(t, err)
	defer c.Close()

	test.RoundTrip(t, c)
}

// test that client can return gracefully when the server is not exist
func TestServerNotExist(t *testing.T) {
	_, err := NewClient([]string{"http://localhost:1234"}, nil)

	// make sure it is network error
	_, ok := err.(net.Error)
	require.True(t, ok)
}

func TestClientExplicitPanics(t *testing.T) {
	require := require.New(t)

	// explicit panics to guarantee all
	// input parameters are given when creating a client
	require.Panics(func() {
		NewClient(nil, &encoding.MarshalFuncPair{
			Marshal:   proto.MarshalMetadata,
			Unmarshal: proto.UnmarshalMetadata,
		})
	}, "no endpoints given")
	require.Panics(func() {
		NewClient([]string{"foo"}, &encoding.MarshalFuncPair{
			Marshal:   nil,
			Unmarshal: proto.UnmarshalMetadata,
		})
	}, "no marshal func given")
	require.Panics(func() {
		NewClient([]string{"foo"}, &encoding.MarshalFuncPair{
			Marshal:   proto.MarshalMetadata,
			Unmarshal: nil,
		})
	}, "no unmarshal func given")
	require.Panics(func() {
		NewClient([]string{"foo"}, &encoding.MarshalFuncPair{
			Marshal:   nil,
			Unmarshal: nil,
		})
	}, "no (un)marshal funcs given")
	require.Panics(func() {
		NewClient(nil, nil)
	}, "nothing given")
}

func TestClientNilKeys(t *testing.T) {
	etcd, err := NewEmbeddedServer()
	require.NoError(t, err)

	c, err := NewClient([]string{etcd.ListenAddr()}, nil)
	require.NoError(t, err)
	defer c.Close()

	test.ClientNilKeys(t, c)
}

func TestInvalidMetadataObject(t *testing.T) {
	require := require.New(t)

	etcd, err := NewEmbeddedServer()
	require.NoError(err)

	c, err := NewClient([]string{etcd.ListenAddr()}, nil)
	require.NoError(err)
	defer c.Close()

	// store an invalid value
	func() {
		ctx, cancel := context.WithTimeout(context.Background(), metaOpTimeout)
		defer cancel()

		_, err = c.etcdClient.Put(ctx, "foo", "bar")
		require.NoError(err)
	}()

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

// test that client can return gracefully when the server is down
func TestServerDown(t *testing.T) {
	require := require.New(t)

	etcd, err := NewEmbeddedServer()
	require.Nil(err)

	c, err := NewClient([]string{etcd.ListenAddr()}, nil)
	require.Nil(err)
	defer c.Close()

	md := metastor.Metadata{Key: []byte("key")}

	// make sure we can do some operation to server
	err = c.SetMetadata(md)
	require.Nil(err)

	outMD, err := c.GetMetadata(md.Key)
	require.Nil(err)
	require.NotNil(outMD)
	require.Equal(md, *outMD)

	// stop the server
	etcd.Stop()

	// test the SET
	startCh := make(chan struct{}, 1)
	errCh := make(chan error, 1)
	go func() {
		startCh <- struct{}{}
		err := c.SetMetadata(md)
		errCh <- err
	}()

	// wait until goroutine has started,
	// otherwise our timing (wait) might be off
	<-startCh

	select {
	case err := <-errCh:
		require.Error(err)
		t.Logf("operation exited successfully")
	case <-time.After(metaOpTimeout + time.Second*30):
		// the put operation should be exited before the timeout
		t.Error("the operation should already returns with error")
	}

	// test the GET
	go func() {
		startCh <- struct{}{}
		_, err := c.GetMetadata(md.Key)
		errCh <- err
	}()

	// wait until goroutine has started,
	// otherwise our timing (wait) might be off
	<-startCh

	select {
	case err := <-errCh:
		require.Error(err)
		t.Logf("operation exited successfully")
	case <-time.After(metaOpTimeout + time.Second*30):
		// the Get operation should be exited before the timeout
		t.Error("the operation should already returns with error")
	}
}

func TestClientUpdate(t *testing.T) {
	etcd, err := NewEmbeddedServer()
	require.NoError(t, err)

	c, err := NewClient([]string{etcd.ListenAddr()}, nil)
	require.NoError(t, err)
	defer c.Close()

	test.ClientUpdate(t, c)
}

func TestClientUpdateAsync(t *testing.T) {
	etcd, err := NewEmbeddedServer()
	require.NoError(t, err)

	c, err := NewClient([]string{etcd.ListenAddr()}, nil)
	require.NoError(t, err)
	defer c.Close()

	test.ClientUpdateAsync(t, c)
}
