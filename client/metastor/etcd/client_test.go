package etcd

import (
	"context"
	"errors"
	"math"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zero-os/0-stor/client/metastor"
	"github.com/zero-os/0-stor/client/metastor/encoding/proto"
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
	md := metastor.Data{
		Key:   []byte("two"),
		Epoch: 123456789,
		Chunks: []*metastor.Chunk{
			&metastor.Chunk{
				Size:   math.MaxInt64,
				Key:    []byte("foo"),
				Shards: nil,
			},
			&metastor.Chunk{
				Size:   1234,
				Key:    []byte("bar"),
				Shards: []string{"foo"},
			},
			&metastor.Chunk{
				Size:   2,
				Key:    []byte("baz"),
				Shards: []string{"bar", "foo"},
			},
		},
		Next:     []byte("one"),
		Previous: []byte("three"),
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

	err = c.SetMetadata(metastor.Data{})
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
	err = c.SetMetadata(metastor.Data{Key: []byte("whatever-forever")})
	require.NoError(err)
	myEncodingErr := errors.New("encoding error: pwned")
	c.marshal = func(md metastor.Data) ([]byte, error) {
		return nil, myEncodingErr
	}
	err = c.SetMetadata(metastor.Data{Key: []byte("whatever-forever")})
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

	md := metastor.Data{Key: []byte("key")}

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
	done := make(chan struct{}, 1)
	go func() {
		err = c.SetMetadata(md)
		_, ok := err.(net.Error)
		require.True(ok)
		done <- struct{}{}
	}()

	select {
	case <-time.After(metaOpTimeout + (5 * time.Second)):
		// the put operation should be exited before the timeout
		t.Error("the operation should already returns with error")
	case <-done:
		t.Logf("operation exited successfully")
	}

	// test the GET
	done = make(chan struct{}, 1)
	go func() {
		_, err = c.GetMetadata(md.Key)
		_, ok := err.(net.Error)
		require.True(ok)
		done <- struct{}{}
	}()

	select {
	case <-time.After(metaOpTimeout + (5 * time.Second)):
		// the Get operation should be exited before the timeout
		t.Error("the operation should already returns with error")
	case <-done:
		t.Logf("operation exited successfully")
	}

}
