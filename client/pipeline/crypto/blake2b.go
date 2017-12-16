package crypto

import (
	"hash"

	"github.com/minio/blake2b-simd"
)

// SumBlake2b256 creates and returns a hash,
// for and given some binary input data,
// using the third-party blake2b-256 algorithm.
func SumBlake2b256(data []byte) []byte {
	hashed := blake2b.Sum256(data)
	return hashed[:]
}

// NewBlake2b256Hasher creates a new hasher,
// using the Blake2b (32 bytes output) algorithm.
func NewBlake2b256Hasher(key []byte) (*Blake2b256Hasher, error) {
	hash, err := blake2b.New(
		&blake2b.Config{
			Size: 32,
			Key:  key,
		})
	if err != nil {
		return nil, err
	}

	return &Blake2b256Hasher{hash: hash}, nil
}

// Blake2b256Hasher defines a crypto-hasher,
// using the third-party blake2b-256 algorithm.
// It can be used to create a hash,
// given some binary input data.
type Blake2b256Hasher struct {
	hash hash.Hash
}

// HashBytes implements Hasher.HashBytes
func (hasher Blake2b256Hasher) HashBytes(data []byte) []byte {
	hasher.hash.Reset()
	hasher.hash.Write(data)
	return hasher.hash.Sum(nil)
}

// SumBlake2b512 creates and returns a hash,
// for and given some binary input data,
// using the third-party blake2b-512 algorithm.
func SumBlake2b512(data []byte) []byte {
	hashed := blake2b.Sum512(data)
	return hashed[:]
}

// NewBlake2b512Hasher creates a new hasher,
// using the Blake2b (64 bytes output) algorithm.
func NewBlake2b512Hasher(key []byte) (*Blake2b512Hasher, error) {
	hash, err := blake2b.New(
		&blake2b.Config{
			Size: 64,
			Key:  key,
		})
	if err != nil {
		return nil, err
	}

	return &Blake2b512Hasher{hash: hash}, nil
}

// Blake2b512Hasher defines a crypto-hasher,
// using the third-party blake2b-512 algorithm.
// It can be used to create a hash,
// given some binary input data.
type Blake2b512Hasher struct {
	hash hash.Hash
}

// HashBytes implements Hasher.HashBytes
func (hasher Blake2b512Hasher) HashBytes(data []byte) []byte {
	hasher.hash.Reset()
	hasher.hash.Write(data)
	return hasher.hash.Sum(nil)
}

func init() {
	RegisterHasher(HashTypeBlake2b256, "blake2b_256", func(key []byte) (Hasher, error) {
		return NewBlake2b256Hasher(key)
	})
	RegisterHasher(HashTypeBlake2b512, "blake2b_512", func(key []byte) (Hasher, error) {
		return NewBlake2b512Hasher(key)
	})
}
