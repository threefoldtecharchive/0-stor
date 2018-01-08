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
	"net"
	"testing"
	"time"

	"github.com/zero-os/0-stor/client/metastor/db/test"

	"github.com/stretchr/testify/require"
)

func TestDB_RoundTrip(t *testing.T) {
	etcd, err := NewEmbeddedServer()
	require.NoError(t, err)

	db, err := New([]string{etcd.ListenAddr()})
	require.NoError(t, err)
	defer db.Close()

	test.RoundTrip(t, db)
}

func TestDB_SyncUpdate(t *testing.T) {
	etcd, err := NewEmbeddedServer()
	require.NoError(t, err)

	db, err := New([]string{etcd.ListenAddr()})
	require.NoError(t, err)
	defer db.Close()

	test.SyncUpdate(t, db)
}

func TestDB_AsyncUpdate(t *testing.T) {
	etcd, err := NewEmbeddedServer()
	require.NoError(t, err)

	db, err := New([]string{etcd.ListenAddr()})
	require.NoError(t, err)
	defer db.Close()

	test.AsyncUpdate(t, db)
}

func TestDB_ConstructorErrors(t *testing.T) {
	_, err := New(nil)
	require.Error(t, err)
}

// test that client can return gracefully when the server is not exist
func TestDB_NonExistingServer(t *testing.T) {
	_, err := New([]string{"http://foo:42"})
	require.Error(t, err)
	// make sure it is network error
	_, ok := err.(net.Error)
	require.True(t, ok)
}

func TestDB_ServerDown(t *testing.T) {
	require := require.New(t)

	etcd, err := NewEmbeddedServer()
	require.NoError(err)

	db, err := New([]string{etcd.ListenAddr()})
	require.NoError(err)
	defer db.Close()

	key, value := []byte("foo"), []byte("bar")

	// make sure we can do some operation to server
	err = db.Set(key, value)
	require.NoError(err)

	output, err := db.Get(key)
	require.NoError(err)
	require.NotNil(output)
	require.Equal(value, output)

	// stop the server
	etcd.Stop()

	// test the SET
	startCh := make(chan struct{}, 1)
	errCh := make(chan error, 1)
	go func() {
		startCh <- struct{}{}
		err := db.Set(key, []byte("42"))
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
		_, err := db.Get(key)
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
