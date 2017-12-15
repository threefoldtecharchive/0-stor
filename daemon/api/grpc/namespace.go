package grpc

import (
	"errors"

	"github.com/zero-os/0-stor/client/itsyouonline"
	pb "github.com/zero-os/0-stor/daemon/api/grpc/schema"
	"golang.org/x/net/context"
)

var (
	errNilPermission  = errors.New("nil permission")
	errEmptyNamespace = errors.New("empty namespace")
	errEmptyUserID    = errors.New("empty user ID")
)

type namespaceClient interface {
	CreateJWT(namespace string, perm itsyouonline.Permission) (string, error)
	CreateNamespace(namespace string) error
	DeleteNamespace(namespace string) error
	GetPermission(namespace, userID string) (itsyouonline.Permission, error)
	GivePermission(namespace, userID string, perm itsyouonline.Permission) error
	RemovePermission(namespace, userID string, perm itsyouonline.Permission) error
}
type namespaceSrv struct {
	client namespaceClient
}

func newNamespaceSrv(client namespaceClient) *namespaceSrv {
	return &namespaceSrv{
		client: client,
	}
}

func (ns *namespaceSrv) CreateJWT(ctx context.Context, req *pb.CreateJWTRequest) (*pb.CreateJWTReply, error) {
	if req.Namespace == "" {
		return nil, errEmptyNamespace
	}

	if req.Permission == nil {
		return nil, errNilPermission
	}

	token, err := ns.client.CreateJWT(req.Namespace, pbPermToStorPerm(req.Permission))
	if err != nil {
		return nil, err
	}

	return &pb.CreateJWTReply{
		Token: token,
	}, nil
}

func (ns *namespaceSrv) CreateNamespace(ctx context.Context, req *pb.NamespaceRequest) (*pb.NamespaceReply, error) {
	err := checkNamespaceReq(req)
	if err != nil {
		return nil, err
	}

	err = ns.client.CreateNamespace(req.Namespace)
	if err != nil {
		return nil, err
	}

	return &pb.NamespaceReply{}, nil
}

func (ns *namespaceSrv) DeleteNamespace(ctx context.Context, req *pb.NamespaceRequest) (*pb.NamespaceReply, error) {
	err := checkNamespaceReq(req)
	if err != nil {
		return nil, err
	}

	err = ns.client.DeleteNamespace(req.Namespace)
	if err != nil {
		return nil, err
	}

	return &pb.NamespaceReply{}, nil
}

func (ns *namespaceSrv) GivePermission(ctx context.Context, req *pb.EditPermissionRequest) (*pb.EditPermissionReply, error) {
	err := checkEditPermissionRequest(req)
	if err != nil {
		return nil, err
	}

	err = ns.client.GivePermission(req.Namespace, req.UserID, pbPermToStorPerm(req.Permission))
	if err != nil {
		return nil, err
	}
	return &pb.EditPermissionReply{}, nil
}

func (ns *namespaceSrv) RemovePermission(ctx context.Context, req *pb.EditPermissionRequest) (*pb.EditPermissionReply, error) {
	err := checkEditPermissionRequest(req)
	if err != nil {
		return nil, err
	}

	err = ns.client.RemovePermission(req.Namespace, req.UserID, pbPermToStorPerm(req.Permission))
	if err != nil {
		return nil, err
	}

	return &pb.EditPermissionReply{}, nil
}

func (ns *namespaceSrv) GetPermission(ctx context.Context, req *pb.GetPermissionRequest) (*pb.GetPermissionReply, error) {
	err := checkGetPermissionRequest(req)
	if err != nil {
		return nil, err
	}

	perm, err := ns.client.GetPermission(req.Namespace, req.UserID)
	if err != nil {
		return nil, err
	}

	return &pb.GetPermissionReply{
		Permission: storPermToPbPerm(perm),
	}, nil
}

// convert protobuf permission object to 0-stor permission object
func pbPermToStorPerm(pbPerm *pb.Permission) itsyouonline.Permission {
	if pbPerm == nil {
		return itsyouonline.Permission{}
	}

	return itsyouonline.Permission{
		Write:  pbPerm.Write,
		Read:   pbPerm.Read,
		Delete: pbPerm.Delete,
		Admin:  pbPerm.Admin,
	}
}

// convert 0-stor permission object to protobuf permission object
func storPermToPbPerm(perm itsyouonline.Permission) *pb.Permission {
	return &pb.Permission{
		Write:  perm.Write,
		Read:   perm.Read,
		Delete: perm.Delete,
		Admin:  perm.Admin,
	}
}

// check protobuf EditPermissionRequest validity
func checkEditPermissionRequest(req *pb.EditPermissionRequest) error {
	if req.Namespace == "" {
		return errEmptyNamespace
	}
	if req.UserID == "" {
		return errEmptyUserID
	}
	if req.Permission == nil {
		return errNilPermission
	}
	return nil
}

// check protobuf GetPermissionRequest validity
func checkGetPermissionRequest(req *pb.GetPermissionRequest) error {
	if req.Namespace == "" {
		return errEmptyNamespace
	}
	if req.UserID == "" {
		return errEmptyUserID
	}
	return nil
}

// check protobuf NamespaceRequest validity
func checkNamespaceReq(req *pb.NamespaceRequest) error {
	if req.Namespace == "" {
		return errEmptyNamespace
	}
	return nil
}
