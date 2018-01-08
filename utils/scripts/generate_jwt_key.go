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
	"crypto/rand"
	"flag"
	"fmt"
	"io"
	"os"
)

var (
	keySize = flag.Int("size", 2048, "size of the key")
	out     = flag.String("out", "", "destination of the generate key. If empty output on stdout")
)

func diep(msg string) {
	fmt.Fprintf(os.Stderr, "%v", msg)
	os.Exit(1)
}

func main() {
	flag.Parse()

	b := make([]byte, *keySize)
	_, err := rand.Read(b)
	if err != nil {
		diep(err.Error())
	}

	var output io.Writer
	if *out == "" {
		output = os.Stdout
	} else {
		f, err := os.OpenFile(*out, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0660)
		if err != nil {
			diep(err.Error())
		}
		defer f.Close()
		output = f
	}

	if _, err := fmt.Fprintf(output, "%x", b); err != nil {
		diep(err.Error())
	}
}
