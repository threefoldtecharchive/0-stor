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

package bencher

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	testKey   = []byte("testKey")
	testValue = []byte("testValue")
)

func TestInMemoryMetaClient(t *testing.T) {
	require := require.New(t)

	servers, cleanupZstor := newTestZstorServers(t, 4)
	defer cleanupZstor()

	shards := make([]string, len(servers))
	for i, server := range servers {
		shards[i] = server.Address()
	}

	clientConfig := newDefaultZstorConfig(shards, nil, 64)

	client, err := newClientFromConfig(&clientConfig, 1)
	require.NoError(err, "Failed to create client")

	_, err = client.Write(testKey, bytes.NewReader(testValue))
	require.NoError(err, "Failed to write to client")

	buf := bytes.NewBuffer(nil)
	err = client.Read(testKey, buf)
	require.NoError(err, "Failed to read from client")
	require.Equal(testValue, buf.Bytes(), "Read value should be equal to value originally set in the zstor")
}
