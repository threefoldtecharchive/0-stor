// shared middleware logic

package server

import (
	"errors"

	"github.com/zero-os/0-stor/server/grpc"

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

// extractLabelFromContext extracts label from grpc context's
func extractLabelFromContext(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", errors.New("no metadata found in grpc context")
	}

	label, ok := md[grpc.MetaLabelKey]
	if !ok || len(label) < 1 {
		return "", errors.New("no label found in grpc context")
	}

	return label[0], nil
}

// unauthCodeError returns the provided error with a grpc Unauthenticated code
func unauthCodeError(err error) error {
	return status.Error(codes.Unauthenticated, err.Error())
}
