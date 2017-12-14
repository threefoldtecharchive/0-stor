package storage

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewRandomStoragePanics(t *testing.T) {
	require.Panics(t, func() {
		NewRandomObjectStorage(nil)
	}, "no cluster given")
}

func TestRandomStorageReadCheckWrite(t *testing.T) {
	cluster, cleanup, err := newGRPCServerCluster(3)
	require.NoError(t, err)
	defer cleanup()

	storage, err := NewRandomObjectStorage(cluster)
	require.NoError(t, err)

	testStorageReadCheckWrite(t, storage)
}

func TestRandomStorageRepair(t *testing.T) {
	require := require.New(t)

	storage, err := NewRandomObjectStorage(dummyCluster{})
	require.NoError(err)

	cfg, err := storage.Repair(ObjectConfig{})
	require.Equal(ErrNotSupported, err)
	require.Equal(ObjectConfig{}, cfg)
}
