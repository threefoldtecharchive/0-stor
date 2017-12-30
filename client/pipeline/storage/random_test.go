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

func TestRandomStorageReadCheckWriteDelete(t *testing.T) {
	cluster, cleanup, err := newGRPCServerCluster(3)
	require.NoError(t, err)
	defer cleanup()
	defer cluster.Close()

	storage, err := NewRandomChunkStorage(cluster)
	require.NoError(t, err)

	testStorageReadCheckWriteDelete(t, storage)
}

func TestRandomStorageRepair(t *testing.T) {
	require := require.New(t)

	storage, err := NewRandomChunkStorage(dummyCluster{})
	require.NoError(err)

	cfg, err := storage.RepairChunk(ChunkConfig{})
	require.Equal(ErrNotSupported, err)
	require.Nil(cfg)
}
