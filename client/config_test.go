package client

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zero-os/0-stor/client/itsyouonline"
	"github.com/zero-os/0-stor/client/pipeline"
	"github.com/zero-os/0-stor/client/pipeline/processing"
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
