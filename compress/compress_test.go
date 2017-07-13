package compress

import (
	"bytes"
	"crypto/rand"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestRoundTrip tests that compress and uncompress is the identity
// function.
func TestRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		typ  int
	}{
		{"gzip", TypeGzip},
		{"snappy", TypeSnappy},
		{"lz4", TypeLz4},
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

	buf := new(bytes.Buffer)

	// create writer
	w, err := NewWriter(conf, buf)
	assert.Nil(t, err)

	// compress by write to the writer
	_, err = w.Write(payload)
	assert.Nil(t, err)

	err = w.Flush()
	assert.Nil(t, err)

	err = w.Close()
	assert.Nil(t, err)

	// create reader
	r, err := NewReader(conf, buf)
	assert.Nil(t, err)

	// decompress by read from the reader
	b, err := ioutil.ReadAll(r)
	assert.Nil(t, err)

	err = r.Close()
	assert.Nil(t, err)

	// compare decompressed with original payload
	assert.Equal(t, payload, b)
}

func TestWriterReset(t *testing.T) {
	tests := []struct {
		name string
		typ  int
	}{
		{"gzip", TypeGzip},
		{"snappy", TypeSnappy},
		{"lz4", TypeLz4},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testWriterReset(t, Config{
				Type: test.typ,
			})
		})
	}

}

func testWriterReset(t *testing.T, conf Config) {
	buf := new(bytes.Buffer)
	buf2 := new(bytes.Buffer)

	w, err := NewWriter(conf, buf)
	assert.Nil(t, err)

	msg := []byte("hello world")
	w.Write(msg)
	w.Close()
	w.Reset(buf2)
	w.Write(msg)
	w.Close()

	assert.Equal(t, buf, buf2)
}
