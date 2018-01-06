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
	"testing"

	"github.com/zero-os/0-stor/client/itsyouonline"
	"github.com/zero-os/0-stor/daemon/api/grpc/rpctypes"
	pb "github.com/zero-os/0-stor/daemon/api/grpc/schema"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

func TestCreateNamespace(t *testing.T) {
	nsSrv := newNamespaceService(&namespaceClientStub{})

	_, err := nsSrv.CreateNamespace(context.Background(), &pb.CreateNamespaceRequest{
		Namespace: "ns",
	})
	require.NoError(t, err)
}

func TestCreateNamespaceError(t *testing.T) {
	nsSrv := newNamespaceService(&namespaceClientStub{})

	// empty namespace
	_, err := nsSrv.CreateNamespace(context.Background(), &pb.CreateNamespaceRequest{})
	require.Equal(t, rpctypes.ErrGRPCNilNamespace, err)

	// not supported (no IYO client defined)
	nsSrv.client = nilIYOClient{}
	_, err = nsSrv.CreateNamespace(context.Background(), &pb.CreateNamespaceRequest{
		Namespace: "ns",
	})
	require.Equal(t, rpctypes.ErrGRPCNotSupported, err)
}

func TestDeleteNamespace(t *testing.T) {
	nsSrv := newNamespaceService(&namespaceClientStub{})

	_, err := nsSrv.DeleteNamespace(context.Background(), &pb.DeleteNamespaceRequest{
		Namespace: "ns",
	})
	require.NoError(t, err)
}

func TestDeleteNamespaceError(t *testing.T) {
	nsSrv := newNamespaceService(&namespaceClientStub{})

	// empty namespace
	_, err := nsSrv.DeleteNamespace(context.Background(), &pb.DeleteNamespaceRequest{})
	require.Equal(t, rpctypes.ErrGRPCNilNamespace, err)

	// not supported (no IYO client defined)
	nsSrv.client = nilIYOClient{}
	_, err = nsSrv.DeleteNamespace(context.Background(), &pb.DeleteNamespaceRequest{
		Namespace: "ns",
	})
	require.Equal(t, rpctypes.ErrGRPCNotSupported, err)
}

func TestSetPermission(t *testing.T) {
	nsSrv := newNamespaceService(&namespaceClientStub{})

	_, err := nsSrv.SetPermission(context.Background(), &pb.SetPermissionRequest{
		Namespace:  "ns",
		UserID:     "user",
		Permission: &pb.Permission{},
	})
	require.NoError(t, err)
}

func TestSetPermissionError(t *testing.T) {
	nsSrv := newNamespaceService(&namespaceClientStub{})

	// empty namespace
	_, err := nsSrv.SetPermission(context.Background(), &pb.SetPermissionRequest{
		UserID:     "user",
		Permission: &pb.Permission{},
	})
	require.Equal(t, rpctypes.ErrGRPCNilNamespace, err)

	// empty userID
	_, err = nsSrv.SetPermission(context.Background(), &pb.SetPermissionRequest{
		Namespace:  "ns",
		Permission: &pb.Permission{},
	})
	require.Equal(t, rpctypes.ErrGRPCNilUserID, err)

	// nil permission
	_, err = nsSrv.SetPermission(context.Background(), &pb.SetPermissionRequest{
		Namespace: "ns",
		UserID:    "user",
	})
	require.Equal(t, rpctypes.ErrGRPCNilPermissions, err)

	// not supported (no IYO client defined)
	nsSrv.client = nilIYOClient{}
	_, err = nsSrv.SetPermission(context.Background(), &pb.SetPermissionRequest{
		Namespace:  "ns",
		UserID:     "user",
		Permission: &pb.Permission{},
	})
	require.Equal(t, rpctypes.ErrGRPCNotSupported, err)
}

func TestGetPermission(t *testing.T) {
	nsSrv := newNamespaceService(&namespaceClientStub{})

	_, err := nsSrv.GetPermission(context.Background(), &pb.GetPermissionRequest{
		Namespace: "ns",
		UserID:    "user",
	})
	require.NoError(t, err)
}

func TestGetPermissionError(t *testing.T) {
	nsSrv := newNamespaceService(&namespaceClientStub{})

	// empty namespace
	_, err := nsSrv.GetPermission(context.Background(), &pb.GetPermissionRequest{
		UserID: "user",
	})
	require.Equal(t, rpctypes.ErrGRPCNilNamespace, err)

	// empty userID
	_, err = nsSrv.GetPermission(context.Background(), &pb.GetPermissionRequest{
		Namespace: "ns",
	})
	require.Equal(t, rpctypes.ErrGRPCNilUserID, err)

	// not supported (no IYO client defined)
	nsSrv.client = nilIYOClient{}
	_, err = nsSrv.GetPermission(context.Background(), &pb.GetPermissionRequest{
		Namespace: "ns",
		UserID:    "user",
	})
	require.Equal(t, rpctypes.ErrGRPCNotSupported, err)
}

type namespaceClientStub struct{}

func (ncs *namespaceClientStub) CreateNamespace(namespace string) error {
	return nil
}
func (ncs *namespaceClientStub) DeleteNamespace(namespace string) error {
	return nil
}
func (ncs *namespaceClientStub) SetPermission(namespace, userID string, perm itsyouonline.Permission) error {
	return nil
}
func (ncs *namespaceClientStub) GetPermission(namespace, userID string) (itsyouonline.Permission, error) {
	return itsyouonline.Permission{}, nil
}

var (
	_ namespaceClient = (*namespaceClientStub)(nil)
)

func TestComputePermissionsToRemove(t *testing.T) {
	p := func(r, w, d, a int) itsyouonline.Permission {
		return itsyouonline.Permission{Read: r == 1, Write: w == 1, Delete: d == 1, Admin: a == 1}
	}
	testCases := []struct {
		current, toRemove, expected itsyouonline.Permission
	}{
		{p(0, 0, 0, 0), p(1, 1, 1, 1), p(0, 0, 0, 0)},
		{p(0, 0, 0, 0), p(0, 1, 0, 1), p(0, 0, 0, 0)},
		{p(0, 0, 0, 0), p(0, 0, 0, 0), p(0, 0, 0, 0)},
		{p(1, 1, 1, 1), p(0, 0, 0, 0), p(1, 1, 1, 1)},
		{p(1, 1, 1, 1), p(1, 0, 1, 0), p(0, 1, 0, 1)},
		{p(1, 1, 1, 1), p(1, 1, 1, 1), p(0, 0, 0, 0)},
	}
	for _, testCase := range testCases {
		result := computePermissionsToRemove(testCase.current, testCase.toRemove)
		assert.Equal(t, testCase.expected, result)
	}
}

func TestComputePermissionsToGive(t *testing.T) {
	p := func(r, w, d, a int) itsyouonline.Permission {
		return itsyouonline.Permission{Read: r == 1, Write: w == 1, Delete: d == 1, Admin: a == 1}
	}
	testCases := []struct {
		current, toGive, expected itsyouonline.Permission
	}{
		{p(0, 0, 0, 0), p(1, 1, 1, 1), p(1, 1, 1, 1)},
		{p(0, 0, 0, 0), p(0, 1, 0, 1), p(0, 1, 0, 1)},
		{p(0, 0, 0, 0), p(0, 0, 0, 0), p(0, 0, 0, 0)},
		{p(1, 1, 1, 1), p(0, 0, 0, 0), p(0, 0, 0, 0)},
		{p(1, 1, 1, 1), p(1, 0, 1, 0), p(0, 0, 0, 0)},
		{p(1, 1, 1, 1), p(1, 1, 1, 1), p(0, 0, 0, 0)},
	}
	for _, testCase := range testCases {
		result := computePermissionsToGive(testCase.current, testCase.toGive)
		assert.Equal(t, testCase.expected, result)
	}
}
