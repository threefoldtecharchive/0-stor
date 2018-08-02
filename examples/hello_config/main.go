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
	"path/filepath"

	"github.com/threefoldtech/0-stor/client"
)

var configPath = filepath.Join(
	os.Getenv("GOPATH"),
	"src/github.com/threefoldtech/0-stor/examples/hello_config/config.yaml")

func main() {
	config, err := client.ReadConfig(configPath)
	if err != nil {
		log.Fatal(err)
	}

	c, err := client.NewClientFromConfig(*config, nil, -1) // use default job count
	if err != nil {
		log.Fatal(err)
	}

	data := []byte("hello 0-stor")
	key := []byte("hi guys")

	// store onto 0-stor
	md, err := c.Write(key, bytes.NewReader(data))
	if err != nil {
		log.Fatal(err)
	}

	// read the data
	buf := bytes.NewBuffer(nil)
	err = c.Read(*md, buf)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("stored data=%v\n", string(buf.Bytes()))
}
