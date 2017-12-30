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

	"github.com/stretchr/testify/require"
)

func TestNewClusterExplicitErrors(t *testing.T) {
	require := require.New(t)

	cluster, err := NewCluster(nil, "foo", nil)
	require.Error(err, "no addresses given")
	require.Nil(cluster)

	cluster, err = NewCluster([]string{"foo"}, "", nil)
	require.Error(err, "no label given")
	require.Nil(cluster)

	cluster, err = NewCluster(nil, "", nil)
	require.Error(err, "no addresses given, nor a label given")
	require.Nil(cluster)
}
