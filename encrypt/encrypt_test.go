package encrypt

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/zero-os/0-stor-lib/fullreadwrite"
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
	buf := fullreadwrite.NewBytesBuffer()

	w, err := NewWriter(buf, conf)
	assert.Nil(t, err)

	_, err = w.Write(plain)
	assert.Nil(t, err)

	// decrypt ag
	ag, err := newAESGCM([]byte(conf.PrivKey), []byte(conf.Nonce))
	assert.Nil(t, err)

	dag, err := ag.Decrypt(buf.Bytes())
	assert.Nil(t, err)
	assert.Equal(t, plain, dag)

	// decrypt
	reader := bytes.NewReader(buf.Bytes())
	r, err := NewReader(reader, conf)
	assert.Nil(t, err)

	decrypted := make([]byte, len(plain))
	_, err = r.Read(decrypted)
	assert.Nil(t, err)

	assert.Equal(t, plain, decrypted)
}
