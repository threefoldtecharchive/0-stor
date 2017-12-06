package memory

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zero-os/0-stor/client/meta"
)

func TestRoundTrip(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	c := NewClient()
	require.NotNil(c)
	defer c.Close()

	// prepare the data
	md := meta.Data{
		Key:   []byte("two"),
		Epoch: 123456789,
		Chunks: []*meta.Chunk{
			&meta.Chunk{
				Size:   math.MaxInt64,
				Key:    []byte("foo"),
				Shards: nil,
			},
			&meta.Chunk{
				Size:   1234,
				Key:    []byte("bar"),
				Shards: []string{"foo"},
			},
			&meta.Chunk{
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
	require.Equal(meta.ErrNotFound, err)

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
	require.Equal(meta.ErrNotFound, err)
}

func TestClientNilKeys(t *testing.T) {
	require := require.New(t)

	c := NewClient()
	require.NotNil(c)
	defer c.Close()

	_, err := c.GetMetadata(nil)
	require.Equal(meta.ErrNilKey, err)

	err = c.SetMetadata(meta.Data{})
	require.Equal(meta.ErrNilKey, err)

	err = c.DeleteMetadata(nil)
	require.Equal(meta.ErrNilKey, err)
}
