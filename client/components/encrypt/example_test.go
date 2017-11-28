package encrypt_test

import (
	"bytes"
	"fmt"
	"log"

	"github.com/zero-os/0-stor/client/components/encrypt"
)

func Example() {
	// given data ...
	data := []byte("hello world")

	// given private key ...
	privKey := "france-though-athens-every-force"

	// we can define config
	conf := encrypt.Config{
		Type:    encrypt.TypeAESGCM,
		PrivKey: privKey,
	}

	// we can define EncypterDecripter
	encdec, err := encrypt.NewEncrypterDecrypter(conf)
	panicOnError(err)

	// encdec is used to encrypt and decrypt data
	chiper, err := encdec.Encrypt(data)
	panicOnError(err)

	decrypted, err := encdec.Decrypt(chiper)
	panicOnError(err)

	fmt.Printf("Initial data: %v\n", string(data))
	fmt.Printf("Result of encryption+decryption: %v\n", string(decrypted))
	if bytes.Compare(data, decrypted) != 0 {
		log.Fatalf("decryption failed")
	}
	fmt.Println("Decrypted data matches initial data")
	// Output:
	// Initial data: hello world
	// Result of encryption+decryption: hello world
	// Decrypted data matches initial data
}

func panicOnError(err error) {
	if err != nil {
		panic(err)
	}
}
