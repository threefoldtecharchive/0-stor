package crypto

import (
	"crypto/rand"
	"math"
	mrand "math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewHasherForType(t *testing.T) {
	require := require.New(t)

	testCases := []struct {
		Type     HashType
		Expected interface{}
	}{
		{HashTypeSHA256, (*SHA256Hasher)(nil)},
		{HashTypeSHA512, (*SHA512Hasher)(nil)},
		{HashTypeBlake2b, (*Blake2bHasher)(nil)},
		{math.MaxUint8, nil},
	}
	for _, tc := range testCases {
		h, err := NewHasherForType(tc.Type)
		if tc.Expected == nil {
			require.Error(err)
			require.Nil(h)
		} else {
			require.NoError(err)
			require.NotNil(h)
			require.IsType(tc.Expected, h)
		}
	}
}

func TestHashTypeMarshalUnmarshal(t *testing.T) {
	require := require.New(t)

	types := []HashType{
		HashTypeSHA256,
		HashTypeSHA512,
		HashTypeBlake2b,
	}
	for _, t := range types {
		b, err := t.MarshalText()
		require.NoError(err)
		require.NotNil(b)

		var o HashType
		err = o.UnmarshalText(b)
		require.NoError(err)
		require.Equal(t, o)
	}
}

func TestHashTypeMarshal(t *testing.T) {
	require := require.New(t)

	testCases := []struct {
		Type     HashType
		Expected string
	}{
		{HashTypeSHA256, "sha_256"},
		{HashTypeSHA512, "sha_512"},
		{HashTypeBlake2b, "blake2b_256"},
		{math.MaxUint8, ""},
	}
	for _, tc := range testCases {
		b, err := tc.Type.MarshalText()
		if tc.Expected == "" {
			require.Error(err)
			require.Nil(b)
		} else {
			require.NoError(err)
			require.Equal(tc.Expected, string(b))
		}
	}
}

func TestHashTypeUnmarshal(t *testing.T) {
	require := require.New(t)

	testCases := []struct {
		String   string
		Expected HashType
		Err      bool
	}{
		{"sha", HashTypeSHA256, false},
		{"SHA", HashTypeSHA256, false},
		{"sha_256", HashTypeSHA256, false},
		{"SHA_256", HashTypeSHA256, false},
		{"sha_512", HashTypeSHA512, false},
		{"SHA_512", HashTypeSHA512, false},
		{"Blake2b", HashTypeBlake2b, false},
		{"BLAKE2b", HashTypeBlake2b, false},
		{"blake2b_256", HashTypeBlake2b, false},
		{"BLAKE2B_256", HashTypeBlake2b, false},
		{"", math.MaxUint8, true},
	}
	for _, tc := range testCases {
		var o HashType
		err := o.UnmarshalText([]byte(tc.String))
		if tc.Err {
			require.Error(err)
			require.Equal(HashTypeSHA256, o)
		} else {
			require.NoError(err)
			require.Equal(tc.Expected, o)
		}
	}
}

func TestHashBytes(t *testing.T) {
	testHashFunc(t, HashBytes, 32)
}

func TestDefaultHasher(t *testing.T) {
	h, err := NewHasher()
	require.NoError(t, err)
	testHashFunc(t, h.HashBytes, 32)
}

func testHashFunc(t *testing.T, h HashFunc, size int) {
	require := require.New(t)
	m := make(map[string]struct{})
	for i := 0; i < 1024; i++ {
		sz := int(mrand.Int31n(1024*8) + 128)
		b := make([]byte, sz)
		n, err := rand.Read(b)
		require.NoError(err)
		require.Equal(sz, n)

		hash := h(b)
		_, ok := m[string(hash)]
		require.False(ok)
		require.Len(hash, size)
		m[string(hash)] = struct{}{}

		for i := 0; i < 128; i++ {
			require.Equal(hash, h(b))
		}
	}
}

func BenchmarkHashBytes(b *testing.B) {
	b.Run("512-bytes", func(b *testing.B) {
		benchmarkHashFunc(b, HashBytes, 512, 32)
	})
	b.Run("4096-bytes", func(b *testing.B) {
		benchmarkHashFunc(b, HashBytes, 4096, 32)
	})
	b.Run("131072-bytes", func(b *testing.B) {
		benchmarkHashFunc(b, HashBytes, 131072, 32)
	})
}

func BenchmarkDefaultHasher(b *testing.B) {
	b.Run("512-bytes", func(b *testing.B) {
		hasher, err := NewHasher()
		if err != nil {
			b.Error(err)
		}
		benchmarkHashFunc(b, hasher.HashBytes, 512, 32)
	})
	b.Run("4096-bytes", func(b *testing.B) {
		hasher, err := NewHasher()
		if err != nil {
			b.Error(err)
		}
		benchmarkHashFunc(b, hasher.HashBytes, 4096, 32)
	})
	b.Run("131072-bytes", func(b *testing.B) {
		hasher, err := NewHasher()
		if err != nil {
			b.Error(err)
		}
		benchmarkHashFunc(b, hasher.HashBytes, 131072, 32)
	})
}

func benchmarkHashFunc(b *testing.B, h HashFunc, isz, osz int) {
	bytes := make([]byte, isz)
	rand.Read(bytes)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		hash := h(bytes)
		if len(hash) != osz {
			b.Errorf("output hash has size %d, while expecting %d", len(hash), osz)
		}
	}
}
