package encrypt

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/zero-os/0-stor/client/lib/block"
)

func TestRoundTrip(t *testing.T) {
	t.Run("aes gcm", func(t *testing.T) {
		privKey := make([]byte, aesGcmKeySize)
		nonce := make([]byte, aesGcmNonceSize)
		rand.Read(privKey)
		rand.Read(nonce)

		conf := Config{
			Type:    TypeAESGCM,
			PrivKey: string(privKey),
			Nonce:   string(nonce),
		}
		testRoundTrip(t, conf)
	})
}

func testRoundTrip(t *testing.T, conf Config) {
	plain := []byte("hello world")

	// encrypt
	buf := block.NewBytesBuffer()

	w, err := NewWriter(buf, conf)
	assert.Nil(t, err)

	resp := w.WriteBlock(plain)
	assert.Nil(t, resp.Err)

	// decrypt ag
	ag, err := newAESGCM([]byte(conf.PrivKey), []byte(conf.Nonce))
	assert.Nil(t, err)

	dag, err := ag.Decrypt(buf.Bytes())
	assert.Nil(t, err)
	assert.Equal(t, plain, dag)

	// decrypt
	r, err := NewReader(conf)
	assert.Nil(t, err)

	decrypted, err := r.ReadBlock(buf.Bytes())
	assert.Nil(t, err)

	assert.Equal(t, plain, decrypted)
}
