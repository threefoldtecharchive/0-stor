package pipe

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/zero-os/0-stor/client/config"
	"github.com/zero-os/0-stor/client/lib/block"
	"github.com/zero-os/0-stor/client/lib/compress"
	"github.com/zero-os/0-stor/client/lib/encrypt"
)

// TestRoundTrip test that read pipe can decode back
// the output of write pipe
func TestRoundTrip(t *testing.T) {
	tests := []struct {
		name         string
		compressType string
	}{
		{"snappy", compress.TypeSnappy},
		{"gzip", compress.TypeGzip},
		{"lz4", compress.TypeLz4},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testRoundTrip(t, test.compressType)
		})
	}

}

func testRoundTrip(t *testing.T, compressType string) {
	compressConf := compress.Config{
		Type: compressType,
	}
	encryptConf := encrypt.Config{
		Type:    encrypt.TypeAESGCM,
		PrivKey: "12345678901234567890123456789012",
		Nonce:   "123456789012",
	}

	conf := config.Config{
		Pipes: []config.Pipe{
			config.Pipe{
				Name:   "pipe1",
				Type:   "compress",
				Config: compressConf,
			},
			config.Pipe{
				Name:   "type2",
				Type:   "encrypt",
				Config: encryptConf,
			},
		},
	}

	data := make([]byte, 4096)
	rand.Read(data)

	// write it
	finalWriter := block.NewBytesBuffer()

	pw, err := NewWritePipe(&conf, finalWriter)
	assert.Nil(t, err)

	resp := pw.WriteBlock(data)
	assert.Nil(t, resp.Err)

	// read it
	rp, err := NewReadPipe(&conf)
	assert.Nil(t, err)

	readResult, err := rp.ReadBlock(finalWriter.Bytes())
	assert.Nil(t, err)
	assert.Equal(t, data, readResult)
}
