package meta

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMetaSize(t *testing.T) {
	t.Run("capnp", func(t *testing.T) {
		meta := createCapnpMeta(t)

		capnpBuf := &bytes.Buffer{}
		err := meta.Encode(capnpBuf)
		require.NoError(t, err)
		fmt.Printf("size capnp: %d\n", len(capnpBuf.Bytes()))
	})

	t.Run("capnp-packed", func(t *testing.T) {
		meta := createCapnpMeta(t)

		capnpBuf := &bytes.Buffer{}
		err := meta.EncodePacked(capnpBuf)
		require.NoError(t, err)
		fmt.Printf("size capnp-packed: %d\n", len(capnpBuf.Bytes()))
	})
}

func createCapnpMeta(t testing.TB) *Meta {
	chunks := make([]*Chunk, 256)
	for i := range chunks {
		chunks[i] = &Chunk{
			Key:  []byte(fmt.Sprintf("chunk%d", i)),
			Size: 1024,
		}
		chunks[i].Shards = make([]string, 5)
		for y := range chunks[i].Shards {
			chunks[i].Shards[y] = fmt.Sprintf("http://127.0.0.1:12345/stor-%d", i)
		}
	}

	meta := New([]byte("testkey"))
	meta.Previous = []byte("previous")
	meta.Next = []byte("next")
	meta.Chunks = chunks

	return meta
}

func BenchmarkEncoding(b *testing.B) {
	b.Run("capnp", func(b *testing.B) {
		meta := createCapnpMeta(b)
		buf := &bytes.Buffer{}
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			err := meta.Encode(buf)
			if err != nil {
				b.Error(err)
			}
			buf.Reset()
		}
	})

	b.Run("capnp-packed", func(b *testing.B) {
		meta := createCapnpMeta(b)
		buf := &bytes.Buffer{}
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			err := meta.EncodePacked(buf)
			if err != nil {
				b.Error(err)
			}
			buf.Reset()
		}
	})
}
