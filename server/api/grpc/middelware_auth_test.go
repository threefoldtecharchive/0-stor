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

	"github.com/zero-os/0-stor/server/jwt"

	"github.com/stretchr/testify/assert"
)

func TestGetJWTMethod(t *testing.T) {
	assert := assert.New(t)
	cases := []struct {
		grpcMethod string
		method     jwt.Method
		err        bool
	}{
		{"/ObjectManager/CreateObject", jwt.MethodWrite, false},
		{"/ObjectManager/GetObject", jwt.MethodRead, false},
		{"/ObjectManager/DeleteObject", jwt.MethodDelete, false},
		{"/ObjectManager/GetObjectStatus", jwt.MethodRead, false},
		{"/ObjectManager/ListObjectKeys", jwt.MethodRead, false},
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
