package crypto

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSHA256Hash(t *testing.T) {
	testHashFunc(t, SHA256Hash, 32)
}

func TestSHA256Hasher(t *testing.T) {
	h, err := NewSHA256hasher()
	require.NoError(t, err)
	testHashFunc(t, h.HashBytes, 32)
}

func TestSHA512Hash(t *testing.T) {
	testHashFunc(t, SHA512Hash, 64)
}

func TestSHA512Hasher(t *testing.T) {
	h, err := NewSHA512hasher()
	require.NoError(t, err)
	testHashFunc(t, h.HashBytes, 64)
}

func BenchmarkSHA256Hash(b *testing.B) {
	b.Run("512-bytes", func(b *testing.B) {
		benchmarkHashFunc(b, SHA256Hash, 512, 32)
	})
	b.Run("4096-bytes", func(b *testing.B) {
		benchmarkHashFunc(b, SHA256Hash, 4096, 32)
	})
	b.Run("131072-bytes", func(b *testing.B) {
		benchmarkHashFunc(b, SHA256Hash, 131072, 32)
	})
}

func BenchmarkSHA512Hash(b *testing.B) {
	b.Run("512-bytes", func(b *testing.B) {
		benchmarkHashFunc(b, SHA512Hash, 512, 64)
	})
	b.Run("4096-bytes", func(b *testing.B) {
		benchmarkHashFunc(b, SHA512Hash, 4096, 64)
	})
	b.Run("131072-bytes", func(b *testing.B) {
		benchmarkHashFunc(b, SHA512Hash, 131072, 64)
	})
}

func BenchmarkSHA256Hasher(b *testing.B) {
	b.Run("512-bytes", func(b *testing.B) {
		hasher, err := NewSHA256hasher()
		if err != nil {
			b.Error(err)
		}
		benchmarkHashFunc(b, hasher.HashBytes, 512, 32)
	})
	b.Run("4096-bytes", func(b *testing.B) {
		hasher, err := NewSHA256hasher()
		if err != nil {
			b.Error(err)
		}
		benchmarkHashFunc(b, hasher.HashBytes, 4096, 32)
	})
	b.Run("131072-bytes", func(b *testing.B) {
		hasher, err := NewSHA256hasher()
		if err != nil {
			b.Error(err)
		}
		benchmarkHashFunc(b, hasher.HashBytes, 131072, 32)
	})
}

func BenchmarkSHA512Hasher(b *testing.B) {
	b.Run("512-bytes", func(b *testing.B) {
		hasher, err := NewSHA512hasher()
		if err != nil {
			b.Error(err)
		}
		benchmarkHashFunc(b, hasher.HashBytes, 512, 64)
	})
	b.Run("4096-bytes", func(b *testing.B) {
		hasher, err := NewSHA512hasher()
		if err != nil {
			b.Error(err)
		}
		benchmarkHashFunc(b, hasher.HashBytes, 4096, 64)
	})
	b.Run("131072-bytes", func(b *testing.B) {
		hasher, err := NewSHA512hasher()
		if err != nil {
			b.Error(err)
		}
		benchmarkHashFunc(b, hasher.HashBytes, 131072, 64)
	})
}
