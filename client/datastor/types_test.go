package datastor

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestObjectStatusString(t *testing.T) {
	require := require.New(t)

	// valid enum values
	require.Equal("missing", ObjectStatusMissing.String())
	require.Equal("ok", ObjectStatusOK.String())
	require.Equal("corrupted", ObjectStatusCorrupted.String())

	// invalid enum value
	require.Empty(ObjectStatus(math.MaxUint8).String())
}
