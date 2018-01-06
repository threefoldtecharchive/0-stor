/*
 * Copyright (C) 2017-2018 GIG Technology NV and Contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package grpc

import (
	"fmt"

	"github.com/zero-os/0-stor/client/itsyouonline"
	"github.com/zero-os/0-stor/daemon/api/grpc/rpctypes"
	pb "github.com/zero-os/0-stor/daemon/api/grpc/schema"

	"golang.org/x/net/context"
)

func newNamespaceService(client namespaceClient) *namespaceService {
	return &namespaceService{client: client}
}

// namespaceService is used to manage (IYO) namespaces and
// the permissions for them for specific users.
type namespaceService struct {
	client namespaceClient
}

// CreateNamespace implements NamespaceServiceServer.CreateNamespace
func (service *namespaceService) CreateNamespace(ctx context.Context, req *pb.CreateNamespaceRequest) (*pb.CreateNamespaceResponse, error) {
	namespace := req.GetNamespace()
	if len(namespace) == 0 {
		return nil, rpctypes.ErrGRPCNilNamespace
	}

	err := service.client.CreateNamespace(namespace)
	if err != nil {
		return nil, err
	}
	return &pb.CreateNamespaceResponse{}, nil
}

// DeleteNamespace implements NamespaceServiceServer.DeleteNamespace
func (service *namespaceService) DeleteNamespace(ctx context.Context, req *pb.DeleteNamespaceRequest) (*pb.DeleteNamespaceResponse, error) {
	namespace := req.GetNamespace()
	if len(namespace) == 0 {
		return nil, rpctypes.ErrGRPCNilNamespace
	}

	err := service.client.DeleteNamespace(namespace)
	if err != nil {
		return nil, err
	}
	return &pb.DeleteNamespaceResponse{}, nil
}

// SetPermission implements NamespaceServiceServer.SetPermission
func (service *namespaceService) SetPermission(ctx context.Context, req *pb.SetPermissionRequest) (*pb.SetPermissionResponse, error) {
	namespace := req.GetNamespace()
	if len(namespace) == 0 {
		return nil, rpctypes.ErrGRPCNilNamespace
	}
	userID := req.GetUserID()
	if len(userID) == 0 {
		return nil, rpctypes.ErrGRPCNilUserID
	}
	perm := req.GetPermission()
	if perm == nil {
		return nil, rpctypes.ErrGRPCNilPermissions
	}

	err := service.client.SetPermission(namespace, userID, itsyouonline.Permission{
		Read:   perm.Read,
		Write:  perm.Write,
		Delete: perm.Delete,
		Admin:  perm.Admin,
	})
	if err != nil {
		return nil, err
	}
	return &pb.SetPermissionResponse{}, nil
}

// GetPermission implements NamespaceServiceServer.GetPermission
func (service *namespaceService) GetPermission(ctx context.Context, req *pb.GetPermissionRequest) (*pb.GetPermissionResponse, error) {
	namespace := req.GetNamespace()
	if len(namespace) == 0 {
		return nil, rpctypes.ErrGRPCNilNamespace
	}
	userID := req.GetUserID()
	if len(userID) == 0 {
		return nil, rpctypes.ErrGRPCNilUserID
	}

	perm, err := service.client.GetPermission(namespace, userID)
	if err != nil {
		return nil, err
	}
	return &pb.GetPermissionResponse{
		Permission: &pb.Permission{
			Read:   perm.Read,
			Write:  perm.Write,
			Delete: perm.Delete,
			Admin:  perm.Admin,
		},
	}, nil
}

// namespaceClient is used by the namespaceService,
// to run the actual business logic of the service.
type namespaceClient interface {
	CreateNamespace(namespace string) error
	DeleteNamespace(namespace string) error
	SetPermission(namespace, userID string, perm itsyouonline.Permission) error
	GetPermission(namespace, userID string) (itsyouonline.Permission, error)
}

// namespaceClientFromIYOClient creates a namespace client,
// given an itsyouonline.Client, which is allowed to be nil.
func namespaceClientFromIYOClient(client *itsyouonline.Client) namespaceClient {
	if client == nil {
		return nilIYOClient{}
	}
	return &iyoClient{client}
}

// iyoClient extends the itsyouonline.Client,
// providing a SetPermission function, which combines the Give and Remove Permission functions.
type iyoClient struct {
	*itsyouonline.Client
}

func (iyo *iyoClient) SetPermission(namespace, userID string, perm itsyouonline.Permission) error {
	currentPermissions, err := iyo.Client.GetPermission(namespace, userID)
	if err != nil {
		return fmt.Errorf("failed to retrieve permission(s) for %s:%s: %v",
			userID, namespace, err)
	}

	// remove permission if needed
	permsToRemove := computePermissionsToRemove(currentPermissions, perm)
	if err := iyo.Client.RemovePermission(namespace, userID, permsToRemove); err != nil {
		return fmt.Errorf("failed to remove permission(s) for %s:%s: %v",
			userID, namespace, err)
	}

	// add permission if needed
	permsToGive := computePermissionsToGive(currentPermissions, perm)
	if err := iyo.Client.GivePermission(namespace, userID, permsToGive); err != nil {
		return fmt.Errorf("fail to give permission(s) for %s:%s: %v",
			userID, namespace, err)
	}

	return nil
}

// nilIYOClient can be used to implement a pure-non-supported namespace service,
// which is useful/required in case the IYO client is nil.
type nilIYOClient struct{}

func (iyo nilIYOClient) CreateNamespace(namespace string) error { return rpctypes.ErrGRPCNotSupported }
func (iyo nilIYOClient) DeleteNamespace(namespace string) error { return rpctypes.ErrGRPCNotSupported }
func (iyo nilIYOClient) SetPermission(namespace, userID string, perm itsyouonline.Permission) error {
	return rpctypes.ErrGRPCNotSupported
}
func (iyo nilIYOClient) GetPermission(namespace, userID string) (itsyouonline.Permission, error) {
	return itsyouonline.Permission{}, rpctypes.ErrGRPCNotSupported
}

func computePermissionsToRemove(current, toRemove itsyouonline.Permission) itsyouonline.Permission {
	return itsyouonline.Permission{
		Read:   current.Read && !toRemove.Read,
		Write:  current.Write && !toRemove.Write,
		Delete: current.Delete && !toRemove.Delete,
		Admin:  current.Admin && !toRemove.Admin,
	}
}

func computePermissionsToGive(current, toGive itsyouonline.Permission) itsyouonline.Permission {
	return itsyouonline.Permission{
		Read:   !current.Read && toGive.Read,
		Write:  !current.Write && toGive.Write,
		Delete: !current.Delete && toGive.Delete,
		Admin:  !current.Admin && toGive.Admin,
	}
}

var (
	_ pb.NamespaceServiceServer = (*namespaceService)(nil)
	_ namespaceClient           = (*iyoClient)(nil)
	_ namespaceClient           = nilIYOClient{}
)
