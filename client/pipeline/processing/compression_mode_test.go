package processing

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompressionModeMarshalUnmarshal(t *testing.T) {
	require := require.New(t)

	types := []CompressionMode{
		CompressionModeDefault,
		CompressionModeBestSpeed,
		CompressionModeBestCompression,
	}
	for _, t := range types {
		b, err := t.MarshalText()
		require.NoError(err)
		require.NotNil(b)

		var o CompressionMode
		err = o.UnmarshalText(b)
		require.NoError(err)
		require.Equal(t, o)
	}
}

func TestCompressionModeMarshal(t *testing.T) {
	require := require.New(t)

	testCases := []struct {
		Type     CompressionMode
		Expected string
	}{
		{CompressionModeDefault, "default"},
		{CompressionModeBestSpeed, "best_speed"},
		{CompressionModeBestCompression, "best_compression"},
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

func TestCompressionModeUnmarshal(t *testing.T) {
	require := require.New(t)

	testCases := []struct {
		String   string
		Expected CompressionMode
		Err      bool
	}{
		{"default", CompressionModeDefault, false},
		{"Default", CompressionModeDefault, false},
		{"DEFAULT", CompressionModeDefault, false},
		{"best_speed", CompressionModeBestSpeed, false},
		{"Best_Speed", CompressionModeBestSpeed, false},
		{"BEST_SPEED", CompressionModeBestSpeed, false},
		{"best_compression", CompressionModeBestCompression, false},
		{"Best_Compression", CompressionModeBestCompression, false},
		{"BEST_COMPRESSION", CompressionModeBestCompression, false},
		{"", CompressionModeDisabled, false},
		{"foo", math.MaxUint8, true},
	}
	for _, tc := range testCases {
		var o CompressionMode
		err := o.UnmarshalText([]byte(tc.String))
		if tc.Err {
			require.Error(err)
			require.Equal(CompressionModeDisabled, o)
		} else {
			require.NoError(err)
			require.Equal(tc.Expected, o)
		}
	}
}
