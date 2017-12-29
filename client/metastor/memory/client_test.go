package memory

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/sync/errgroup"

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

func TestClientUpdate(t *testing.T) {
	require := require.New(t)

	c := NewClient()
	require.NotNil(c)
	defer c.Close()

	require.Panics(func() {
		c.UpdateMetadata([]byte("foo"), nil)
	}, "no callback given")

	_, err := c.UpdateMetadata(nil,
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

	c := NewClient()
	require.NotNil(c)
	defer c.Close()

	const (
		jobs = 1024
	)
	var (
		err error
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
