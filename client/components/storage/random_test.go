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

	storage := NewRandomObjectStorage(cluster)
	require.NotNil(t, storage)

	testStorageReadCheckWrite(t, storage)
}

func TestRandomStorageRepair(t *testing.T) {
	require := require.New(t)

	storage := NewRandomObjectStorage(dummyCluster{})
	require.NotNil(storage)

	defer func() {
		err := storage.Close()
		require.NoError(err)
	}()

	cfg, err := storage.Repair(ObjectConfig{})
	require.Equal(ErrNotSupported, err)
	require.Equal(ObjectConfig{}, cfg)
}
