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

package encoding

import (
	"math"
	"testing"

	"github.com/zero-os/0-stor/server"

	"github.com/stretchr/testify/require"
)

func TestObjectEncodingDecoding(t *testing.T) {
	require := require.New(t)

	validTestCases := []server.Object{
		{Data: []byte("1")},
		{Data: []byte("Hello, World!")},
		{Data: []byte("大家好")},
	}
	for _, validTestCase := range validTestCases {
		data, err := EncodeObject(validTestCase)
		require.NoError(err)
		require.NotNil(data)

		obj, err := DecodeObject(data)
		require.NoError(err)
		require.Equal(validTestCase, obj)
	}
}

func TestInvalidObjectDecoding(t *testing.T) {
	require := require.New(t)

	// invalid encodings as nil data was given,
	// or because not enough data was given, to be even possibly valid
	_, err := DecodeObject(nil)
	require.Equal(ErrInvalidData, err)
	_, err = DecodeObject([]byte{4, 2})
	require.Equal(ErrInvalidData, err)
	_, err = DecodeObject([]byte{1, 2, 3, 4})
	require.Equal(ErrInvalidData, err)

	// invalid crc
	_, err = DecodeObject([]byte{1, 2, 3, 4, 5})
	require.Equal(ErrInvalidChecksum, err)
}

func TestInvalidObjectEncoding(t *testing.T) {
	require := require.New(t)

	// an error is only to be expected in case the
	// given object has no data defined
	_, err := EncodeObject(server.Object{})
	require.Error(err)
	_, err = EncodeObject(server.Object{Data: nil})
	require.Error(err)
}

func TestNamespaceEncodingDecoding(t *testing.T) {
	require := require.New(t)

	validTestCases := []server.Namespace{
		{Label: []byte("1")},
		{Reserved: 1, Label: []byte("1")},
		{Reserved: 42, Label: []byte("42")},
		{Reserved: math.MaxUint64, Label: []byte("大家好")},
	}
	for _, validTestCase := range validTestCases {
		data, err := EncodeNamespace(validTestCase)
		require.NoError(err)
		require.NotNil(data)

		ns, err := DecodeNamespace(data)
		require.NoError(err)
		require.Equal(validTestCase, ns)
	}
}

func TestInvalidNamespaceDecoding(t *testing.T) {
	require := require.New(t)

	// invalid encodings as nil data was given,
	// or because not enough data was given, to be even possibly valid
	_, err := DecodeNamespace(nil)
	require.Equal(ErrInvalidData, err)
	_, err = DecodeNamespace([]byte{4, 2})
	require.Equal(ErrInvalidData, err)
	_, err = DecodeNamespace([]byte{1, 2, 3, 4})
	require.Equal(ErrInvalidData, err)
	_, err = DecodeNamespace([]byte{1, 2, 3, 4, 5, 6, 7, 8})
	require.Equal(ErrInvalidData, err)
	_, err = DecodeNamespace([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12})
	require.Equal(ErrInvalidData, err)

	// invalid crc
	_, err = DecodeNamespace([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13})
	require.Equal(ErrInvalidChecksum, err)
}

func TestInvalidNamespaceEncoding(t *testing.T) {
	require := require.New(t)

	// an error is only to be expected in case the
	// given namespace has no label defined
	_, err := EncodeNamespace(server.Namespace{})
	require.Error(err)
	_, err = EncodeNamespace(server.Namespace{Reserved: 42})
	require.Error(err)
	_, err = EncodeNamespace(server.Namespace{Reserved: 42, Label: nil})
	require.Error(err)
}

func TestStoreStatEncodingDecoding(t *testing.T) {
	require := require.New(t)

	validTestCases := []server.StoreStat{
		{},
		{SizeAvailable: 1, SizeUsed: 0},
		{SizeAvailable: 0, SizeUsed: 1},
		{SizeAvailable: 1, SizeUsed: 1},
		{SizeAvailable: math.MaxUint64, SizeUsed: 0},
		{SizeAvailable: math.MaxUint64, SizeUsed: 42},
		{SizeAvailable: 0, SizeUsed: math.MaxUint64},
		{SizeAvailable: 42, SizeUsed: math.MaxUint64},
		{SizeAvailable: 123456789, SizeUsed: 987654321},
		{SizeAvailable: math.MaxUint64, SizeUsed: math.MaxUint64},
	}
	for _, validTestCase := range validTestCases {
		data := EncodeStoreStat(validTestCase)
		require.NotNil(data)

		stat, err := DecodeStoreStat(data)
		require.NoError(err)
		require.Equal(validTestCase, stat)
	}
}

func TestInvalidStoreStatDecoding(t *testing.T) {
	require := require.New(t)

	// invalid encodings as nil data was given,
	// or because not enough data was given, to be even possibly valid
	_, err := DecodeStoreStat(nil)
	require.Equal(ErrInvalidData, err)
	_, err = DecodeStoreStat([]byte{4, 2})
	require.Equal(ErrInvalidData, err)
	_, err = DecodeStoreStat([]byte{1, 2, 3, 4})
	require.Equal(ErrInvalidData, err)
	_, err = DecodeStoreStat([]byte{1, 2, 3, 4, 5, 6, 7, 8})
	require.Equal(ErrInvalidData, err)
	_, err = DecodeStoreStat([]byte{1, 2, 3, 4,
		5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19})
	require.Equal(ErrInvalidData, err)

	// invalid crc
	_, err = DecodeStoreStat([]byte{1, 2, 3, 4,
		5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20})
	require.Equal(ErrInvalidChecksum, err)
}

func TestDataPackaging(t *testing.T) {
	require := require.New(t)

	testCases := []string{
		"1",
		"Hello, World!",
		"大家好",
	}

	for _, testCase := range testCases {
		// allocate proper data package
		data := make([]byte, checksumSize+len(testCase))
		copy(data[checksumSize:], testCase[:])

		// package our fresh data
		packageData(data)

		// unpackage it again
		blob, err := unpackageData(data)
		require.NoError(err)
		require.Equal(testCase, string(blob))

		// let's fool around a bit with our data to make it invalid,
		// the only error at this point that is really possible,
		// would be when the blob's crc (packaged as part of the data)
		// does not match the blob
		origByte := data[0]
		data[0] = data[1]
		data[1] = data[2]
		blob, err = unpackageData(data)
		require.Equal(ErrInvalidChecksum, err)
		require.Nil(blob)
		// repair the crc binary data again,
		// let's validate we can still unpackage normally
		data[1] = data[0]
		data[0] = origByte
		blob, err = unpackageData(data)
		require.NoError(err)
		require.Equal(testCase, string(blob))
		// and let's than try to mess with the data
		data[len(data)-1] = 2
		data[len(data)-2] = 4
		blob, err = unpackageData(data)
		require.Equal(ErrInvalidChecksum, err)
		require.Nil(blob)
	}
}
