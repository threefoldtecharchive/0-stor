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
	"github.com/zero-os/0-stor/server/stats"

	log "github.com/Sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

// unaryStatsInterceptor creates an interceptor for a unary server method,
// which collects global read/write statistics.
// The method name defines whether it counts as read or write.
func unaryStatsInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		go statsLogger(ctx, info.FullMethod)

		return handler(ctx, req)
	}
}

// streamStatsInterceptor creates an interceptor for a streaming server method,
// which collects global read/write statistics.
// The method name defines whether it counts as read or write.
func streamStatsInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		go statsLogger(stream.Context(), info.FullMethod)

		return handler(srv, stream)
	}
}

func statsLogger(ctx context.Context, grpcMethod string) {
	label, err := extractStringFromContext(ctx, rpctypes.MetaLabelKey)
	if err != nil {
		log.Errorf("Stat was not logged due to error: %v", err)
	}

	statsFunc, err := getStatsFunc(grpcMethod)
	if err != nil {
		log.Errorf("Stat was not logged due to error: %v", err)
	}
	statsFunc(label)
}

func getStatsFunc(grpcMethod string) (labelStatsFunc, error) {
	switch {
	case strings.HasPrefix(grpcMethod, objectPrefix):
		m := grpcMethod[objectPrefixLength:]
		f, ok := _StatsObjectMethodsMap[m]
		if !ok {
			return nil, errors.New("namespace object does not contain method " + m)
		}
		return f, nil

	case strings.HasPrefix(grpcMethod, namespacePrefix):
		m := grpcMethod[namespacePrefixLength:]
		f, ok := _StatsNamespaceMethodsMap[m]
		if !ok {
			return nil, errors.New("namespace namespace does not contain method " + m)
		}
		return f, nil

	default:
		return nil, fmt.Errorf("namespace `%s` not recognized by authentication middleware", grpcMethod)
	}
}

type labelStatsFunc func(label string)

var (
	_StatsObjectMethodsMap = map[string]labelStatsFunc{
		"GetObject":       stats.IncrRead,
		"GetObjectStatus": stats.IncrRead,
		"ListObjectKeys":  stats.IncrRead,
		"CreateObject":    stats.IncrWrite,
		"DeleteObject":    stats.IncrWrite,
	}
	_StatsNamespaceMethodsMap = map[string]labelStatsFunc{
		"GetNamespace": stats.IncrRead,
	}
)
