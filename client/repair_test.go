package client

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zero-os/0-stor/client/pipeline/storage"
)

func TestRepair(t *testing.T) {
	servers, serverClean := testGRPCServer(t, 4)
	defer serverClean()

	shards := make([]string, len(servers))
	for i, server := range servers {
		shards[i] = server.Address()
	}

	config := newDefaultConfig(shards, 1024)

	tt := []struct {
		name string

		DataShardCount   int
		ParityShardCount int

		repairErr error
	}{
		{
			name:           "replication",
			DataShardCount: 4,
			repairErr:      nil,
		},
		{
			name:             "distribution",
			DataShardCount:   3,
			ParityShardCount: 1,
			repairErr:        nil,
		},
		{
			name:             "no-repair-suport",
			DataShardCount:   0,
			ParityShardCount: 0,
			repairErr:        ErrRepairSupport,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			config.Pipeline.Distribution.DataShardCount = tc.DataShardCount
			config.Pipeline.Distribution.ParityShardCount = tc.ParityShardCount
			testRepair(t, config, tc.repairErr)
		})
	}
}

func testRepair(t *testing.T, config Config, repairErr error) {
	c, err := getTestClient(config)
	require.NoError(t, err, "fail to create client")
	defer c.Close()

	data := make([]byte, 1204*10)
	key := make([]byte, 64)

	_, err = rand.Read(data)
	require.NoError(t, err, "fail to read random data")
	_, err = rand.Read(key)
	require.NoError(t, err, "fail to read random key")

	meta, err := c.Write(key, data)
	require.NoError(t, err, "fail write data")

	// Check status is ok after a write
	status, err := c.Check(meta.Key)
	require.NoError(t, err, "fail to check object")
	require.True(t, status == storage.CheckStatusValid || status == storage.CheckStatusOptimal)

	// corrupt file by removing a block
	store, err := c.datastorCluster.GetShard(meta.Chunks[0].Objects[0].ShardID)
	require.NoError(t, err)
	err = store.DeleteObject(meta.Chunks[0].Objects[0].Key)
	require.NoError(t, err)

	// Check status is corrupted
	status, err = c.Check(meta.Key)
	require.NoError(t, err, "fail to check object")
	require.True(t, status == storage.CheckStatusValid || status == storage.CheckStatusInvalid)

	// try to repair
	err = c.Repair(meta.Key)
	if repairErr != nil {
		assert.Error(t, repairErr, err)
	}

	if repairErr == nil {
		require.NoError(t, err)
		// make sure we can read the data again
		readData, err := c.Read(meta.Key)
		require.NoError(t, err)
		assert.Equal(t, data, readData, "restored data is not the same as initial data")
	}

}
