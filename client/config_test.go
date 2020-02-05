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
	"crypto/tls"
	"testing"

	"github.com/threefoldtech/0-stor/client/datastor"
	"github.com/threefoldtech/0-stor/client/datastor/pipeline"
	"github.com/threefoldtech/0-stor/client/processing"

	"github.com/stretchr/testify/require"
)

func TestDecodeZstorExampleConfig(t *testing.T) {
	cfg, err := ReadConfig("../cmd/zstor/config.yaml")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	expectedCfg := Config{
		Password:  "mypass",
		Namespace: "namespace1",
		DataStor: DataStorConfig{
			Shards: []datastor.ShardConfig{
				{Address: "127.0.0.1:12345"},
				{Address: "127.0.0.1:12346"},
				{Address: "127.0.0.1:12347"},
				{Address: "127.0.0.1:12348"},
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
		},
	}

	require.Equal(t, expectedCfg, *cfg)
}

func TestReadConfigErrors(t *testing.T) {
	require := require.New(t)

	cfg, err := ReadConfig("")
	require.Error(err, "invalid path")
	require.Nil(cfg)

	cfg, err = ReadConfig("client.go")
	require.Error(err, "invalid config")
	require.Nil(cfg)
}

func TestTLSVersionOrDefaultConfig(t *testing.T) {
	var v TLSVersion
	require.Equal(t, uint16(tls.VersionTLS10), v.VersionTLSOrDefault(tls.VersionTLS10))
	require.Equal(t, uint16(tls.VersionTLS11), v.VersionTLSOrDefault(tls.VersionTLS11))
	require.Equal(t, uint16(tls.VersionTLS12), v.VersionTLSOrDefault(tls.VersionTLS12))
}

func TestTLSVersionMarshalUnmarshal(t *testing.T) {
	testCases := []TLSVersion{
		UndefinedTLSVersion,
		TLSVersion10, TLSVersion11, TLSVersion12,
	}
	for _, testCase := range testCases {
		text, err := testCase.MarshalText()
		require.NoError(t, err)
		var v TLSVersion
		err = v.UnmarshalText(text)
		require.NoError(t, err)
		require.Equal(t, testCase, v)
	}
}

func TestTLSVersionConfig(t *testing.T) {
	tt := []struct {
		input    string
		expected uint16
		value    string
	}{
		{
			"TLS10",
			tls.VersionTLS10,
			"TLS10",
		},
		{
			"tls10",
			tls.VersionTLS10,
			"TLS10",
		},
		{
			"tLS10",
			tls.VersionTLS10,
			"TLS10",
		},
		{
			"TLS11",
			tls.VersionTLS11,
			"TLS11",
		},
		{
			"tls11",
			tls.VersionTLS11,
			"TLS11",
		},
		{
			"TLS12",
			tls.VersionTLS12,
			"TLS12",
		},
		{
			"tls12",
			tls.VersionTLS12,
			"TLS12",
		},
		{
			"foo",
			0,
			"",
		},
	}

	for _, tc := range tt {
		t.Run(tc.input, func(t *testing.T) {
			var v TLSVersion
			err := v.UnmarshalText([]byte(tc.input))
			if tc.value == "" {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expected, v.VersionTLSOrDefault(0))
			require.Equal(t, tc.value, v.String())
		})
	}
}
