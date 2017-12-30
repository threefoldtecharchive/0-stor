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

package pipeline

import (
	"bytes"
	"crypto/rand"
	mathRand "math/rand"
	"testing"

	"github.com/zero-os/0-stor/client/pipeline/storage"

	"github.com/stretchr/testify/require"
)

func testPipelineWriteReadDeleteCheck(t *testing.T, pipeline Pipeline) {
	t.Run("fixed-data", func(t *testing.T) {
		testCases := []string{
			"a",
			"Hello, World!",
			"大家好",
			"This... is my finger :)",
		}
		for _, testCase := range testCases {
			testPipelineWriteReadDeleteCheckCycle(t, pipeline, []byte(testCase))
		}
	})

	t.Run("random-data", func(t *testing.T) {
		for i := 0; i < 8; i++ {
			inputData := make([]byte, mathRand.Int31n(512)+1)
			rand.Read(inputData)

			testPipelineWriteReadDeleteCheckCycle(t, pipeline, []byte(inputData))
		}
	})
}

func testPipelineWriteReadDeleteCheckCycle(t *testing.T, pipeline Pipeline, input []byte) {
	require := require.New(t)

	buf := bytes.NewBuffer(nil)

	err := pipeline.Read(nil, buf)
	require.Error(err, "no chunks given to read")
	_, err = pipeline.Check(nil, true)
	require.Error(err, "no chunks given to check")
	_, err = pipeline.Repair(nil)
	require.Error(err, "no chunks given to repair")
	err = pipeline.Delete(nil)
	require.Error(err, "no chunks given to delete")

	chunks, err := pipeline.Write(bytes.NewReader(input))
	require.NoError(err)
	require.NotEmpty(chunks)

	status, err := pipeline.Check(chunks, false)
	require.NoError(err)
	require.Equal(storage.CheckStatusOptimal, status)
	status, err = pipeline.Check(chunks, true)
	require.NoError(err)
	require.NotEqual(storage.CheckStatusInvalid, status)

	err = pipeline.Read(chunks, buf)
	require.NoError(err)
	require.Equal(input, buf.Bytes())
	buf.Reset()

	// let's pwn the hash
	chunk := chunks[0]
	chunks[0].Hash = []byte("foo")

	// if the hash of a chunk is invalid, check will pass, but read will not
	status, err = pipeline.Check(chunks, false)
	require.NoError(err)
	require.Equal(storage.CheckStatusOptimal, status)
	status, err = pipeline.Check(chunks, true)
	require.NoError(err)
	require.NotEqual(storage.CheckStatusInvalid, status)
	err = pipeline.Read(chunks, buf)
	require.Error(err)
	buf.Reset()

	chunks[0] = chunk

	err = pipeline.Delete(chunks)
	require.NoError(err)

	err = pipeline.Read(chunks, buf)
	require.Error(err, "data is deleted and can't be read")

	status, err = pipeline.Check(chunks, false)
	require.NoError(err)
	require.Equal(storage.CheckStatusInvalid, status, "data is deleted and thus invalid")
	status, err = pipeline.Check(chunks, true)
	require.NoError(err)
	require.Equal(storage.CheckStatusInvalid, status, "data is deleted and thus invalid")

	_, err = pipeline.Repair(chunks)
	require.Error(err, "data is deleted and thus cannot be repaired")
}

func testPipelineCheckRepair(t *testing.T, pipeline Pipeline) {
	t.Run("fixed-data", func(t *testing.T) {
		testCases := []string{
			"a",
			"Hello, World!",
			"大家好",
			"This... is my finger :)",
		}
		for _, testCase := range testCases {
			testPipelineCheckRepairCycle(t, pipeline, []byte(testCase))
		}
	})

	t.Run("random-data", func(t *testing.T) {
		for i := 0; i < 8; i++ {
			inputData := make([]byte, mathRand.Int31n(512)+1)
			rand.Read(inputData)

			testPipelineCheckRepairCycle(t, pipeline, []byte(inputData))
		}
	})
}

func testPipelineCheckRepairCycle(t *testing.T, pipeline Pipeline, input []byte) {
	require := require.New(t)

	chunks, err := pipeline.Write(bytes.NewReader(input))
	require.NoError(err)
	require.NotEmpty(chunks)

	status, err := pipeline.Check(chunks, false)
	require.NoError(err)
	require.Equal(storage.CheckStatusOptimal, status)
	status, err = pipeline.Check(chunks, true)
	require.NoError(err)
	require.NotEqual(storage.CheckStatusInvalid, status)

	chunk := chunks[0]
	chunks[0].Objects = nil

	_, err = pipeline.Check(chunks, false)
	require.Error(err, "chunk #0 has no objects")
	_, err = pipeline.Repair(chunks)
	require.Error(err, "chunk #0 has no objects")

	chunks[0] = chunk
	chunks[0].Objects[0].Key = []byte("foo")

	status, err = pipeline.Check(chunks, false)
	require.NoError(err)
	require.NotEqual(storage.CheckStatusOptimal, status, "could be valid or invalid, but never optimal")
	status, err = pipeline.Check(chunks, true)
	require.NoError(err)
	require.NotEqual(storage.CheckStatusOptimal, status, "could be valid or invalid, but never optimal")

	if status != storage.CheckStatusInvalid {
		// let's try to repair it, as this should be possible when it's not invalid
		chunks, err = pipeline.Repair(chunks)
		require.NoError(err)

		// data should be fine once more
		status, err = pipeline.Check(chunks, false)
		require.NoError(err)
		require.Equal(storage.CheckStatusOptimal, status)
		status, err = pipeline.Check(chunks, true)
		require.NoError(err)
		require.NotEqual(storage.CheckStatusInvalid, status)
	}

	// now let's clean up
	err = pipeline.Delete(chunks)
	require.NoError(err)
}
