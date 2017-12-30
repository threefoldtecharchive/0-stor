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

package db

import (
	"encoding/binary"
)

const (
	// SequenceKeyByteSize is the amount of bytes used to store the sequence
	// as part of a ScopedSequenceKey.
	SequenceKeyByteSize = 8
)

// ScopedSequenceKey is a utility functions which allows you to generate
// a binary key for a given scope (key) and sequence (index).
//
// The last 4 bytes of the returned key will be the encoded sequence,
// while the other bytes are simply the bytes of the scopeKey as given.
func ScopedSequenceKey(scopeKey []byte, sequence uint64) []byte {
	if scopeKey == nil {
		panic("no scopeKey given")
	}
	skl := len(scopeKey)
	key := make([]byte, skl+SequenceKeyByteSize)
	copy(key, scopeKey)
	binary.LittleEndian.PutUint64(key[skl:], sequence)
	return key
}

// DataKey returns the full key based on a given label
// and the generated part of a scoped data key.
func DataKey(label, genKey []byte) []byte {
	if label == nil {
		panic("no label given")
	}
	if genKey == nil {
		panic("no key given")
	}

	labelLength := len(label)
	key := make([]byte, labelLength+len(genKey)+3)
	key[0], key[1] = 'd', ':'
	copy(key[2:], label)
	key[labelLength+2] = ':'
	copy(key[labelLength+3:], genKey)
	return key
}

// DataScopeKey returns the key that is to be used,
// as the scope key of a data key.
func DataScopeKey(label []byte) []byte {
	if label == nil {
		panic("no label given")
	}

	key := make([]byte, len(label)+3)
	key[0], key[1] = 'd', ':'
	copy(key[2:], label)
	key[len(key)-1] = ':'
	return key
}

// NamespaceKey returns the label key for a given label.
func NamespaceKey(label []byte) []byte {
	if label == nil {
		panic("no label given")
	}

	key := make([]byte, len(label)+2)
	key[0], key[1] = '@', ':'
	copy(key[2:], label)
	return key
}

// UnlistedKey can be used to turn a key into a key
// that wouldn't be listed when listing based on a prefix.
func UnlistedKey(key []byte) []byte {
	if key == nil {
		panic("no key given")
	}

	outputKey := make([]byte, len(key)+2)
	outputKey[0], outputKey[1] = '_', '_'
	copy(outputKey[2:], key)
	return outputKey
}

const (
	// StoreStatsKey is the key (name) to be used to store
	// the global store statistics.
	StoreStatsKey = "$"
)
