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
	_, err = c.WriteF(key, bytes.NewReader(data))
	if err != nil {
		log.Fatal(err)
	}

	// read the data
	buf := bytes.NewBuffer(nil)
	err = c.ReadF(key, buf)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("stored data=%v\n", string(buf.Bytes()))
}
