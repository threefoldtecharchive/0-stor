package storage

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewRandomStoragePanics(t *testing.T) {
	require.Panics(t, func() {
		NewRandomChunkStorage(nil)
	}, "no cluster given")
}

func TestRandomStorageReadCheckWrite(t *testing.T) {
	cluster, cleanup, err := newGRPCServerCluster(3)
	require.NoError(t, err)
	defer cleanup()

	storage, err := NewRandomChunkStorage(cluster)
	require.NoError(t, err)

	testStorageReadCheckWrite(t, storage)
}

func TestRandomStorageRepair(t *testing.T) {
	require := require.New(t)

	storage, err := NewRandomChunkStorage(dummyCluster{})
	require.NoError(err)

	cfg, err := storage.RepairChunk(ChunkConfig{})
	require.Equal(ErrNotSupported, err)
	require.Nil(cfg)
}
