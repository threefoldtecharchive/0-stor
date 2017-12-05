package grpc

import (
	"context"
	"errors"
	"strings"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"

	"github.com/stretchr/testify/require"
)

func TestExtractStringFromContext(t *testing.T) {
	require := require.New(t)

	ctx := context.Background()
	require.NotNil(ctx)
	label, err := extractStringFromContext(ctx, "bar")
	require.Error(err)
	require.Empty(label)

	md := metadata.MD{}
	ctx = metadata.NewIncomingContext(ctx, md)
	require.NotNil(ctx)
	label, err = extractStringFromContext(ctx, "bar")
	require.Error(err)
	require.Empty(label)

	md["bar"] = []string{"foo"}
	label, err = extractStringFromContext(ctx, "bar")
	require.NoError(err)
	require.Equal("foo", label)
}

func TestUnauthenticatedError(t *testing.T) {
	require := require.New(t)
	err := errors.New("Hello Error")
	require.NotNil(err)
	statusErr := unauthenticatedError(err)
	errStr := statusErr.Error()
	require.True(strings.Contains(errStr, "Hello Error"))
	require.True(strings.Contains(errStr, codes.Unauthenticated.String()))
}
