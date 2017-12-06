package crypto

import (
	"crypto/sha256"
	"crypto/sha512"
	"hash"
)

// NewSHA256hasher creates a new hasher,
// using the SHA256 (32 bytes output) algorithm.
func NewSHA256hasher() (*SHA256Hasher, error) {
	return &SHA256Hasher{hash: sha256.New()}, nil
}

// SHA256Hash creates and returns a hash,
// for and given some binary input data,
// using the std sha256 algorithm.
func SHA256Hash(data []byte) []byte {
	hash := sha256.Sum256(data)
	return hash[:]
}

// SHA256Hasher defines a crypto-hasher, using the std SHA256 algorithm.
// It can be used to create a hash, given some binary input data.
type SHA256Hasher struct {
	hash hash.Hash
}

// HashBytes implements Hasher.HashBytes
func (hasher SHA256Hasher) HashBytes(data []byte) []byte {
	hasher.hash.Reset()
	hasher.hash.Write(data)
	hash := hasher.hash.Sum(nil)
	return hash[:]
}

// SHA512Hash creates and returns a hash,
// for and given some binary input data,
// using the std sha512 algorithm.
func SHA512Hash(data []byte) []byte {
	hash := sha512.Sum512(data)
	return hash[:]
}

// NewSHA512hasher creates a new hasher,
// using the SHA512 (64 bytes output) algorithm.
func NewSHA512hasher() (*SHA512Hasher, error) {
	return &SHA512Hasher{hash: sha512.New()}, nil
}

// SHA512Hasher defines a crypto-hasher, using the std SHA512 algorithm.
// It can be used to create a hash, given some binary input data.
type SHA512Hasher struct {
	hash hash.Hash
}

// HashBytes implements Hasher.HashBytes
func (hasher SHA512Hasher) HashBytes(data []byte) []byte {
	hasher.hash.Reset()
	hasher.hash.Write(data)
	hash := hasher.hash.Sum(nil)
	return hash[:]
}
