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
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

func TestExtractStringFromContext(t *testing.T) {
	require := require.New(t)

	ctx := context.Background()
	require.NotNil(ctx)
	label, err := extractStringFromContext(ctx, "bar")
	require.Error(err)
	require.Empty(label)

	md := metadata.MD{}
	ctx = metadata.NewIncomingContext(ctx, md)
	require.NotNil(ctx)
	label, err = extractStringFromContext(ctx, "bar")
	require.Error(err)
	require.Empty(label)

	md["bar"] = []string{"foo"}
	label, err = extractStringFromContext(ctx, "bar")
	require.NoError(err)
	require.Equal("foo", label)
}
