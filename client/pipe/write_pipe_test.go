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

func TestPipeWriter(t *testing.T) {
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
			testPipeWriter(t, test.compressType)
		})
	}
}

func testPipeWriter(t *testing.T, compressType string) {
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

	finalWriter := block.NewBytesBuffer()

	pw, err := NewWritePipe(&conf, finalWriter)
	assert.Nil(t, err)

	resp := pw.WriteBlock(data)
	assert.Nil(t, resp.Err)

	// compare with manual writer
	resultManual := func() []byte {
		// (1) compress it
		bufComp := block.NewBytesBuffer()
		compressor, err := compress.NewWriter(compressConf, bufComp)
		assert.Nil(t, err)
		resp := compressor.WriteBlock(data)
		assert.Nil(t, resp.Err)

		// (2) encrypt it
		bufEncryp := block.NewBytesBuffer()
		encrypter, err := encrypt.NewWriter(bufEncryp, encryptConf)
		assert.Nil(t, err)
		resp = encrypter.WriteBlock(bufComp.Bytes())
		assert.Nil(t, resp.Err)

		return bufEncryp.Bytes()
	}()

	assert.Equal(t, resultManual, finalWriter.Bytes())
}
