package client

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClientTraverse(t *testing.T) {
	testTraverse(t, true)
}

func TestClientTraversePostOrder(t *testing.T) {
	testTraverse(t, false)
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
			err = cli.Write(key, bytes.NewReader(values[i]))
		} else {
			err = cli.WriteLinked(key, prevKey, bytes.NewReader(values[i]))
		}
		require.NoError(t, err)
		prevKey = key
	}

	endEpoch := EpochNow()

	// walk over it
	var it TraverseIterator
	if forward {
		it, err = cli.Traverse(firstKey, startEpoch, endEpoch)
		require.NoError(t, err)
	} else {
		it, err = cli.TraversePostOrder(lastKey, endEpoch, startEpoch)
		require.NoError(t, err)
	}

	var i int
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
		require.Equal(t, values[idx], buf.Bytes())

		i++
	}

	require.Equal(t, 100, i)
}
