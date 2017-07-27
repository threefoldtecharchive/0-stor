package encrypt

import (
	"crypto/aes"
	"crypto/cipher"
	"io"
)

// Key and nonce size
const (
	aesGcmNonceSize = 12
	aesGcmKeySize   = 32
)

type aesgcm struct {
	w     io.Writer
	gcm   cipher.AEAD
	nonce []byte
}

func newAESGCM(privKey, nonce []byte) (*aesgcm, error) {
	if len(privKey) != aesGcmKeySize {
		return nil, errInvalidPrivKeyLen
	}
	if len(nonce) != aesGcmNonceSize {
		return nil, errInvalidNonceLen
	}

	block, err := aes.NewCipher(privKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return &aesgcm{
		gcm:   gcm,
		nonce: nonce,
	}, nil
}

func (ag *aesgcm) Encrypt(plain []byte) []byte {
	return ag.gcm.Seal(nil, ag.nonce, plain, nil)
}

func (ag *aesgcm) Decrypt(cipher []byte) ([]byte, error) {
	return ag.gcm.Open(nil, ag.nonce, cipher, nil)
}
