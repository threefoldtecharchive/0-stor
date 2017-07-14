package replication

import (
	"bytes"
	"crypto/rand"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
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
	var writers []io.Writer
	numWriter := 10
	data := make([]byte, 4096)
	rand.Read(data)

	for i := 0; i < numWriter; i++ {
		writers = append(writers, new(bytes.Buffer))
	}

	w := NewWriter(writers, async)
	n, err := w.Write(data)
	assert.Nil(t, err)
	assert.Equal(t, numWriter*len(data), n)
	for i := 0; i < numWriter; i++ {
		buff := writers[i].(*bytes.Buffer)
		assert.Equal(t, data, buff.Bytes())
	}
}
