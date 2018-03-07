/*
 * Copyright (C) 2017-2018 GIG Technology NV and Contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package crypto

import (
	"hash"

	"golang.org/x/crypto/blake2b"
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
	hash, err := blake2b.New256(key)
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
	hash, err := blake2b.New512(key)
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
