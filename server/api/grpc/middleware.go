package grpc

import (
	"errors"
	"fmt"

	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	objectPrefix          = "/ObjectManager/"
	objectPrefixLength    = len(objectPrefix)
	namespacePrefix       = "/NamespaceManager/"
	namespacePrefixLength = len(namespacePrefix)
)

// extractStringFromContext extracts a string from grpc context's
func extractStringFromContext(ctx context.Context, key string) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", errors.New("no metadata found in grpc context")
	}

	slice, ok := md[key]
	if !ok || len(slice) < 1 {
		return "", fmt.Errorf("no %s found metadata from grpc context", key)
	}

	return slice[0], nil
}

// unauthenticatedError returns the provided error with a grpc Unauthenticated code
func unauthenticatedError(err error) error {
	return status.Error(codes.Unauthenticated, err.Error())
}
