package grpc

import (
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"

	"github.com/zero-os/0-stor/client/itsyouonline"
	pb "github.com/zero-os/0-stor/daemon/api/grpc/schema"
)

func TestCreateJWT(t *testing.T) {
	nsSrv := newNamespaceSrv(&namespaceClientStub{})

	rep, err := nsSrv.CreateJWT(context.Background(), &pb.CreateJWTRequest{
		Namespace:  "ns",
		Permission: &pb.Permission{},
	})

	require.NoError(t, err)
	require.NotEmpty(t, rep.Token)
}

func TestCreateJWTErrors(t *testing.T) {
	nsSrv := newNamespaceSrv(&namespaceClientStub{})

	// empty namespace
	_, err := nsSrv.CreateJWT(context.Background(), &pb.CreateJWTRequest{
		Permission: &pb.Permission{},
	})

	require.Equal(t, errEmptyNamespace, err)

	// nil permission
	_, err = nsSrv.CreateJWT(context.Background(), &pb.CreateJWTRequest{
		Namespace: "ns",
	})

	require.Equal(t, errNilPermission, err)

}

func TestCreateNamespace(t *testing.T) {
	nsSrv := newNamespaceSrv(&namespaceClientStub{})

	_, err := nsSrv.CreateNamespace(context.Background(), &pb.NamespaceRequest{
		Namespace: "ns",
	})

	require.NoError(t, err)
}

func TestCreateNamespaceError(t *testing.T) {
	nsSrv := newNamespaceSrv(&namespaceClientStub{})

	// empty namespace
	_, err := nsSrv.CreateNamespace(context.Background(), &pb.NamespaceRequest{})

	require.Equal(t, errEmptyNamespace, err)
}

func TestDeleteNamespace(t *testing.T) {
	nsSrv := newNamespaceSrv(&namespaceClientStub{})

	_, err := nsSrv.DeleteNamespace(context.Background(), &pb.NamespaceRequest{
		Namespace: "ns",
	})

	require.NoError(t, err)
}

func TestDeleteNamespaceError(t *testing.T) {
	nsSrv := newNamespaceSrv(&namespaceClientStub{})

	_, err := nsSrv.DeleteNamespace(context.Background(), &pb.NamespaceRequest{})

	require.Equal(t, errEmptyNamespace, err)
}

func TestGivePermission(t *testing.T) {
	nsSrv := newNamespaceSrv(&namespaceClientStub{})

	_, err := nsSrv.GivePermission(context.Background(), &pb.EditPermissionRequest{
		Namespace:  "ns",
		UserID:     "user",
		Permission: &pb.Permission{},
	})

	require.NoError(t, err)
}

func TestGivePermissionError(t *testing.T) {
	nsSrv := newNamespaceSrv(&namespaceClientStub{})

	// empty namespace
	_, err := nsSrv.GivePermission(context.Background(), &pb.EditPermissionRequest{
		UserID:     "user",
		Permission: &pb.Permission{},
	})

	require.Equal(t, errEmptyNamespace, err)

	// empty namespace
	_, err = nsSrv.GivePermission(context.Background(), &pb.EditPermissionRequest{
		Namespace:  "ns",
		Permission: &pb.Permission{},
	})

	require.Equal(t, errEmptyUserID, err)

	// nil permission
	_, err = nsSrv.GivePermission(context.Background(), &pb.EditPermissionRequest{
		Namespace: "ns",
		UserID:    "user",
	})

	require.Equal(t, errNilPermission, err)

}

func TestRemovePermission(t *testing.T) {
	nsSrv := newNamespaceSrv(&namespaceClientStub{})

	_, err := nsSrv.RemovePermission(context.Background(), &pb.EditPermissionRequest{
		Namespace:  "ns",
		UserID:     "user",
		Permission: &pb.Permission{},
	})

	require.NoError(t, err)
}

func TestRemovePermissionError(t *testing.T) {
	nsSrv := newNamespaceSrv(&namespaceClientStub{})

	// empty namespace
	_, err := nsSrv.RemovePermission(context.Background(), &pb.EditPermissionRequest{
		UserID:     "user",
		Permission: &pb.Permission{},
	})

	require.Equal(t, errEmptyNamespace, err)
}

func TestGetPermission(t *testing.T) {
	nsSrv := newNamespaceSrv(&namespaceClientStub{})

	_, err := nsSrv.GetPermission(context.Background(), &pb.GetPermissionRequest{
		Namespace: "ns",
		UserID:    "user",
	})

	require.NoError(t, err)
}

func TestGetPermissionError(t *testing.T) {
	nsSrv := newNamespaceSrv(&namespaceClientStub{})

	// empty namespace
	_, err := nsSrv.GetPermission(context.Background(), &pb.GetPermissionRequest{
		UserID: "user",
	})

	require.Equal(t, errEmptyNamespace, err)

	// empty userID
	_, err = nsSrv.GetPermission(context.Background(), &pb.GetPermissionRequest{
		Namespace: "ns",
	})

	require.Equal(t, errEmptyUserID, err)

}

type namespaceClientStub struct {
}

func (ncs *namespaceClientStub) CreateJWT(namespace string, perm itsyouonline.Permission) (string, error) {
	return "some_jwt_token", nil
}
func (ncs *namespaceClientStub) CreateNamespace(namespace string) error {
	return nil
}
func (ncs *namespaceClientStub) DeleteNamespace(namespace string) error {
	return nil
}
func (ncs *namespaceClientStub) GetPermission(namespace, userID string) (itsyouonline.Permission, error) {
	return itsyouonline.Permission{}, nil
}

func (ncs *namespaceClientStub) GivePermission(namespace, userID string, perm itsyouonline.Permission) error {
	return nil
}
func (ncs *namespaceClientStub) RemovePermission(namespace, userID string, perm itsyouonline.Permission) error {
	return nil
}
