package distribution_test

import (
	"bytes"
	"fmt"
	"log"

	"github.com/zero-os/0-stor/client/components/distribution"
)

func Example() {
	// given data ...
	data := []byte("hello world")

	// we can define an encoder
	const (
		k = 4 // DataShards
		m = 2 // ParityShards
	)

	e, err := distribution.NewEncoder(k, m)
	panicOnError(err)

	// encoder encodes data
	encoded, err := e.Encode(data)
	panicOnError(err)

	// we can define decoder
	dec, err := distribution.NewDecoder(k, m)
	panicOnError(err)

	// decoder restores data
	decoded, err := dec.Decode(encoded, len(data))

	fmt.Printf("Given data: %v\n", string(data))
	fmt.Printf("Restored data: %v\n", string(decoded))

	if bytes.Compare(data, decoded) != 0 {
		log.Fatalf("restore failed")
	} else {
		fmt.Println("Data restored successfully")
	}
	// Output:
	// Given data: hello world
	// Restored data: hello world
	// Data restored successfully
}

func panicOnError(err error) {
	if err != nil {
		panic(err)
	}
}
