package config

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/zero-os/0-stor-lib/compress"
	"github.com/zero-os/0-stor-lib/encrypt"
	"github.com/zero-os/0-stor-lib/hash"
)

func TestPipeWriter(t *testing.T) {
	tests := []struct {
		name         string
		compressType int
	}{
		{"gzip", compress.TypeGzip},
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
	hashConf := hash.Config{
		Type: hash.TypeBlake2,
	}

	conf := Config{
		Pipes: []Pipe{
			Pipe{
				Name:   "pipe1",
				Type:   compressStr,
				Action: "write",
				Config: compressConf,
			},
			Pipe{
				Name:   "type2",
				Type:   encryptStr,
				Action: "write",
				Config: encryptConf,
			},
			Pipe{
				Name:   "type2",
				Type:   hashStr,
				Action: "write",
				Config: hash.Config{
					Type: hash.TypeBlake2,
				},
			},
		},
	}

	data := make([]byte, 4096)
	rand.Read(data)

	finalWriter := new(bytes.Buffer)

	pw, err := conf.CreatePipeWriter(finalWriter)
	assert.Nil(t, err)

	_, err = pw.Write(data)
	assert.Nil(t, err)

	// compare with manual writer
	resultManual := func() []byte {
		// (1) compress it
		bufComp := new(bytes.Buffer)
		compressor, err := compress.NewWriter(compressConf, bufComp)
		assert.Nil(t, err)
		_, err = compressor.Write(data)
		assert.Nil(t, err)

		// (2) encrypt it
		bufEncryp := new(bytes.Buffer)
		encrypter, err := encrypt.NewWriter(bufEncryp, encryptConf)
		assert.Nil(t, err)
		_, err = encrypter.Write(bufComp.Bytes())
		assert.Nil(t, err)

		// (3) hash it
		bufHash := new(bytes.Buffer)
		hasher, err := hash.NewWriter(bufHash, hashConf)
		assert.Nil(t, err)
		_, err = hasher.Write(bufEncryp.Bytes())
		assert.Nil(t, err)

		return bufHash.Bytes()
	}()

	assert.Equal(t, resultManual, finalWriter.Bytes())
}
