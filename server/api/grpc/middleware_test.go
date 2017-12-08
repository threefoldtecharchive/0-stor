package grpc

import (
	"context"
	"testing"

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
