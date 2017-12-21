package server

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultObjectStatus(t *testing.T) {
	var status ObjectStatus
	require.Equal(t, ObjectStatusMissing, status)
}

func TestObjectStatusString(t *testing.T) {
	require := require.New(t)

	require.Equal("ok", ObjectStatusOK.String())
	require.Equal("missing", ObjectStatusMissing.String())
	require.Equal("corrupted", ObjectStatusCorrupted.String())
}
