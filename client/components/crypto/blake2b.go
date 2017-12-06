package crypto

import (
	"hash"

	"github.com/minio/blake2b-simd"
)

// Blake2bHash creates and returns a hash,
// for and given some binary input data,
// using the third-party blake2b algorithm.
func Blake2bHash(data []byte) []byte {
	hashed := blake2b.Sum256(data)
	return hashed[:]
}

// NewBlake2bHasher creates a new hasher,
// using the Blake2b (32 bytes output) algorithm.
func NewBlake2bHasher(key []byte) (*Blake2bHasher, error) {
	hash, err := blake2b.New(
		&blake2b.Config{
			Size: 32,
			Key:  key,
		})
	if err != nil {
		return nil, err
	}

	return &Blake2bHasher{hash: hash}, nil
}

// Blake2bHasher defines a crypto-hasher, using the third-party blake2b algorithm.
// It can be used to create a hash, given some binary input data.
type Blake2bHasher struct {
	hash hash.Hash
}

// HashBytes implements Hasher.HashBytes
func (hasher Blake2bHasher) HashBytes(data []byte) []byte {
	hasher.hash.Reset()
	hasher.hash.Write(data)
	return hasher.hash.Sum(nil)
}
