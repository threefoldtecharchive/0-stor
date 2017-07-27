package compress

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/zero-os/0-stor-lib/fullreadwrite"
)

// TestRoundTrip tests that compress and uncompress is the identity
// function.
func TestRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		typ  int
	}{
		//{"gzip", TypeGzip},
		{"snappy", TypeSnappy},
		//{"lz4", TypeLz4},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testRoundTrip(t, Config{
				Type: test.typ,
			})
		})
	}
}

func testRoundTrip(t *testing.T, conf Config) {
	payload := make([]byte, 4096)
	rand.Read(payload)

	buf := fullreadwrite.NewBytesBuffer()

	// create writer
	w, err := NewWriter(conf, buf)
	assert.Nil(t, err)

	// compress by write to the writer
	_, err = w.Write(payload)
	assert.Nil(t, err)

	// create reader
	r, err := NewReader(conf, buf)
	if !assert.Nil(t, err) {
		return
	}

	// decompress by read from the reader
	b, err := r.ReadAll(buf.Bytes())
	if !assert.Nil(t, err) {
		return
	}

	// compare decompressed with original payload
	assert.Equal(t, payload, b)
}
