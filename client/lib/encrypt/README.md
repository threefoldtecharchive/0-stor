# Encrypt

Will supports for both symetric and asymetric encryption.
Creation of an encrypter takes the required key(s).

Encryption supported:
- AES GCM

## Example
```
	privKey := make([]byte, aesGcmKeySize)
	nonce := make([]byte, aesGcmNonceSize)
	rand.Read(privKey)
	rand.Read(nonce)

	conf := Config{
		Type:    TypeAESGCM,
		PrivKey: string(privKey),
		Nonce:   string(nonce),
	}
	
	plain := []byte("hello world")

	// encrypt
	buf := block.NewBytesBuffer()

	w, _ := NewWriter(buf, conf)

	resp := w.WriteBlock(plain)

	// decrypt
	r, _ := NewReader(conf)

	decrypted, err := r.ReadBlock(buf.Bytes())

	assert.Equal(t, plain, decrypted)

```
