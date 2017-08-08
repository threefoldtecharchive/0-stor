# 0-stor client   [![godoc](https://godoc.org/github.com/zero-os/0-stor/client?status.svg)](https://godoc.org/github.com/zero-os/0-stor/client)

[Specifications](specs)

API documentation : [https://godoc.org/github.com/zero-os/0-stor/client](https://godoc.org/github.com/zero-os/0-stor/client)

## Supported protocols

- REST (incomplete) : storing and retrieving data already supported
- GRPC (todo)

## Preprocessing pipe

We can process the data through configurable pipe before uploading it to 0-stor server.

Supported processing pipes:
- [compress](./lib/compress)
- [encrypt](./lib/encrypt)
- [split](./lib/chunker)
- [distribution / erasure coding](./lib/distribution) : need to be in the end of write pipe or start of read pipe
- [replication](./lib/replication) (TODO)
- [chunker](./lib/chunker) : need to be in the start of write pipe or end of read pipe

These components are not really for data processing, but can also be inserted into the pipe
- [0-stor rest/grpc client](./stor) : will be added to the end of pipe automatically if there is no component which 
  upload to 0-stor server


## Plain 0-stor client

We can use this client to store data to 0-stor server and retrieve it back without any pipe

Example

To run this example, you need to run:
- 0-stor server at port 12345
- 0-stor server at port 12346
- etcd server at port 2379

```
package main

import (
	"log"

	"github.com/zero-os/0-stor/client"
	"github.com/zero-os/0-stor/client/config"
)

func main() {

	conf := config.Config{
		Organization: "labhijau",
		Namespace:    "thedisk",
		Protocol:     "rest",
		Shards:       []string{"http://127.0.0.1:12345", "http://127.0.0.1:12346"},
		MetaShards:   []string{"http://127.0.0.1:2379"},
		IYOAppID:     "the_id",
		IYOSecret:    "the_secret",
	}
	c, err := client.New(&conf)
	if err != nil {
		log.Fatal(err)
	}

	data := []byte("hello 0-stor")
	key := []byte("hi guys")

	// stor to 0-stor
	_, err = c.Write(key, data)
	if err != nil {
		log.Fatal(err)
	}

	// read the data
	stored, err := c.Read(key)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("stored data=%v\n", string(stored))
}


```
## 0-stor client with configurable pipe

In below example, we create 0-stor client with four pipes:

- compress
- encrypt
- distribution


When we store the data, the data will be processed as follow: 
plain data -> compress -> encrypt -> distribution/erasure encoding (which send to 0-stor server and write metadata)

When we get the data from 0-stor, the reverse process will happen:
distribution/erasure decoding (which read metdata & Get data from 0-stor) -> decrypt -> decompress -> plain data.

To run this example, you need to run:
- 0-stor server at port 12345
- 0-stor server at port 12346
- etcd server at port 2379


### Example using configuration config object


```go
package main

import (
	"log"

	"github.com/zero-os/0-stor/client"
	"github.com/zero-os/0-stor/client/config"
	"github.com/zero-os/0-stor/client/lib/compress"
	"github.com/zero-os/0-stor/client/lib/distribution"
	"github.com/zero-os/0-stor/client/lib/encrypt"
)

func main() {
	compressConf := compress.Config{
		Type: compress.TypeSnappy,
	}
	encryptConf := encrypt.Config{
		Type:    encrypt.TypeAESGCM,
		PrivKey: "ab345678901234567890123456789012",
		Nonce:   "123456789012",
	}

	distConf := distribution.Config{
		Data:   1,
		Parity: 1,
	}

	conf := config.Config{
		Organization: "labhijau",
		Namespace:    "thedisk",
		Protocol:     "rest",
		IYOAppID:     "the_id",
		IYOSecret:    "the_secret",
		Shards:       []string{"http://127.0.0.1:12345", "http://127.0.0.1:12346"},
		MetaShards:   []string{"http://127.0.0.1:2379"},
		Pipes: []config.Pipe{
			config.Pipe{
				Name:   "pipe1",
				Type:   "compress",
				Config: compressConf,
			},
			config.Pipe{
				Name:   "pipe2",
				Type:   "encrypt",
				Config: encryptConf,
			},

			config.Pipe{
				Name:   "pipe3",
				Type:   "distribution",
				Config: distConf,
			},
		},
	}
	c, err := client.New(&conf)
	if err != nil {
		log.Fatal(err)
	}

	data := []byte("Hi you!")
	key := []byte("how low can you go")

	// stor to 0-stor
	_, err = c.Write(key, data)
	if err != nil {
		log.Fatal(err)
	}

	// get the value
	stored, err := c.Read(key)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("stored value=%v\n", string(stored))
}


```

### Example using configuration file 

```go
package main

import (
	"log"
	"os"

	"github.com/zero-os/0-stor/client"
	"github.com/zero-os/0-stor/client/config"
)

func main() {
	f, err := os.Open("./config.yaml")
	if err != nil {
		log.Fatal(err)
	}

	conf, err := config.NewFromReader(f)
	if err != nil {
		log.Fatal(err)
	}

	client, err := client.New(conf)
	if err != nil {
		log.Fatal(err)
	}

	data := []byte("hello 0-stor")
	key := []byte("hello")

	// stor to 0-stor
	_, err = client.Write(key, data)
	if err != nil {
		log.Fatal(err)
	}

	// read data
	stored, err := client.Read(key)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("stored value=%v\n", string(stored))

}


```

## Configuration 

Configuration file example can be found on [config.yaml](./cmd/cli/config.yaml).
All configuration can be found on https://godoc.org/github.com/zero-os/0-stor/client/config

## Libraries

This client some libraries that can be used independently.
See [lib](./lib) directory for more details.

## CLI

A simple cli can be found on [cli](./cmd/cli) directory.

## Daemon

There will be client daemon on [daemon](./cmd/daemon) directory.
