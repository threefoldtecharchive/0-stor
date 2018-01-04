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
	"log"

	datastor "github.com/zero-os/0-stor/client/datastor/grpc"
)

func main() {
	// create a client to connect to a `--no-auth` zstordb server
	// which is listening on a TCP connection at the local port `:12345`
	client, err := datastor.NewClient("127.0.0.1:12345", "test", nil)
	if err != nil {
		log.Fatal(err)
	}

	data := []byte("hello 0-stor")

	// write data
	key, err := client.CreateObject(data)
	if err != nil {
		log.Fatal(err)
	}

	// read data
	object, err := client.GetObject(key)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("stored data=%v\n", string(object.Data))
}
