package crypto

import (
	"crypto/rand"
	"math"
	mrand "math/rand"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewHasher(t *testing.T) {
	require := require.New(t)

	testCases := []struct {
		Type     HashType
		Expected interface{}
	}{
		{HashTypeSHA256, (*SHA256Hasher)(nil)},
		{HashTypeSHA512, (*SHA512Hasher)(nil)},
		{HashTypeBlake2b256, (*Blake2b256Hasher)(nil)},
		{HashTypeBlake2b512, (*Blake2b512Hasher)(nil)},
		{myCustomHashType, myCustomHasher{}},
		{math.MaxUint8, nil},
	}
	for _, tc := range testCases {
		h, err := NewHasher(tc.Type)
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
		HashTypeBlake2b256,
		HashTypeBlake2b512,
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
		{HashTypeBlake2b256, "blake2b_256"},
		{HashTypeBlake2b512, "blake2b_512"},
		{myCustomHashType, myCustomHashTypeStr},
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
		{"sha_256", HashTypeSHA256, false},
		{"SHA_256", HashTypeSHA256, false},
		{"sha_512", HashTypeSHA512, false},
		{"SHA_512", HashTypeSHA512, false},
		{"blake2b_256", HashTypeBlake2b256, false},
		{"BLAKE2B_256", HashTypeBlake2b256, false},
		{"blake2b_512", HashTypeBlake2b512, false},
		{"BLAKE2B_512", HashTypeBlake2b512, false},
		{myCustomHashTypeStr, myCustomHashType, false},
		{strings.ToUpper(myCustomHashTypeStr), myCustomHashType, false},
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

func TestSum256(t *testing.T) {
	testSumFunc(t, Sum256, 32)
}

func TestSum512(t *testing.T) {
	testSumFunc(t, Sum512, 64)
}

func TestDefaultHasher256(t *testing.T) {
	h, err := NewDefaultHasher256()
	require.NoError(t, err)
	testSumFunc(t, h.HashBytes, 32)
}

func TestDefaultHasher512(t *testing.T) {
	h, err := NewDefaultHasher512()
	require.NoError(t, err)
	testSumFunc(t, h.HashBytes, 64)
}

func testSumFunc(t *testing.T, h HashFunc, size int) {
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

func BenchmarkSum256(b *testing.B) {
	b.Run("512-bytes", func(b *testing.B) {
		benchmarkHashFunc(b, Sum256, 512, 32)
	})
	b.Run("4096-bytes", func(b *testing.B) {
		benchmarkHashFunc(b, Sum256, 4096, 32)
	})
	b.Run("131072-bytes", func(b *testing.B) {
		benchmarkHashFunc(b, Sum256, 131072, 32)
	})
}

func BenchmarkSum512(b *testing.B) {
	b.Run("512-bytes", func(b *testing.B) {
		benchmarkHashFunc(b, Sum512, 512, 64)
	})
	b.Run("4096-bytes", func(b *testing.B) {
		benchmarkHashFunc(b, Sum512, 4096, 64)
	})
	b.Run("131072-bytes", func(b *testing.B) {
		benchmarkHashFunc(b, Sum512, 131072, 64)
	})
}

func BenchmarkDefaultHasher256(b *testing.B) {
	b.Run("512-bytes", func(b *testing.B) {
		hasher, err := NewDefaultHasher256()
		if err != nil {
			b.Error(err)
		}
		benchmarkHashFunc(b, hasher.HashBytes, 512, 32)
	})
	b.Run("4096-bytes", func(b *testing.B) {
		hasher, err := NewDefaultHasher256()
		if err != nil {
			b.Error(err)
		}
		benchmarkHashFunc(b, hasher.HashBytes, 4096, 32)
	})
	b.Run("131072-bytes", func(b *testing.B) {
		hasher, err := NewDefaultHasher256()
		if err != nil {
			b.Error(err)
		}
		benchmarkHashFunc(b, hasher.HashBytes, 131072, 32)
	})
}

func BenchmarkDefaultHasher512(b *testing.B) {
	b.Run("512-bytes", func(b *testing.B) {
		hasher, err := NewDefaultHasher512()
		if err != nil {
			b.Error(err)
		}
		benchmarkHashFunc(b, hasher.HashBytes, 512, 64)
	})
	b.Run("4096-bytes", func(b *testing.B) {
		hasher, err := NewDefaultHasher512()
		if err != nil {
			b.Error(err)
		}
		benchmarkHashFunc(b, hasher.HashBytes, 4096, 64)
	})
	b.Run("131072-bytes", func(b *testing.B) {
		hasher, err := NewDefaultHasher512()
		if err != nil {
			b.Error(err)
		}
		benchmarkHashFunc(b, hasher.HashBytes, 131072, 64)
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

// some tests to ensure a user can register its own func,
// without overwriting the existing hash algorithms

func TestMyCustomHasher(t *testing.T) {
	require := require.New(t)

	hasher, err := NewHasher(myCustomHashType)
	require.NoError(err)
	require.NotNil(hasher)
	require.IsType(myCustomHasher{}, hasher)

	require.Equal(
		[]byte("01234567890123456789012345678901"),
		hasher.HashBytes(nil))
	require.Equal(
		[]byte("01234567890123456789012345678901"),
		hasher.HashBytes([]byte("foo")))
	require.Equal(
		[]byte("01234567890123456789012345678901"),
		hasher.HashBytes([]byte("01234567890123456789012345678901")))
}

func TestRegisterHashExplicitPanics(t *testing.T) {
	require := require.New(t)

	require.Panics(func() {
		RegisterHash(myCustomHashType, myCustomHashTypeStr, nil)
	}, "no constructor given")

	require.Panics(func() {
		RegisterHash(myCustomHashTypeNumberTwo, "", newMyCustomHasher)
	}, "no string version given for non-registered hash")
}

func TestRegisterHashIgnoreStringExistingHash(t *testing.T) {
	require := require.New(t)

	require.Equal(myCustomHashTypeStr, myCustomHashType.String())
	RegisterHash(myCustomHashType, "foo", newMyCustomHasher)
	require.Equal(myCustomHashTypeStr, myCustomHashType.String())
}

const (
	myCustomHashType = iota + MaxStandardHashType + 1
	myCustomHashTypeNumberTwo

	myCustomHashTypeStr = "bad_256"
)

type myCustomHasher struct{}

func (ch myCustomHasher) HashBytes([]byte) []byte {
	return []byte("01234567890123456789012345678901")
}

func newMyCustomHasher() (Hasher, error) {
	return myCustomHasher{}, nil
}

func init() {
	RegisterHash(myCustomHashType, myCustomHashTypeStr, newMyCustomHasher)
}
