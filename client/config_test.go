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

package client

import (
	"testing"

	"github.com/zero-os/0-stor/client/itsyouonline"
	"github.com/zero-os/0-stor/client/pipeline"
	"github.com/zero-os/0-stor/client/pipeline/processing"

	"github.com/stretchr/testify/require"
)

func TestDecodeZstorExampleConfig(t *testing.T) {
	cfg, err := ReadConfig("../cmd/zstor/config.yaml")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	expectedCfg := Config{
		IYO: itsyouonline.Config{
			Organization:      "myorg",
			ApplicationID:     "appID",
			ApplicationSecret: "secret",
		},
		Namespace: "namespace1",
		DataStor: DataStorConfig{
			Shards: []string{
				"127.0.0.1:12345",
				"127.0.0.1:12346",
				"127.0.0.1:12347",
				"127.0.0.1:12348",
			},
		},
		MetaStor: MetaStorConfig{
			Shards: []string{
				"127.0.0.1:2379",
				"127.0.0.1:22379",
				"127.0.0.1:32379",
			},
		},
		Pipeline: pipeline.Config{
			BlockSize: 4096,
			Compression: pipeline.CompressionConfig{
				Type: processing.CompressionTypeSnappy,
				Mode: processing.CompressionModeDefault,
			},
			Encryption: pipeline.EncryptionConfig{
				Type:       processing.EncryptionTypeAES,
				PrivateKey: "ab345678901234567890123456789012",
			},
			Distribution: pipeline.ObjectDistributionConfig{
				DataShardCount:   3,
				ParityShardCount: 1,
			},
		},
	}

	require.Equal(t, expectedCfg, *cfg)
}

func TestReadConfigErrors(t *testing.T) {
	require := require.New(t)

	cfg, err := ReadConfig("")
	require.Error(err, "invalid path")
	require.Nil(cfg)

	cfg, err = ReadConfig(testPrivateKeyPath)
	require.Error(err, "invalid config")
	require.Nil(cfg)
}

const testPrivateKeyPath = "../devcert/jwt_key.pem"
