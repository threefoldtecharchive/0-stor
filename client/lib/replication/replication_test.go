package replication

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/zero-os/0-stor/client/lib/block"
)

func TestReplication(t *testing.T) {
	t.Run("async", func(t *testing.T) {
		testReplication(t, true)
	})
	t.Run("sync", func(t *testing.T) {
		testReplication(t, false)
	})
}

func testReplication(t *testing.T, async bool) {
	var writers []block.Writer
	numWriter := 10
	data := make([]byte, 4096)
	rand.Read(data)

	// create block writers which do the replication
	for i := 0; i < numWriter; i++ {
		writers = append(writers, block.NewBytesBuffer())
	}

	w := NewWriter(writers, Config{Async: async})
	resp := w.WriteBlock(data)
	assert.Nil(t, resp.Err)
	assert.Equal(t, numWriter*len(data), resp.Written)
	for i := 0; i < numWriter; i++ {
		buff := writers[i].(*block.BytesBuffer)
		assert.Equal(t, data, buff.Bytes())
	}
}
