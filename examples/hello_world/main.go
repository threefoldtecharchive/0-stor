package main

import (
	"bytes"
	"log"

	"github.com/zero-os/0-stor/client"
	"github.com/zero-os/0-stor/client/pipeline"
	"github.com/zero-os/0-stor/client/pipeline/processing"
)

func main() {
	// This example doesn't use IYO-based Authentication,
	// and only works if the zstordb servers used have the `--no-auth` flag defined.
	config := client.Config{
		Namespace: "thedisk",
		DataStor: client.DataStorConfig{
			Shards: []string{"127.0.0.1:12345", "127.0.0.1:12346", "127.0.0.1:12347"},
		},
		MetaStor: client.MetaStorConfig{
			Shards: []string{"127.0.0.1:2379"},
		},
		Pipeline: pipeline.Config{
			BlockSize: 4096,
			Compression: pipeline.CompressionConfig{
				Mode: processing.CompressionModeDefault,
			},
			Encryption: pipeline.EncryptionConfig{
				PrivateKey: "ab345678901234567890123456789012",
			},
			Distribution: pipeline.ObjectDistributionConfig{
				DataShardCount:   2,
				ParityShardCount: 1,
			},
		},
	}

	c, err := client.NewClientFromConfig(config, -1) // use default job count
	if err != nil {
		log.Fatal(err)
	}

	data := []byte("hello 0-stor")
	key := []byte("hi guys")

	// store onto 0-stor
	_, err = c.WriteF(key, bytes.NewReader(data), nil)
	if err != nil {
		log.Fatal(err)
	}

	// read the data
	buf := bytes.NewBuffer(nil)
	_, err = c.ReadF(key, buf)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("stored data=%v\n", string(buf.Bytes()))
}
