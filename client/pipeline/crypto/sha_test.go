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

package crypto

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSumSHA256(t *testing.T) {
	testSumFunc(t, SumSHA256, 32)
}

func TestSHA256Hasher_WithoutKey(t *testing.T) {
	h, err := NewSHA256Hasher(nil)
	require.NoError(t, err)
	testSumFunc(t, h.HashBytes, 32)
}

func TestSHA256Hasher_WithKey_64(t *testing.T) {
	h, err := NewSHA256Hasher(
		[]byte("01234567"))
	require.NoError(t, err)
	testSumFunc(t, h.HashBytes, 32)
}

func TestSHA256Hasher_WithKey_256(t *testing.T) {
	h, err := NewSHA256Hasher([]byte("01234567890123456789012345678901"))
	require.NoError(t, err)
	testSumFunc(t, h.HashBytes, 32)
}

func TestSumSHA512(t *testing.T) {
	testSumFunc(t, SumSHA512, 64)
}

func TestSHA512Hasher_WithoutKey(t *testing.T) {
	h, err := NewSHA512Hasher(nil)
	require.NoError(t, err)
	testSumFunc(t, h.HashBytes, 64)
}

func TestSHA512Hasher_WithKey_64(t *testing.T) {
	h, err := NewSHA512Hasher(
		[]byte("01234567"))
	require.NoError(t, err)
	testSumFunc(t, h.HashBytes, 64)
}

func TestSHA512Hasher_WithKey_256(t *testing.T) {
	h, err := NewSHA512Hasher(
		[]byte("01234567890123456789012345678901"))
	require.NoError(t, err)
	testSumFunc(t, h.HashBytes, 64)
}

func TestSHA512Hasher_WithKey_512(t *testing.T) {
	h, err := NewSHA512Hasher(
		[]byte("0123456789012345678901234567890101234567890123456789012345678901"))
	require.NoError(t, err)
	testSumFunc(t, h.HashBytes, 64)
}

func BenchmarkSumSHA256(b *testing.B) {
	b.Run("512-bytes", func(b *testing.B) {
		benchmarkHashFunc(b, SumSHA256, 512, 32)
	})
	b.Run("4096-bytes", func(b *testing.B) {
		benchmarkHashFunc(b, SumSHA256, 4096, 32)
	})
	b.Run("131072-bytes", func(b *testing.B) {
		benchmarkHashFunc(b, SumSHA256, 131072, 32)
	})
}

func BenchmarkSumSHA512(b *testing.B) {
	b.Run("512-bytes", func(b *testing.B) {
		benchmarkHashFunc(b, SumSHA512, 512, 64)
	})
	b.Run("4096-bytes", func(b *testing.B) {
		benchmarkHashFunc(b, SumSHA512, 4096, 64)
	})
	b.Run("131072-bytes", func(b *testing.B) {
		benchmarkHashFunc(b, SumSHA512, 131072, 64)
	})
}

func BenchmarkSHA256Hasher_WithoutKey(b *testing.B) {
	b.Run("512-bytes", func(b *testing.B) {
		hasher, err := NewSHA256Hasher(nil)
		if err != nil {
			b.Error(err)
		}
		benchmarkHashFunc(b, hasher.HashBytes, 512, 32)
	})
	b.Run("4096-bytes", func(b *testing.B) {
		hasher, err := NewSHA256Hasher(nil)
		if err != nil {
			b.Error(err)
		}
		benchmarkHashFunc(b, hasher.HashBytes, 4096, 32)
	})
	b.Run("131072-bytes", func(b *testing.B) {
		hasher, err := NewSHA256Hasher(nil)
		if err != nil {
			b.Error(err)
		}
		benchmarkHashFunc(b, hasher.HashBytes, 131072, 32)
	})
	b.Run("1048576-bytes", func(b *testing.B) {
		hasher, err := NewSHA256Hasher(nil)
		if err != nil {
			b.Error(err)
		}
		benchmarkHashFunc(b, hasher.HashBytes, 1048576, 32)
	})
}

func BenchmarkSHA256Hasher_WithKey(b *testing.B) {
	b.Run("512-bytes", func(b *testing.B) {
		hasher, err := NewSHA256Hasher([]byte("012345678901234567890123456789001"))
		if err != nil {
			b.Error(err)
		}
		benchmarkHashFunc(b, hasher.HashBytes, 512, 32)
	})
	b.Run("4096-bytes", func(b *testing.B) {
		hasher, err := NewSHA256Hasher([]byte("012345678901234567890123456789001"))
		if err != nil {
			b.Error(err)
		}
		benchmarkHashFunc(b, hasher.HashBytes, 4096, 32)
	})
	b.Run("131072-bytes", func(b *testing.B) {
		hasher, err := NewSHA256Hasher([]byte("012345678901234567890123456789001"))
		if err != nil {
			b.Error(err)
		}
		benchmarkHashFunc(b, hasher.HashBytes, 131072, 32)
	})
	b.Run("1048576-bytes", func(b *testing.B) {
		hasher, err := NewSHA256Hasher([]byte("012345678901234567890123456789001"))
		if err != nil {
			b.Error(err)
		}
		benchmarkHashFunc(b, hasher.HashBytes, 1048576, 32)
	})
}

func BenchmarkSHA512Hasher_WithoutKey(b *testing.B) {
	b.Run("512-bytes", func(b *testing.B) {
		hasher, err := NewSHA512Hasher(nil)
		if err != nil {
			b.Error(err)
		}
		benchmarkHashFunc(b, hasher.HashBytes, 512, 64)
	})
	b.Run("4096-bytes", func(b *testing.B) {
		hasher, err := NewSHA512Hasher(nil)
		if err != nil {
			b.Error(err)
		}
		benchmarkHashFunc(b, hasher.HashBytes, 4096, 64)
	})
	b.Run("131072-bytes", func(b *testing.B) {
		hasher, err := NewSHA512Hasher(nil)
		if err != nil {
			b.Error(err)
		}
		benchmarkHashFunc(b, hasher.HashBytes, 131072, 64)
	})
	b.Run("1048576-bytes", func(b *testing.B) {
		hasher, err := NewSHA512Hasher(nil)
		if err != nil {
			b.Error(err)
		}
		benchmarkHashFunc(b, hasher.HashBytes, 1048576, 64)
	})
}

func BenchmarkSHA512Hasher_WithKey(b *testing.B) {
	b.Run("512-bytes", func(b *testing.B) {
		hasher, err := NewSHA512Hasher(
			[]byte("012345678901234567890123456789001012345678901234567890123456789001"))
		if err != nil {
			b.Error(err)
		}
		benchmarkHashFunc(b, hasher.HashBytes, 512, 64)
	})
	b.Run("4096-bytes", func(b *testing.B) {
		hasher, err := NewSHA512Hasher(
			[]byte("012345678901234567890123456789001012345678901234567890123456789001"))
		if err != nil {
			b.Error(err)
		}
		benchmarkHashFunc(b, hasher.HashBytes, 4096, 64)
	})
	b.Run("131072-bytes", func(b *testing.B) {
		hasher, err := NewSHA512Hasher(
			[]byte("012345678901234567890123456789001012345678901234567890123456789001"))
		if err != nil {
			b.Error(err)
		}
		benchmarkHashFunc(b, hasher.HashBytes, 131072, 64)
	})
	b.Run("1048576-bytes", func(b *testing.B) {
		hasher, err := NewSHA512Hasher(
			[]byte("012345678901234567890123456789001012345678901234567890123456789001"))
		if err != nil {
			b.Error(err)
		}
		benchmarkHashFunc(b, hasher.HashBytes, 1048576, 64)
	})
}
