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
