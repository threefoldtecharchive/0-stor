package processing

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompressionTypeMarshalUnmarshal(t *testing.T) {
	require := require.New(t)

	types := []CompressionType{
		CompressionTypeSnappy,
		CompressionTypeLZ4,
		CompressionTypeGZip,
	}
	for _, t := range types {
		b, err := t.MarshalText()
		require.NoError(err)
		require.NotNil(b)

		var o CompressionType
		err = o.UnmarshalText(b)
		require.NoError(err)
		require.Equal(t, o)
	}
}

func TestCompressionTypeMarshal(t *testing.T) {
	require := require.New(t)

	testCases := []struct {
		Type     CompressionType
		Expected string
	}{
		{CompressionTypeSnappy, "snappy"},
		{CompressionTypeLZ4, "lz4"},
		{CompressionTypeGZip, "gzip"},
		{math.MaxUint8, "255"},
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

func TestCompressionTypeUnmarshal(t *testing.T) {
	require := require.New(t)

	testCases := []struct {
		String   string
		Expected CompressionType
		Err      bool
	}{
		{"snappy", CompressionTypeSnappy, false},
		{"Snappy", CompressionTypeSnappy, false},
		{"SNAPPY", CompressionTypeSnappy, false},
		{"lz4", CompressionTypeLZ4, false},
		{"LZ4", CompressionTypeLZ4, false},
		{"gzip", CompressionTypeGZip, false},
		{"GZip", CompressionTypeGZip, false},
		{"GZIP", CompressionTypeGZip, false},
		{"", DefaultCompressionType, false},
		{"some invalid type", math.MaxUint8, true},
	}
	for _, tc := range testCases {
		var o CompressionType
		err := o.UnmarshalText([]byte(tc.String))
		if tc.Err {
			require.Error(err)
			require.Equal(DefaultCompressionType, o)
		} else {
			require.NoError(err)
			require.Equal(tc.Expected, o)
		}
	}
}
