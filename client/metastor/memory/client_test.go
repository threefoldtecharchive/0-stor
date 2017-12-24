package memory

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zero-os/0-stor/client/metastor"
)

func TestRoundTrip(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	c := NewClient()
	require.NotNil(c)
	defer c.Close()

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
	_, err := c.GetMetadata(md.Key)
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

func TestClientNilKeys(t *testing.T) {
	require := require.New(t)

	c := NewClient()
	require.NotNil(c)
	defer c.Close()

	_, err := c.GetMetadata(nil)
	require.Equal(metastor.ErrNilKey, err)

	err = c.SetMetadata(metastor.Metadata{})
	require.Equal(metastor.ErrNilKey, err)

	err = c.DeleteMetadata(nil)
	require.Equal(metastor.ErrNilKey, err)
}
