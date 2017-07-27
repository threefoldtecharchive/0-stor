package config

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/zero-os/0-stor/client/fullreadwrite"
	"github.com/zero-os/0-stor/client/lib/compress"
	"github.com/zero-os/0-stor/client/lib/encrypt"
)

func TestPipeWriter(t *testing.T) {
	tests := []struct {
		name         string
		compressType int
	}{
		//{"gzip", compress.TypeGzip},
		{"snappy", compress.TypeSnappy},
		//{"lz4", compress.TypeLz4},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testPipeWriter(t, test.compressType)
		})
	}
}

func testPipeWriter(t *testing.T, compressType int) {
	compressConf := compress.Config{
		Type: compressType,
	}
	encryptConf := encrypt.Config{
		Type:    encrypt.TypeAESGCM,
		PrivKey: "12345678901234567890123456789012",
		Nonce:   "123456789012",
	}

	conf := Config{
		Pipes: []Pipe{
			Pipe{
				Name:   "pipe1",
				Type:   compressStr,
				Config: compressConf,
			},
			Pipe{
				Name:   "type2",
				Type:   encryptStr,
				Config: encryptConf,
			},
		},
	}

	data := make([]byte, 4096)
	rand.Read(data)

	finalWriter := fullreadwrite.NewBytesBuffer()

	pw, err := conf.CreatePipeWriter(finalWriter)
	assert.Nil(t, err)

	resp := pw.WriteFull(data)
	assert.Nil(t, resp.Err)

	// compare with manual writer
	resultManual := func() []byte {
		// (1) compress it
		bufComp := fullreadwrite.NewBytesBuffer()
		compressor, err := compress.NewWriter(compressConf, bufComp)
		assert.Nil(t, err)
		_, err = compressor.Write(data)
		assert.Nil(t, err)

		// (2) encrypt it
		bufEncryp := fullreadwrite.NewBytesBuffer()
		encrypter, err := encrypt.NewWriter(bufEncryp, encryptConf)
		assert.Nil(t, err)
		_, err = encrypter.Write(bufComp.Bytes())
		assert.Nil(t, err)

		return bufEncryp.Bytes()
	}()

	assert.Equal(t, resultManual, finalWriter.Bytes())
}
