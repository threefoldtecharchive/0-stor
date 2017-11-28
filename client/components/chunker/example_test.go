package chunker_test

import (
	"fmt"

	"github.com/zero-os/0-stor/client/components/chunker"
)

func Example() {
	// given chunker config ...
	conf := chunker.Config{
		ChunkSize: 10,
	}

	// given data slice
	data := make([]byte, 25)

	// we can split data in chunks
	c := chunker.NewChunker(conf)
	c.Chunk(data)
	for c.Next() {
		val := c.Value()
		fmt.Printf("val=%v\n", val)
	}
	// Output:
	// val=[0 0 0 0 0 0 0 0 0 0]
	// val=[0 0 0 0 0 0 0 0 0 0]
	// val=[0 0 0 0 0]
}
