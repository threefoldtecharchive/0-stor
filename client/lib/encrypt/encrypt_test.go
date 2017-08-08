package encrypt

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zero-os/0-stor/client/lib/block"
	"github.com/zero-os/0-stor/client/meta"
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
	require.Nil(t, err)

	md, err := meta.New(nil, 0, nil)
	require.Nil(t, err)

	_, err = w.WriteBlock(nil, plain, md)
	require.Nil(t, err)

	// decrypt ag
	ag, err := newAESGCM([]byte(conf.PrivKey), []byte(conf.Nonce))
	require.Nil(t, err)

	dag, err := ag.Decrypt(buf.Bytes())
	require.Nil(t, err)
	require.Equal(t, plain, dag)

	// decrypt
	r, err := NewReader(conf)
	require.Nil(t, err)

	decrypted, err := r.ReadBlock(buf.Bytes())
	require.Nil(t, err)

	require.Equal(t, plain, decrypted)
}
