package test

import (
	"testing"
	"github.com/zero-os/0-stor/store/scope"
	"github.com/stretchr/testify/require"
)

func TestScopeEncodeDecode(t *testing.T){
	// some decoding
	scopeStr := "user:memberof:stor.*.read"
	s := new(scope.Scope)
	err := s.Decode(scopeStr)
	require.NoError(t, err)
	require.Equal(t, s.Namespace, "")
	require.Equal(t, s.Actor, "user")
	require.Equal(t, s.Action, "memberof")
	require.Equal(t, s.Organization, "stor")
	require.Equal(t, s.Permission, "read")

	scopeStr = "user:memberof:stor.namespace"
	err = s.Decode(scopeStr)
	require.NoError(t, err)
	require.Equal(t, s.Namespace, "namespace")
	require.Equal(t, s.Actor, "user")
	require.Equal(t, s.Action, "memberof")
	require.Equal(t, s.Organization, "stor")
	require.Equal(t, s.Permission, "admin")

	scopeStr = "user:memberof:stor.*"
	err = s.Decode(scopeStr)
	require.NoError(t, err)
	require.Equal(t, s.Namespace, "")
	require.Equal(t, s.Actor, "user")
	require.Equal(t, s.Action, "memberof")
	require.Equal(t, s.Organization, "stor")
	require.Equal(t, s.Permission, "admin")


	// some encode

	s = &scope.Scope{
		Namespace:"",
		Permission:"write",
		Actor:"user",
		Action: "memberof",
		Organization: "orgo",
	}

	scopeStr, err = s.Encode()
	require.NoError(t, err)
	require.Equal(t, scopeStr, "user:memberof:orgo.*.write")

	s = &scope.Scope{
		Namespace:"",
		Permission:"admin",
		Actor:"user",
		Action: "memberof",
		Organization: "orgo",
	}

	scopeStr, err = s.Encode()
	require.NoError(t, err)
	require.Equal(t, scopeStr, "user:memberof:orgo.*")




}