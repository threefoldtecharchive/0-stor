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
	"errors"
	"fmt"
	"strings"

	"github.com/zero-os/0-stor/server/api/grpc/rpctypes"
	"github.com/zero-os/0-stor/server/jwt"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

// unaryJWTAuthInterceptor creates an interceptor for a unary server method,
// which authenticates any method based on its label and name,
// using the user's JWT token.
func unaryJWTAuthInterceptor(v jwt.TokenVerifier) grpc.UnaryServerInterceptor {
	interceptor := &jwtAuthInterceptor{v}
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		err := interceptor.jwtAuthenticator(ctx, info.FullMethod)
		if err != nil {
			return ctx, err
		}
		return handler(ctx, req)
	}
}

// streamJWTAuthInterceptor creates an interceptor for a streaming server method,
// which authenticates any method based on its label and name,
// using the user's JWT token.
func streamJWTAuthInterceptor(v jwt.TokenVerifier) grpc.StreamServerInterceptor {
	interceptor := &jwtAuthInterceptor{v}
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		err := interceptor.jwtAuthenticator(stream.Context(), info.FullMethod)
		if err != nil {
			return err
		}
		return handler(srv, stream)
	}
}

type jwtAuthInterceptor struct {
	v jwt.TokenVerifier
}

func (interceptor *jwtAuthInterceptor) jwtAuthenticator(ctx context.Context, grpcMethod string) error {
	label, err := extractStringFromContext(ctx, rpctypes.MetaLabelKey)
	if err != nil {
		return rpctypes.ErrNilLabel
	}

	method, err := getJWTMethod(grpcMethod)
	if err != nil {
		return rpctypes.ErrUnimplemented
	}

	err = interceptor.v.ValidateJWT(ctx, method, label)
	if err != nil {
		return rpctypes.ErrPermissionDenied
	}

	return nil
}

func getJWTMethod(grpcMethod string) (jwt.Method, error) {
	switch {
	case strings.HasPrefix(grpcMethod, objectPrefix):
		m := grpcMethod[objectPrefixLength:]

		jwtM, ok := _JWTObjectMethodsMap[m]
		if !ok {
			return 0, errors.New("namespace object does not contain method " + m)
		}

		return jwtM, nil

	case strings.HasPrefix(grpcMethod, namespacePrefix):
		m := grpcMethod[namespacePrefixLength:]

		jwtM, ok := _JWTNamespaceMethodsMap[m]
		if !ok {
			return 0, errors.New("namespace namespace does not contain method " + m)
		}

		return jwtM, nil
	default:
		return 0, fmt.Errorf("namespace `%s` not recognized by authentication middleware", grpcMethod)
	}
}

var (
	_JWTObjectMethodsMap = map[string]jwt.Method{
		"GetObject":       jwt.MethodRead,
		"GetObjectStatus": jwt.MethodRead,
		"ListObjectKeys":  jwt.MethodRead,
		"CreateObject":    jwt.MethodWrite,
		"DeleteObject":    jwt.MethodDelete,
	}
	_JWTNamespaceMethodsMap = map[string]jwt.Method{
		"GetNamespace": jwt.MethodAdmin,
	}
)
