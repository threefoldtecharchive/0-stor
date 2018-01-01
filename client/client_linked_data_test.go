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

package client

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"strings"
	"testing"

	"github.com/zero-os/0-stor/client/metastor"
	"github.com/zero-os/0-stor/client/pipeline"

	"github.com/stretchr/testify/require"
)

func TestClient_WriteLinkedErrors(t *testing.T) {
	servers, serverClean := testGRPCServer(t, 1)
	defer serverClean()

	dataShards := []string{servers[0].Address()}
	config := newDefaultConfig(dataShards, 0)
	config.Pipeline.Distribution = pipeline.ObjectDistributionConfig{}

	cli, _, err := getTestClient(config)

	require := require.New(t)
	require.NoError(err)

	_, _, err = cli.WriteLinked(nil, []byte("bar"), bytes.NewReader(nil))
	require.Error(err, "no key given")
	_, _, err = cli.WriteLinked([]byte("foo"), nil, bytes.NewReader(nil))
	require.Error(err, "no prev-key given")
	_, _, err = cli.WriteLinked(nil, nil, bytes.NewReader(nil))
	require.Error(err, "no key or prev-key given")
	_, _, err = cli.WriteLinked([]byte("foo"), []byte("bar"), nil)
	require.Error(err, "no reader given")
	_, _, err = cli.WriteLinked(nil, nil, nil)
	require.Error(err, "nothing given")
}

func TestClient_WriteLinked(t *testing.T) {
	servers, serverClean := testGRPCServer(t, 1)
	defer serverClean()

	dataShards := []string{servers[0].Address()}
	config := newDefaultConfig(dataShards, 0)
	config.Pipeline.Distribution = pipeline.ObjectDistributionConfig{}

	cli, _, err := getTestClient(config)
	require := require.New(t)
	require.NoError(err)

	_, err = cli.Write([]byte{'a'}, strings.NewReader("foo"))
	require.NoError(err)
	_, _, err = cli.WriteLinked([]byte{'b'}, []byte{'a'}, strings.NewReader("bar"))
	require.NoError(err)
	_, _, err = cli.WriteLinked([]byte{'c'}, []byte{'b'}, strings.NewReader("baz"))
	require.NoError(err)

	md, err := cli.metastorClient.GetMetadata([]byte{'a'})
	require.NoError(err)
	require.NotNil(md)
	require.Equal("a", string(md.Key))
	require.Nil(md.PreviousKey)
	require.Equal("b", string(md.NextKey))

	md, err = cli.metastorClient.GetMetadata([]byte{'b'})
	require.NoError(err)
	require.NotNil(md)
	require.Equal("b", string(md.Key))
	require.Equal("a", string(md.PreviousKey))
	require.Equal("c", string(md.NextKey))

	md, err = cli.metastorClient.GetMetadata([]byte{'c'})
	require.NoError(err)
	require.NotNil(md)
	require.Equal("c", string(md.Key))
	require.Equal("b", string(md.PreviousKey))
	require.Nil(md.NextKey)
}

func TestClient_Traverse(t *testing.T) {
	testTraverse(t, true)
}

func TestClient_TraverseErrors(t *testing.T) {
	servers, serverClean := testGRPCServer(t, 1)
	defer serverClean()

	dataShards := []string{servers[0].Address()}
	config := newDefaultConfig(dataShards, 0)
	config.Pipeline.Distribution = pipeline.ObjectDistributionConfig{}

	cli, _, err := getTestClient(config)

	require := require.New(t)
	require.NoError(err)

	it, err := cli.Traverse(nil, 0, 0)
	require.Error(err, "no startKey given")
	require.Nil(it)
	it, err = cli.Traverse(nil, 1, 0)
	require.Error(err, "invalid epoch range given")
	require.Nil(it)
}

func TestClient_TraversePostOrder(t *testing.T) {
	testTraverse(t, false)
}

func TestClient_TraversePostOrderErrors(t *testing.T) {
	servers, serverClean := testGRPCServer(t, 1)
	defer serverClean()

	dataShards := []string{servers[0].Address()}
	config := newDefaultConfig(dataShards, 0)
	config.Pipeline.Distribution = pipeline.ObjectDistributionConfig{}

	cli, _, err := getTestClient(config)

	require := require.New(t)
	require.NoError(err)

	it, err := cli.TraversePostOrder(nil, 0, 0)
	require.Error(err, "no startKey given")
	require.Nil(it)
	it, err = cli.TraversePostOrder(nil, 0, 1)
	require.Error(err, "invalid epoch range given")
	require.Nil(it)
}

func testTraverse(t *testing.T, forward bool) {
	servers, serverClean := testGRPCServer(t, 4)
	defer serverClean()

	dataShards := make([]string, len(servers))
	for i, server := range servers {
		dataShards[i] = server.Address()
	}

	config := newDefaultConfig(dataShards, 0)

	cli, _, err := getTestClient(config)
	require.Nil(t, err)

	// create keys & data
	var keys [][]byte
	var values [][]byte

	// initialize the data
	for i := 0; i < 100; i++ {
		key := []byte(fmt.Sprintf("key#%d", i))
		keys = append(keys, key)

		val := make([]byte, 1024)
		rand.Read(val)
		values = append(values, val)
	}
	firstKey, lastKey := keys[0], keys[99]

	startEpoch := EpochNow()

	// do the write
	var prevKey []byte

	for i, key := range keys {
		if prevKey == nil {
			_, err = cli.Write(key, bytes.NewReader(values[i]))
		} else {
			_, _, err = cli.WriteLinked(key, prevKey, bytes.NewReader(values[i]))
		}
		require.NoError(t, err)
		prevKey = key
	}

	endEpoch := EpochNow()

	epochRanges := []struct {
		start, end int64
	}{
		{startEpoch, endEpoch},
		{0, endEpoch},
		{startEpoch, 0},
		{0, 0},
		{0, -1},
		{-1, 0},
		{-1, -1},
	}

	for _, epochRange := range epochRanges {
		// walk over it
		var it TraverseIterator
		if forward {
			it, err = cli.Traverse(firstKey, epochRange.start, epochRange.end)
			require.NoError(t, err)
		} else {
			it, err = cli.TraversePostOrder(lastKey, epochRange.end, epochRange.start)
			require.NoError(t, err)
		}

		_, err = it.GetMetadata()
		require.Error(t, err, "Next needs to be called first")
		err = it.ReadData(bytes.NewBuffer(nil))
		require.Error(t, err, "Next needs to be called first")

		require.Panics(t, func() {
			it.ReadData(nil)
		}, "no writer given")

		var (
			i                int
			lastMetadataRead metastor.Metadata
			lastDataRead     []byte
		)
		for it.Next() {
			idx := i
			if !forward {
				idx = len(keys) - i - 1
			}

			if i < 99 {
				idy := idx
				if forward {
					idy++
				} else {
					idy--
				}

				key, ok := it.PeekNextKey()
				require.True(t, ok)
				require.Equal(t, string(keys[idy]), string(key))
			} else {
				_, ok := it.PeekNextKey()
				require.False(t, ok)
			}

			md, err := it.GetMetadata()
			require.NoError(t, err)
			require.NotNil(t, md)
			require.Equal(t, keys[idx], md.Key)

			buf := bytes.NewBuffer(nil)
			err = it.ReadData(buf)
			require.NoError(t, err)
			dataRead := buf.Bytes()
			require.Equal(t, values[idx], dataRead)

			i++
			if i == 100 {
				lastMetadataRead = *md
				lastDataRead = make([]byte, len(dataRead))
				copy(lastDataRead, dataRead)
			}
		}
		require.Equal(t, 100, i)

		// iterator should now be invalid
		require.False(t, it.Next())
		_, ok := it.PeekNextKey()
		require.False(t, ok)

		// the latest values however will still be OK
		md, err := it.GetMetadata()
		require.NoError(t, err)
		require.NotNil(t, md)
		require.Equal(t, lastMetadataRead, *md)
		buf := bytes.NewBuffer(nil)
		err = it.ReadData(buf)
		require.NoError(t, err)
		require.Equal(t, lastDataRead, buf.Bytes())
	}
}
