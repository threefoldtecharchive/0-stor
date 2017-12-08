package grpc

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
		{"/ObjectManager/SetObject", jwt.MethodWrite, false},
		{"/ObjectManager/GetObject", jwt.MethodRead, false},
		{"/ObjectManager/DeleteObject", jwt.MethodDelete, false},
		{"/ObjectManager/GetObjectStatus", jwt.MethodRead, false},
		{"/ObjectManager/ListObjectKeys", jwt.MethodRead, false},
		{"/ObjectManager/SetReferenceList", jwt.MethodWrite, false},
		{"/ObjectManager/GetReferenceList", jwt.MethodRead, false},
		{"/ObjectManager/GetReferenceCount", jwt.MethodRead, false},
		{"/ObjectManager/AppendToReferenceList", jwt.MethodWrite, false},
		{"/ObjectManager/DeleteFromReferenceList", jwt.MethodDelete, false},
		{"/ObjectManager/DeleteReferenceList", jwt.MethodDelete, false},
		{"/NamespaceManager/GetNamespace", jwt.MethodAdmin, false},
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
