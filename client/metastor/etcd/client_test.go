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
	"fmt"
	"math"
	"net"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/zero-os/0-stor/client/metastor"
	"github.com/zero-os/0-stor/client/metastor/encoding/proto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

func TestRoundTrip(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	etcd, err := NewEmbeddedServer()
	require.NoError(err)
	defer etcd.Stop()

	c, err := NewClient([]string{etcd.ListenAddr()})
	require.NoError(err)
	defer c.Close()

	require.Equal([]string{etcd.ListenAddr()}, c.Endpoints())

	// prepare the data
	md := metastor.Metadata{
		Key:            []byte("two"),
		Size:           42,
		CreationEpoch:  123456789,
		LastWriteEpoch: 123456789,
		Chunks: []metastor.Chunk{
			{
				Size:    math.MaxInt64,
				Hash:    []byte("foo"),
				Objects: nil,
			},
			{
				Size: 1234,
				Hash: []byte("bar"),
				Objects: []metastor.Object{
					{
						Key:     []byte("foo"),
						ShardID: "bar",
					},
				},
			},
			{
				Size: 2,
				Hash: []byte("baz"),
				Objects: []metastor.Object{
					{
						Key:     []byte("foo"),
						ShardID: "bar",
					},
					{
						Key:     []byte("bar"),
						ShardID: "baz",
					},
				},
			},
		},
		NextKey:     []byte("one"),
		PreviousKey: []byte("three"),
	}

	// ensure metadata is not there yet
	_, err = c.GetMetadata(md.Key)
	require.Equal(metastor.ErrNotFound, err)

	// set the metadata
	err = c.SetMetadata(md)
	require.NoError(err)

	// get it back
	storedMd, err := c.GetMetadata(md.Key)
	require.NoError(err)

	// check stored value
	assert.NotNil(storedMd)
	assert.Equal(md, *storedMd)

	err = c.DeleteMetadata(md.Key)
	require.NoError(err)
	// make sure we can't get it back
	_, err = c.GetMetadata(md.Key)
	require.Equal(metastor.ErrNotFound, err)
}

// test that client can return gracefully when the server is not exist
func TestServerNotExist(t *testing.T) {
	_, err := NewClient([]string{"http://localhost:1234"})

	// make sure it is network error
	_, ok := err.(net.Error)
	require.True(t, ok)
}

func TestClientExplicitPanics(t *testing.T) {
	require := require.New(t)

	// explicit panics to guarantee all
	// input parameters are given when creating a client
	require.Panics(func() {
		NewClientWithEncoding(nil, proto.MarshalMetadata, proto.UnmarshalMetadata)
	}, "no endpoints given")
	require.Panics(func() {
		NewClientWithEncoding([]string{"foo"}, nil, proto.UnmarshalMetadata)
	}, "no marshal func given")
	require.Panics(func() {
		NewClientWithEncoding([]string{"foo"}, proto.MarshalMetadata, nil)
	}, "no unmarshal func given")
	require.Panics(func() {
		NewClientWithEncoding([]string{"foo"}, nil, nil)
	}, "no (un)marshal funcs given")
	require.Panics(func() {
		NewClientWithEncoding(nil, nil, nil)
	}, "nothing given")
}

func TestClientNilKeys(t *testing.T) {
	require := require.New(t)

	etcd, err := NewEmbeddedServer()
	require.NoError(err)

	c, err := NewClient([]string{etcd.ListenAddr()})
	require.NoError(err)
	defer c.Close()

	_, err = c.GetMetadata(nil)
	require.Equal(metastor.ErrNilKey, err)

	err = c.SetMetadata(metastor.Metadata{})
	require.Equal(metastor.ErrNilKey, err)

	err = c.DeleteMetadata(nil)
	require.Equal(metastor.ErrNilKey, err)
}

func TestInvalidMetadataObject(t *testing.T) {
	require := require.New(t)

	etcd, err := NewEmbeddedServer()
	require.NoError(err)

	c, err := NewClient([]string{etcd.ListenAddr()})
	require.NoError(err)
	defer c.Close()

	require.Equal([]string{etcd.ListenAddr()}, c.Endpoints())

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

	c, err := NewClient([]string{etcd.ListenAddr()})
	require.Nil(err)
	defer c.Close()

	require.Equal([]string{etcd.ListenAddr()}, c.Endpoints())

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
	require := require.New(t)

	etcd, err := NewEmbeddedServer()
	require.NoError(err)

	c, err := NewClient([]string{etcd.ListenAddr()})
	require.NoError(err)
	defer c.Close()

	require.Panics(func() {
		c.UpdateMetadata([]byte("foo"), nil)
	}, "no callback given")

	_, err = c.UpdateMetadata(nil,
		func(md metastor.Metadata) (*metastor.Metadata, error) { return &md, nil })
	require.Equal(metastor.ErrNilKey, err)

	_, err = c.UpdateMetadata([]byte("foo"),
		func(md metastor.Metadata) (*metastor.Metadata, error) { return &md, nil })
	require.Equal(metastor.ErrNotFound, err)

	err = c.SetMetadata(metastor.Metadata{Key: []byte("foo")})
	require.NoError(err)

	md, err := c.GetMetadata([]byte("foo"))
	require.NoError(err)
	require.Equal("foo", string(md.Key))
	require.Equal(int64(0), md.Size)

	md, err = c.UpdateMetadata([]byte("foo"),
		func(md metastor.Metadata) (*metastor.Metadata, error) {
			md.Size = 42
			return &md, nil
		})
	require.NoError(err)
	require.Equal("foo", string(md.Key))
	require.Equal(int64(42), md.Size)

	md, err = c.GetMetadata([]byte("foo"))
	require.NoError(err)
	require.Equal("foo", string(md.Key))
	require.Equal(int64(42), md.Size)
}

func TestClientUpdateAsync(t *testing.T) {
	require := require.New(t)

	etcd, err := NewEmbeddedServer()
	require.NoError(err)

	c, err := NewClient([]string{etcd.ListenAddr()})
	require.NoError(err)
	defer c.Close()

	const (
		jobs = 128
	)
	var (
		key = []byte("foo")
	)

	err = c.SetMetadata(metastor.Metadata{Key: key})
	require.NoError(err)

	group, _ := errgroup.WithContext(context.Background())
	for i := 0; i < jobs; i++ {
		i := i
		group.Go(func() error {
			var expectedSize int64
			md, err := c.UpdateMetadata(key,
				func(md metastor.Metadata) (*metastor.Metadata, error) {
					md.Size++
					md.NextKey = []byte(string(md.NextKey) + fmt.Sprintf("%d,", i))
					expectedSize = md.Size
					return &md, nil
				})
			if err != nil {
				return err
			}
			if md == nil {
				return fmt.Errorf("job #%d: md is nil while this is not expected", i)
			}
			if expectedSize != md.Size {
				return fmt.Errorf("job #%d: unexpected size => %d != %d",
					i, expectedSize, md.Size)
			}
			return nil
		})
	}
	require.NoError(group.Wait())

	md, err := c.GetMetadata(key)
	require.NoError(err)
	require.Equal(string(key), string(md.Key))
	require.Equal(int64(jobs), md.Size)
	require.NotEmpty(md.NextKey)

	rawIntegers := strings.Split(string(md.NextKey[:len(md.NextKey)-1]), ",")
	require.Len(rawIntegers, jobs)

	integers := make([]int, jobs)
	for i, raw := range rawIntegers {
		integer, err := strconv.Atoi(raw)
		require.NoError(err)
		integers[i] = integer
	}

	sort.Ints(integers)
	for i := 0; i < jobs; i++ {
		require.Equal(i, integers[i])
	}
}
