package pipe

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/zero-os/0-stor/client/config"
	"github.com/zero-os/0-stor/client/lib/block"
	"github.com/zero-os/0-stor/client/lib/compress"
	"github.com/zero-os/0-stor/client/lib/encrypt"
	"github.com/zero-os/0-stor/client/lib/hash"
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
	hashConf := hash.Config{
		Type: hash.TypeBlake2,
	}

	conf := config.Config{
		Pipes: []config.Pipe{
			config.Pipe{
				Name:   "pipe1",
				Type:   "compress",
				Config: compressConf,
			},
			config.Pipe{
				Name:   "pipe2",
				Type:   "encrypt",
				Config: encryptConf,
			},
			config.Pipe{
				Name:   "pipe3",
				Type:   "hash",
				Config: hashConf,
			},
		},
	}

	data := make([]byte, 4096)
	rand.Read(data)

	finalWriter := block.NewBytesBuffer()

	pw, err := NewWritePipe(&conf, finalWriter)
	if !assert.Nil(t, err) {
		return
	}

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

		// (3) hash it
		bufHash := block.NewBytesBuffer()
		hasher, err := hash.NewWriter(bufHash, hashConf)
		assert.Nil(t, err)
		resp = hasher.WriteBlock(bufEncryp.Bytes())
		assert.Nil(t, resp.Err)

		return bufHash.Bytes()
	}()

	assert.Equal(t, resultManual, finalWriter.Bytes())
}
