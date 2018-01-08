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
	"os"
	"path"

	"github.com/zero-os/0-stor/client"
	datastor "github.com/zero-os/0-stor/client/datastor/grpc"
	"github.com/zero-os/0-stor/client/datastor/pipeline"
	"github.com/zero-os/0-stor/client/metastor"
	"github.com/zero-os/0-stor/client/metastor/db/badger"
	"github.com/zero-os/0-stor/client/processing"
)

var (
	// As we create a custom client, we can also freely choose any metastor client to use,
	// in this example we use the alternative (but still provided) metastor client,
	// backed by an internal badger database, which requires 2 directories on the local FS,
	// one to store the metadata, and one to store the actual data.
	dbDir = path.Join(
		os.Getenv("GOPATH"),
		"src", "github.com", "zero-os", "0-stor",
		"examples", "custom_client", ".db")
	dataDir = path.Join(dbDir, "data")
	metaDir = path.Join(dbDir, "meta")

	// Nn this example we use just a single zstordb server,
	// however should you desire to use replication or erasue-coding,
	// as to distribute your date, you'll need more servers.
	// If you can afford erasure-coding it is definitely recommended.
	zstordbAddresses = []string{
		// in this example we expect that the server is started
		// with the `--no-auth` flag
		"127.0.0.1:12345",
	}

	// See https://godoc.org/github.com/zero-os/0-stor/pipeline#Config
	// for more information and all options available.
	// Note that you can also create a pipeline manually using the
	// constructors of the pipeline implementations.
	pipelineConfig = pipeline.Config{
		BlockSize: 4096,
		Compression: pipeline.CompressionConfig{
			Mode: processing.CompressionModeDefault,
		},
		Encryption: pipeline.EncryptionConfig{
			PrivateKey: mySuperSecretPrivateKey,
		},
	}
)

const (
	// All data is stored under a namespace,
	// for this example we assume this namespace will be named test.
	namespace = "test"

	// private key used for the encryption of our data and metadata
	mySuperSecretPrivateKey = "abcdefghijklmnopqrstuvwxyzabcdef"
)

func main() {
	// create our badger-backed metastor database
	metaDB, err := badger.New(dataDir, metaDir)
	if err != nil {
		log.Fatal(err)
	}
	// create the metastor client,
	// with encryption enabled and our created badger DB backend
	metaClient, err := metastor.NewClient(metastor.Config{
		Database: metaDB,
		ProcessorConstructor: func() (processing.Processor, error) {
			return processing.NewAESEncrypterDecrypter([]byte(mySuperSecretPrivateKey))
		},
		// you can also customize the (un)marshal logic,
		// to use something different other than the default gogo-protobuf marshaler
	})

	// create a datastor cluster, using our predefined addresses and namespace,
	// which will be used to store the actual data
	datastorCluster, err := datastor.NewCluster(zstordbAddresses, namespace, nil)
	if err != nil {
		log.Fatal(err)
	}

	// create our pipeline which will be used to process or data prior to storage,
	// and process it once again upon reading it back from storage
	pipeline, err := pipeline.NewPipeline(pipelineConfig, datastorCluster, -1)
	if err != nil {
		log.Fatal(err)
	}

	// create our custom client
	c := client.NewClient(metaClient, pipeline)

	data := []byte("hello 0-stor")
	key := []byte("hi guys")

	// store onto 0-stor
	_, err = c.Write(key, bytes.NewReader(data))
	if err != nil {
		log.Fatal(err)
	}

	// read the data
	buf := bytes.NewBuffer(nil)
	err = c.Read(key, buf)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("stored data=%v\n", string(buf.Bytes()))
}
