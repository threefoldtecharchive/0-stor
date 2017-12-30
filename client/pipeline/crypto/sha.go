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
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"hash"
)

// NewSHA256Hasher creates a new hasher,
// using the SHA256 (32 bytes output) algorithm.
//
// Key is an optional private key to add authentication to the output,
// when the key is not given the hasher will produce
// cryptographically secure checksums, without any proof of ownership.
func NewSHA256Hasher(key []byte) (*SHA256Hasher, error) {
	if key == nil {
		return &SHA256Hasher{hash: sha256.New()}, nil
	}

	h := hmac.New(sha256.New, key)
	return &SHA256Hasher{hash: h}, nil
}

// SumSHA256 creates and returns a hash,
// for and given some binary input data,
// using the std sha256 algorithm.
func SumSHA256(data []byte) []byte {
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

// SumSHA512 creates and returns a hash,
// for and given some binary input data,
// using the std sha512 algorithm.
func SumSHA512(data []byte) []byte {
	hash := sha512.Sum512(data)
	return hash[:]
}

// NewSHA512Hasher creates a new hasher,
// using the SHA512 (64 bytes output) algorithm.
//
// Key is an optional private key to add authentication to the output,
// when the key is not given the hasher will produce
// cryptographically secure checksums, without any proof of ownership.
func NewSHA512Hasher(key []byte) (*SHA512Hasher, error) {
	if key == nil {
		return &SHA512Hasher{hash: sha512.New()}, nil
	}

	h := hmac.New(sha512.New, key)
	return &SHA512Hasher{hash: h}, nil
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

func init() {
	RegisterHasher(HashTypeSHA256, "sha_256", func(key []byte) (Hasher, error) {
		return NewSHA256Hasher(key)
	})
	RegisterHasher(HashTypeSHA512, "sha_512", func(key []byte) (Hasher, error) {
		return NewSHA512Hasher(key)
	})
}
