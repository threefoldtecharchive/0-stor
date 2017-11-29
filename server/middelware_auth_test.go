package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zero-os/0-stor/server/jwt"
)

func TestGetJWTMethod(t *testing.T) {
	assert := assert.New(t)
	cases := []struct {
		grpcMethod string
		method     jwt.Method
		err        bool
	}{
		{"/ObjectManager/Get", jwt.MethodRead, false},
		{"/ObjectManager/List", jwt.MethodRead, false},
		{"/ObjectManager/Exists", jwt.MethodRead, false},
		{"/ObjectManager/Check", jwt.MethodRead, false},
		{"/ObjectManager/Create", jwt.MethodWrite, false},
		{"/ObjectManager/SetReferenceList", jwt.MethodWrite, false},
		{"/ObjectManager/AppendReferenceList", jwt.MethodWrite, false},
		{"/ObjectManager/RemoveReferenceList", jwt.MethodWrite, false},
		{"/ObjectManager/Delete", jwt.MethodDelete, false},
		{"/NamespaceManager/Get", jwt.MethodAdmin, false},
		{"", 0, true},
		{"/ObjectManager/", 0, true},
		{"/NamespaceManager/", 0, true},
		{"/ObjectManager/Foo", 0, true},
		{"/NamespaceManager/Bar", 0, true},
	}

	for _, c := range cases {
		m, err := getJWTMethod(c.grpcMethod)
		if c.err {
			assert.Error(err)
		} else {
			assert.Equal(c.method, m)
			assert.NoError(err)
		}
	}
}
