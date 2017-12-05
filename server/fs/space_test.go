package fs

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFreeSpace(t *testing.T) {
	_, err := FreeSpace("")
	require.Error(t, err)

	size, err := FreeSpace(".")
	require.NoError(t, err)
	require.NotZero(t, size)
}
