package pipe

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/zero-os/0-stor-lib/compress"
	"github.com/zero-os/0-stor-lib/config"
	"github.com/zero-os/0-stor-lib/encrypt"
)

func TestRoundTrip(t *testing.T) {
	compressConf := compress.Config{
		Type: compress.TypeSnappy,
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
				Action: "write",
				Config: compressConf,
			},
			config.Pipe{
				Name:   "type2",
				Type:   "encrypt",
				Action: "write",
				Config: encryptConf,
			},
		},
	}

	data := make([]byte, 4096)
	rand.Read(data)

	// write it
	finalWriter := new(bytes.Buffer)

	pw, err := conf.CreatePipeWriter(finalWriter)
	assert.Nil(t, err)

	_, err = pw.Write(data)
	assert.Nil(t, err)

	// read it
	ars, err := conf.CreateAllReaders()
	assert.Nil(t, err)

	rp := NewReadPipe(ars)
	readResult, err := rp.ReadAll(finalWriter.Bytes())
	assert.Nil(t, err)
	assert.Equal(t, data, readResult)
}
