package crypto

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewBlake2bHasher(t *testing.T) {
	h, err := NewBlake2bHasher(make([]byte, 65))
	require.Error(t, err, "key too large")
	require.Nil(t, h)
}

func TestBlake2bHash(t *testing.T) {
	testHashFunc(t, Blake2bHash, 32)
}

func TestBlake2bHasher_WithoutKey(t *testing.T) {
	h, err := NewBlake2bHasher(nil)
	require.NoError(t, err)
	testHashFunc(t, h.HashBytes, 32)
}

func TestBlake2bHasher_WithKey(t *testing.T) {
	h, err := NewBlake2bHasher([]byte("01234567890123456789012345678901"))
	require.NoError(t, err)
	testHashFunc(t, h.HashBytes, 32)
}

func BenchmarkBlake2bHash(b *testing.B) {
	b.Run("512-bytes", func(b *testing.B) {
		benchmarkHashFunc(b, Blake2bHash, 512, 32)
	})
	b.Run("4096-bytes", func(b *testing.B) {
		benchmarkHashFunc(b, Blake2bHash, 4096, 32)
	})
	b.Run("131072-bytes", func(b *testing.B) {
		benchmarkHashFunc(b, Blake2bHash, 131072, 32)
	})
}

func BenchmarkBlake2bHasher_WithoutKey(b *testing.B) {
	b.Run("512-bytes", func(b *testing.B) {
		hasher, err := NewBlake2bHasher(nil)
		if err != nil {
			b.Error(err)
		}
		benchmarkHashFunc(b, hasher.HashBytes, 512, 32)
	})
	b.Run("4096-bytes", func(b *testing.B) {
		hasher, err := NewBlake2bHasher(nil)
		if err != nil {
			b.Error(err)
		}
		benchmarkHashFunc(b, hasher.HashBytes, 4096, 32)
	})
	b.Run("131072-bytes", func(b *testing.B) {
		hasher, err := NewBlake2bHasher(nil)
		if err != nil {
			b.Error(err)
		}
		benchmarkHashFunc(b, hasher.HashBytes, 131072, 32)
	})
}

func BenchmarkBlake2bHasher_WithKey(b *testing.B) {
	b.Run("512-bytes", func(b *testing.B) {
		hasher, err := NewBlake2bHasher([]byte("01234567890123456789012345678901"))
		if err != nil {
			b.Error(err)
		}
		benchmarkHashFunc(b, hasher.HashBytes, 512, 32)
	})
	b.Run("4096-bytes", func(b *testing.B) {
		hasher, err := NewBlake2bHasher([]byte("01234567890123456789012345678901"))
		if err != nil {
			b.Error(err)
		}
		benchmarkHashFunc(b, hasher.HashBytes, 4096, 32)
	})
	b.Run("131072-bytes", func(b *testing.B) {
		hasher, err := NewBlake2bHasher([]byte("01234567890123456789012345678901"))
		if err != nil {
			b.Error(err)
		}
		benchmarkHashFunc(b, hasher.HashBytes, 131072, 32)
	})
}
