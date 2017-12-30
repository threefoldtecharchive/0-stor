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
	"crypto/rand"
	"math"
	mathRand "math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

func testStorageReadCheckWriteDelete(t *testing.T, storage ChunkStorage) {
	require.NotNil(t, storage)

	t.Run("fixed test cases", func(t *testing.T) {
		dataCases := [][]byte{
			[]byte("b"),
			[]byte("bar"),
			[]byte("大家好"),
			[]byte("Hello, World!"),
			[]byte("Hello, World!"),
		}
		for _, data := range dataCases {
			testStorageReadCheckWriteDeleteCycle(t, storage, data)
		}
	})

	t.Run("random test cases", func(t *testing.T) {
		for i := 0; i < 256; i++ {
			data := make([]byte, mathRand.Int31n(128)+1)
			rand.Read(data)
			testStorageReadCheckWriteDeleteCycle(t, storage, data)
		}
	})
}

func testStorageReadCheckWriteDeleteCycle(t *testing.T, storage ChunkStorage, data []byte) {
	require := require.New(t)

	// write object & validate
	cfg, err := storage.WriteChunk(data)
	require.NoError(err)
	require.NotNil(cfg)
	require.Equal(int64(len(data)), cfg.Size)

	// validate that all shards contain valid data
	status, err := storage.CheckChunk(*cfg, false)
	require.NoError(err)
	require.Equal(CheckStatusOptimal, status)

	// read object & validate
	output, err := storage.ReadChunk(*cfg)
	require.NoError(err)
	require.Equal(data, output)

	// delete the object
	err = storage.DeleteChunk(*cfg)
	require.NoError(err)

	// validate the object is invalid now (as it should be deleted)
	status, err = storage.CheckChunk(*cfg, false)
	require.NoError(err)
	require.Equal(CheckStatusInvalid, status)
}

func TestCheckStatusString(t *testing.T) {
	require := require.New(t)

	// valid enum values
	require.Equal("invalid", CheckStatusInvalid.String())
	require.Equal("valid", CheckStatusValid.String())
	require.Equal("optimal", CheckStatusOptimal.String())

	// invalid enum value
	require.Empty(CheckStatus(math.MaxUint8).String())
}
