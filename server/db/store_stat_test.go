package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStoreStatEncodeDecode(t *testing.T) {
	stat := StoreStat{
		SizeAvailable: 100,
		SizeUsed:      45,
	}

	b, err := stat.Encode()
	require.NoError(t, err, "fail to encode StoreStat")

	stat2 := StoreStat{}
	err = stat2.Decode(b)
	require.NoError(t, err, "fail to decode StoreStat")

	assert.Equal(t, stat, stat2, "two object should be the same")
}
