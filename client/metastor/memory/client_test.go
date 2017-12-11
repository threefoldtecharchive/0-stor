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

	err = c.SetMetadata(metastor.Data{})
	require.Equal(metastor.ErrNilKey, err)

	err = c.DeleteMetadata(nil)
	require.Equal(metastor.ErrNilKey, err)
}
