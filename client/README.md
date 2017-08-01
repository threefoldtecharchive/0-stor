# 0-stor client   [![godoc](https://godoc.org/github.com/zero-os/0-stor/client?status.svg)](https://godoc.org/github.com/zero-os/0-stor/client)

[Specifications](specs)
API documentation : [https://godoc.org/github.com/zero-os/0-stor/client](https://godoc.org/github.com/zero-os/0-stor/client)

## Usage

It needs configuration file in order to be used.

Example

```go
import (
	"log"

	"github.com/zero-os/0-stor/client"
)

func main() {
	client, err := client.New("./simple.yaml")
	if err != nil {
		log.Fatal(err)
	}
	
	data := make([]byte, 4096)

	// store data with key = the_key
	client.Store([]byte("the_key"), data)
}

```

## Configuration 

Configuration file example can be found on [simple.yaml](./cmd/cli/simple.yaml).
All configuration can be found on https://godoc.org/github.com/zero-os/0-stor/client/config

## Libraries

This client some libraries that can be used independently.
See [lib](./lib) directory for more details.

## CLI

A simple cli can be found on [cli](./cmd/cli) directory.

## Daemon

There will be client daemon on [daemon](./cmd/daemon) directory.
