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

package proto

import (
	"fmt"
	"math"
	"testing"

	"github.com/zero-os/0-stor/client/metastor"

	"github.com/stretchr/testify/require"
)

func TestMarshalUnmarshal(t *testing.T) {
	require := require.New(t)

	metadataSlice := []metastor.Metadata{
		{
			Key: []byte("foo"),
		},
		{
			Key:           []byte("bar"),
			CreationEpoch: 42,
		},
		{
			Key:            []byte("42"),
			CreationEpoch:  42,
			LastWriteEpoch: 42,
		},
		{
			Key:            []byte("baz"),
			CreationEpoch:  math.MaxInt64,
			LastWriteEpoch: math.MinInt64,
			NextKey:        []byte("foo"),
			PreviousKey:    []byte("baz"),
		},
		{
			Key:            []byte("two"),
			CreationEpoch:  123456789,
			LastWriteEpoch: 123456789,
			Chunks: []metastor.Chunk{
				{
					Size:    math.MaxInt64,
					Objects: nil,
					Hash:    []byte("foo"),
				},
				{
					Size: 1234,
					Objects: []metastor.Object{
						{
							Key:     []byte("foo"),
							ShardID: "bar",
						},
					},
					Hash: []byte("bar"),
				},
				{
					Size: 2,
					Objects: []metastor.Object{
						{
							Key:     []byte("bar"),
							ShardID: "foo",
						},
						{
							Key:     []byte("foo"),
							ShardID: "bar",
						},
					},
					Hash: []byte("baz"),
				},
			},
			NextKey:     []byte("one"),
			PreviousKey: []byte("three"),
		},
	}

	for _, input := range metadataSlice {
		bytes, err := MarshalMetadata(input)
		require.NoError(err)
		require.NotNil(bytes)

		var output metastor.Metadata
		err = UnmarshalMetadata(bytes, &output)
		require.NoError(err)
		require.Equal(input, output)
	}
}

func TestUnmarshalExplicitPanics(t *testing.T) {
	require := require.New(t)

	require.Panics(func() {
		UnmarshalMetadata(nil, &metastor.Metadata{})
	}, "no data given to unmarshal")
	require.Panics(func() {
		UnmarshalMetadata([]byte("foo"), nil)
	}, "no metastor.Metadata pointer given to unmarshal to")
}

func TestUnmarshalExplicitErrors(t *testing.T) {
	var data metastor.Metadata
	require.Error(t, UnmarshalMetadata([]byte("foo"), &data))
}

func TestMetaSize(t *testing.T) {
	require := require.New(t)

	input := createMeta(t)

	bytes, err := MarshalMetadata(input)
	require.NoError(err)
	require.NotNil(bytes)

	t.Logf("size proto: %d\n", len(bytes))

	var output metastor.Metadata
	err = UnmarshalMetadata(bytes, &output)
	require.NoError(err)
	require.Equal(input, output)
}

func BenchmarkMarshalMetadata(b *testing.B) {
	meta := createMeta(b)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := MarshalMetadata(meta)
		if err != nil {
			b.Error(err)
		}
	}
}

func createMeta(t testing.TB) metastor.Metadata {
	chunks := make([]metastor.Chunk, 256)
	for i := range chunks {
		chunks[i] = metastor.Chunk{
			Hash: []byte(fmt.Sprintf("chunk%d", i)),
			Size: 1024,
		}
		chunks[i].Objects = make([]metastor.Object, 5)
		for y := range chunks[i].Objects {
			chunks[i].Objects[y] = metastor.Object{
				Key:     []byte(fmt.Sprintf("chunk%d", i)),
				ShardID: fmt.Sprintf("http://127.0.0.1:12345/stor-%d", i),
			}
		}
	}

	return metastor.Metadata{
		Key:         []byte("testkey"),
		PreviousKey: []byte("previous"),
		NextKey:     []byte("next"),
		Chunks:      chunks,
	}
}
