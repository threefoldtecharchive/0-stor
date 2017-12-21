package client

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zero-os/0-stor/client/pipeline/storage"
)

func TestCheck(t *testing.T) {
	servers, serverClean := testGRPCServer(t, 4)
	defer serverClean()

	shards := make([]string, len(servers))
	for i, server := range servers {
		shards[i] = server.Address()
	}

	config := newDefaultConfig(shards, 1024)

	c, err := getTestClient(config)
	require.NoError(t, err, "fail to create client")
	defer c.Close()

	data := make([]byte, 602*10)

	_, err = rand.Read(data)
	require.NoError(t, err, "fail to read random data")
	key := []byte("testkey")

	meta, err := c.Write(key, data)
	require.NoError(t, err, "fail write data")

	// Check status is ok after a write
	status, err := c.Check(meta.Key)
	require.NoError(t, err, "fail to check object")
	require.True(t, status == storage.ObjectCheckStatusValid || status == storage.ObjectCheckStatusOptimal)

	// corrupt file by removing blocks
	store, err := c.datastorCluster.GetShard(meta.Chunks[0].Shards[0])
	require.NoError(t, err)

	for i := 0; i < len(meta.Chunks); i += 4 {
		if i%4 == 0 {
			err = store.DeleteObject(meta.Chunks[i].Key)
			require.NoError(t, err)
		}
	}

	// Check status is corrupted
	status, err = c.Check(meta.Key)
	require.NoError(t, err, "fail to check object")
	require.True(t, status == storage.ObjectCheckStatusValid || status == storage.ObjectCheckStatusInvalid)
}
