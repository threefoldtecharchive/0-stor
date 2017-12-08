package grpc

import (
	"errors"
	"fmt"

	"golang.org/x/net/context"
	"google.golang.org/grpc/metadata"
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
