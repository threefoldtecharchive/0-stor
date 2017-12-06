package crypto

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewBlake2b256Hasher(t *testing.T) {
	h, err := NewBlake2b256Hasher(make([]byte, 65))
	require.Error(t, err, "key too large")
	require.Nil(t, h)
}

func TestNewBlake2b512Hasher(t *testing.T) {
	h, err := NewBlake2b512Hasher(make([]byte, 65))
	require.Error(t, err, "key too large")
	require.Nil(t, h)
}

func TestSumBlake2b256(t *testing.T) {
	testSumFunc(t, SumBlake2b256, 32)
}

func TestSumBlake2b512(t *testing.T) {
	testSumFunc(t, SumBlake2b512, 64)
}

func TestSumBlake2b256er_WithoutKey(t *testing.T) {
	h, err := NewBlake2b256Hasher(nil)
	require.NoError(t, err)
	testSumFunc(t, h.HashBytes, 32)
}

func TestSumBlake2b256er_WithKey(t *testing.T) {
	h, err := NewBlake2b256Hasher([]byte("01234567890123456789012345678901"))
	require.NoError(t, err)
	testSumFunc(t, h.HashBytes, 32)
}

func TestSumBlake2b512er_WithoutKey(t *testing.T) {
	h, err := NewBlake2b512Hasher(nil)
	require.NoError(t, err)
	testSumFunc(t, h.HashBytes, 64)
}

func TestSumBlake2b512er_WithKey(t *testing.T) {
	h, err := NewBlake2b512Hasher(
		[]byte("0123456789012345678901234567890101234567890123456789012345678901"))
	require.NoError(t, err)
	testSumFunc(t, h.HashBytes, 64)
}

func BenchmarkSumBlake2b256(b *testing.B) {
	b.Run("512-bytes", func(b *testing.B) {
		benchmarkHashFunc(b, SumBlake2b256, 512, 32)
	})
	b.Run("4096-bytes", func(b *testing.B) {
		benchmarkHashFunc(b, SumBlake2b256, 4096, 32)
	})
	b.Run("131072-bytes", func(b *testing.B) {
		benchmarkHashFunc(b, SumBlake2b256, 131072, 32)
	})
}

func BenchmarkSumBlake2b512(b *testing.B) {
	b.Run("512-bytes", func(b *testing.B) {
		benchmarkHashFunc(b, SumBlake2b512, 512, 64)
	})
	b.Run("4096-bytes", func(b *testing.B) {
		benchmarkHashFunc(b, SumBlake2b512, 4096, 64)
	})
	b.Run("131072-bytes", func(b *testing.B) {
		benchmarkHashFunc(b, SumBlake2b512, 131072, 64)
	})
}

func BenchmarkSumBlake2b256er_WithoutKey(b *testing.B) {
	b.Run("512-bytes", func(b *testing.B) {
		hasher, err := NewBlake2b256Hasher(nil)
		if err != nil {
			b.Error(err)
		}
		benchmarkHashFunc(b, hasher.HashBytes, 512, 32)
	})
	b.Run("4096-bytes", func(b *testing.B) {
		hasher, err := NewBlake2b256Hasher(nil)
		if err != nil {
			b.Error(err)
		}
		benchmarkHashFunc(b, hasher.HashBytes, 4096, 32)
	})
	b.Run("131072-bytes", func(b *testing.B) {
		hasher, err := NewBlake2b256Hasher(nil)
		if err != nil {
			b.Error(err)
		}
		benchmarkHashFunc(b, hasher.HashBytes, 131072, 32)
	})
}

func BenchmarkSumBlake2b256er_WithKey(b *testing.B) {
	b.Run("512-bytes", func(b *testing.B) {
		hasher, err := NewBlake2b256Hasher([]byte("01234567890123456789012345678901"))
		if err != nil {
			b.Error(err)
		}
		benchmarkHashFunc(b, hasher.HashBytes, 512, 32)
	})
	b.Run("4096-bytes", func(b *testing.B) {
		hasher, err := NewBlake2b256Hasher([]byte("01234567890123456789012345678901"))
		if err != nil {
			b.Error(err)
		}
		benchmarkHashFunc(b, hasher.HashBytes, 4096, 32)
	})
	b.Run("131072-bytes", func(b *testing.B) {
		hasher, err := NewBlake2b256Hasher([]byte("01234567890123456789012345678901"))
		if err != nil {
			b.Error(err)
		}
		benchmarkHashFunc(b, hasher.HashBytes, 131072, 32)
	})
}

func BenchmarkSumBlake2b512er_WithoutKey(b *testing.B) {
	b.Run("512-bytes", func(b *testing.B) {
		hasher, err := NewBlake2b512Hasher(nil)
		if err != nil {
			b.Error(err)
		}
		benchmarkHashFunc(b, hasher.HashBytes, 512, 64)
	})
	b.Run("4096-bytes", func(b *testing.B) {
		hasher, err := NewBlake2b512Hasher(nil)
		if err != nil {
			b.Error(err)
		}
		benchmarkHashFunc(b, hasher.HashBytes, 4096, 64)
	})
	b.Run("131072-bytes", func(b *testing.B) {
		hasher, err := NewBlake2b512Hasher(nil)
		if err != nil {
			b.Error(err)
		}
		benchmarkHashFunc(b, hasher.HashBytes, 131072, 64)
	})
}

func BenchmarkSumBlake2b512er_WithKey(b *testing.B) {
	b.Run("512-bytes", func(b *testing.B) {
		hasher, err := NewBlake2b512Hasher(
			[]byte("0123456789012345678901234567890101234567890123456789012345678901"))
		if err != nil {
			b.Error(err)
		}
		benchmarkHashFunc(b, hasher.HashBytes, 512, 64)
	})
	b.Run("4096-bytes", func(b *testing.B) {
		hasher, err := NewBlake2b512Hasher(
			[]byte("0123456789012345678901234567890101234567890123456789012345678901"))
		if err != nil {
			b.Error(err)
		}
		benchmarkHashFunc(b, hasher.HashBytes, 4096, 64)
	})
	b.Run("131072-bytes", func(b *testing.B) {
		hasher, err := NewBlake2b512Hasher(
			[]byte("0123456789012345678901234567890101234567890123456789012345678901"))
		if err != nil {
			b.Error(err)
		}
		benchmarkHashFunc(b, hasher.HashBytes, 131072, 64)
	})
}
