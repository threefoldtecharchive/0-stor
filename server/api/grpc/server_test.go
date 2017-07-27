package grpc

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewServer(t *testing.T) {
	_, err := New("localhost:8080")
	require.NoError(t, err)
}
