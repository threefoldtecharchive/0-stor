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

package metastor

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/zero-os/0-stor/client/metastor/db/test"
	"github.com/zero-os/0-stor/client/metastor/encoding"
	"github.com/zero-os/0-stor/client/metastor/metatypes"
	"github.com/zero-os/0-stor/client/processing"

	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

func TestNewClient_ExplicitErrors(t *testing.T) {
	_, err := NewClient(Config{})
	require.Error(t, err, "no database given")

	_, err = NewClient(Config{
		Database: test.New(),
		MarshalFuncPair: &encoding.MarshalFuncPair{
			Marshal: binaryMetadataMarshal,
		},
	})
	require.Error(t, err, "no unmarshal func given, while received a non-nil pair")

	_, err = NewClient(Config{
		Database: test.New(),
		MarshalFuncPair: &encoding.MarshalFuncPair{
			Unmarshal: binaryMetadataUnmarshal,
		},
	})
	require.Error(t, err, "no marshal func given, while received a non-nil pair")
}

func TestClient_RoundTrip(t *testing.T) {
	testClient(t, testRoundTrip)
}

func TestClient_NilKeys(t *testing.T) {
	testClient(t, testClientNilKeys)
}

func TestClient_Update(t *testing.T) {
	testClient(t, testClientUpdate)
}

func TestClient_UpdateAsync(t *testing.T) {
	testClient(t, testClientUpdateAsync)
}

func testClient(t *testing.T, f func(t *testing.T, c *Client)) {
	t.Run("in_mem_db+default_cfg", func(t *testing.T) {
		client, err := NewClient(Config{
			Database: test.New(),
		})
		require.NoError(t, err)
		defer func() {
			err := client.Close()
			if err != nil {
				panic(err)
			}
		}()

		f(t, client)
	})

	t.Run("in_mem_db+binary_encoding", func(t *testing.T) {
		client, err := NewClient(Config{
			Database:        test.New(),
			MarshalFuncPair: binaryMarshalFuncPair,
		})
		require.NoError(t, err)
		defer func() {
			err := client.Close()
			if err != nil {
				panic(err)
			}
		}()

		f(t, client)
	})

	t.Run("in_mem_db+AES_32", func(t *testing.T) {
		client, err := NewClient(Config{
			Database:             test.New(),
			ProcessorConstructor: encrypterDecrypterConstructor,
		})
		require.NoError(t, err)
		defer func() {
			err := client.Close()
			if err != nil {
				panic(err)
			}
		}()

		f(t, client)
	})

	t.Run("in_mem_db+binary_encoding+AES_32", func(t *testing.T) {
		client, err := NewClient(Config{
			Database:             test.New(),
			MarshalFuncPair:      binaryMarshalFuncPair,
			ProcessorConstructor: encrypterDecrypterConstructor,
		})
		require.NoError(t, err)
		defer func() {
			err := client.Close()
			if err != nil {
				panic(err)
			}
		}()

		f(t, client)
	})

	t.Run("in_mem_db+Snappy_default_compression+AES_32", func(t *testing.T) {
		client, err := NewClient(Config{
			Database:             test.New(),
			ProcessorConstructor: processorChainConstructor,
		})
		require.NoError(t, err)
		defer func() {
			err := client.Close()
			if err != nil {
				panic(err)
			}
		}()

		f(t, client)
	})
}

// testRoundTrip simply tests test the client's set-get-delete cycle
// for all kinds of metadata.
func testRoundTrip(t *testing.T, c *Client) {
	require := require.New(t)
	require.NotNil(c)

	// prepare the data
	md := metatypes.Metadata{
		Key:            []byte("two"),
		Size:           42,
		CreationEpoch:  123456789,
		LastWriteEpoch: 123456789,
		Chunks: []metatypes.Chunk{
			{
				Size:    math.MaxInt64,
				Hash:    []byte("foo"),
				Objects: nil,
			},
			{
				Size: 1234,
				Hash: []byte("bar"),
				Objects: []metatypes.Object{
					{
						Key:     []byte("foo"),
						ShardID: "bar",
					},
				},
			},
			{
				Size: 2,
				Hash: []byte("baz"),
				Objects: []metatypes.Object{
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
	require.Equal(ErrNotFound, err)

	// set the metadata
	err = c.SetMetadata(md)
	require.NoError(err)

	// get it back
	storedMd, err := c.GetMetadata(md.Key)
	require.NoError(err)

	// check stored value
	require.NotNil(storedMd)
	require.Equal(md, *storedMd)

	err = c.DeleteMetadata(md.Key)
	require.NoError(err)
	// make sure we can't get it back
	_, err = c.GetMetadata(md.Key)
	require.Equal(ErrNotFound, err)
}

// testClientNilKeys tests that the given client
// returns the correct ErrNilKey error when no key is given
// for the given functions.
func testClientNilKeys(t *testing.T, c *Client) {
	require := require.New(t)
	require.NotNil(c)

	_, err := c.GetMetadata(nil)
	require.Equal(ErrNilKey, err)

	err = c.SetMetadata(metatypes.Metadata{})
	require.Equal(ErrNilKey, err)

	_, err = c.UpdateMetadata(nil,
		func(metatypes.Metadata) (*metatypes.Metadata, error) { return nil, nil })
	require.Equal(ErrNilKey, err)

	err = c.DeleteMetadata(nil)
	require.Equal(ErrNilKey, err)
}

// testClientUpdate tests that the given function
// can Update existing metadata in a synchronous scenario.
func testClientUpdate(t *testing.T, c *Client) {
	require := require.New(t)
	require.NotNil(c)

	_, err := c.UpdateMetadata(nil,
		func(md metatypes.Metadata) (*metatypes.Metadata, error) { return &md, nil })
	require.Equal(ErrNilKey, err)

	_, err = c.UpdateMetadata([]byte("foo"),
		func(md metatypes.Metadata) (*metatypes.Metadata, error) { return &md, nil })
	require.Equal(ErrNotFound, err)

	err = c.SetMetadata(metatypes.Metadata{Key: []byte("foo")})
	require.NoError(err)

	require.Panics(func() {
		c.UpdateMetadata([]byte("foo"), nil)
	}, "no callback given")

	md, err := c.GetMetadata([]byte("foo"))
	require.NoError(err)
	require.Equal("foo", string(md.Key))
	require.Equal(int64(0), md.Size)

	md, err = c.UpdateMetadata([]byte("foo"),
		func(md metatypes.Metadata) (*metatypes.Metadata, error) {
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

// testClientUpdateAsync tests that the given function
// can Update existing metadata in an asynchronous scenario.
func testClientUpdateAsync(t *testing.T, c *Client) {
	require := require.New(t)
	require.NotNil(c)

	const (
		jobs = 128
	)
	var (
		err error
		key = []byte("foo")
	)

	err = c.SetMetadata(metatypes.Metadata{Key: key})
	require.NoError(err)

	group, _ := errgroup.WithContext(context.Background())
	for i := 0; i < jobs; i++ {
		i := i
		group.Go(func() error {
			var expectedSize int64
			md, err := c.UpdateMetadata(key,
				func(md metatypes.Metadata) (*metatypes.Metadata, error) {
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

func binaryMetadataMarshal(md metatypes.Metadata) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	encoder := gob.NewEncoder(buf)
	err := encoder.Encode(md)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func binaryMetadataUnmarshal(b []byte, md *metatypes.Metadata) error {
	decoder := gob.NewDecoder(bytes.NewReader(b))
	return decoder.Decode(md)
}

var (
	binaryMarshalFuncPair = &encoding.MarshalFuncPair{
		Marshal:   binaryMetadataMarshal,
		Unmarshal: binaryMetadataUnmarshal,
	}

	encrypterDecrypterConstructor = func() (processing.Processor, error) {
		return processing.NewAESEncrypterDecrypter(
			[]byte("01234567890123456789012345678901"))
	}

	processorChainConstructor = func() (processing.Processor, error) {
		cd, err := processing.NewSnappyCompressorDecompressor(processing.CompressionModeDefault)
		if err != nil {
			return nil, err
		}
		ec, err := processing.NewAESEncrypterDecrypter(
			[]byte("01234567890123456789012345678901"))
		if err != nil {
			return nil, err
		}
		return processing.NewProcessorChain([]processing.Processor{cd, ec}), nil
	}
)
