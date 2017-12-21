package main

import (
	"bytes"
	"log"
	"os"
	"path/filepath"

	"github.com/zero-os/0-stor/client"
)

var configPath = filepath.Join(
	os.Getenv("GOPATH"),
	"src/github.com/zero-os/0-stor/examples/hello_config/config.yaml")

func main() {
	// This example doesn't use IYO-based Authentication,
	// and only works if the zstordb servers used have the `--no-auth` flag defined.
	config, err := client.ReadConfig(configPath)
	if err != nil {
		log.Fatal(err)
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
