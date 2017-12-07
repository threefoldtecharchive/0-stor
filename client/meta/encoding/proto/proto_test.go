package proto

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zero-os/0-stor/client/meta"
)

func TestMarshalUnmarshal(t *testing.T) {
	require := require.New(t)

	dataSlice := []meta.Data{
		meta.Data{
			Key: []byte("foo"),
		},
		meta.Data{
			Key:   []byte("bar"),
			Epoch: 42,
		},
		meta.Data{
			Key:      []byte("baz"),
			Epoch:    math.MaxInt64,
			Next:     []byte("foo"),
			Previous: []byte("baz"),
		},
		meta.Data{
			Key:   []byte("two"),
			Epoch: 123456789,
			Chunks: []*meta.Chunk{
				&meta.Chunk{
					Size:   math.MaxInt64,
					Key:    []byte("foo"),
					Shards: nil,
				},
				&meta.Chunk{
					Size:   1234,
					Key:    []byte("bar"),
					Shards: []string{"foo"},
				},
				&meta.Chunk{
					Size:   2,
					Key:    []byte("baz"),
					Shards: []string{"bar", "foo"},
				},
			},
			Next:     []byte("one"),
			Previous: []byte("three"),
		},
	}

	for _, input := range dataSlice {
		bytes, err := MarshalMetadata(input)
		require.NoError(err)
		require.NotNil(bytes)

		var output meta.Data
		err = UnmarshalMetadata(bytes, &output)
		require.NoError(err)
		require.Equal(input, output)
	}
}

func TestUnmarshalExplicitPanics(t *testing.T) {
	require := require.New(t)

	require.Panics(func() {
		UnmarshalMetadata(nil, &meta.Data{})
	}, "no data given to unmarshal")
	require.Panics(func() {
		UnmarshalMetadata([]byte("foo"), nil)
	}, "no meta.Data pointer given to unmarshal to")
}

func TestUnmarshalExplicitErrors(t *testing.T) {
	var data meta.Data
	require.Error(t, UnmarshalMetadata([]byte("foo"), &data))
}

func TestMetaSize(t *testing.T) {
	require := require.New(t)

	input := createMeta(t)

	bytes, err := MarshalMetadata(input)
	require.NoError(err)
	require.NotNil(bytes)

	t.Logf("size proto: %d\n", len(bytes))

	var output meta.Data
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

func createMeta(t testing.TB) meta.Data {
	chunks := make([]*meta.Chunk, 256)
	for i := range chunks {
		chunks[i] = &meta.Chunk{
			Key:  []byte(fmt.Sprintf("chunk%d", i)),
			Size: 1024,
		}
		chunks[i].Shards = make([]string, 5)
		for y := range chunks[i].Shards {
			chunks[i].Shards[y] = fmt.Sprintf("http://127.0.0.1:12345/stor-%d", i)
		}
	}

	return meta.Data{
		Key:      []byte("testkey"),
		Previous: []byte("previous"),
		Next:     []byte("next"),
		Chunks:   chunks,
	}
}
