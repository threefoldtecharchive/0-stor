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

package processing

import (
	"crypto/rand"
	mathRand "math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAESEncrypterDecrypter_InvalidKeyErr(t *testing.T) {
	require := require.New(t)
	for i := 0; i < 40; i++ {
		ed, err := NewAESEncrypterDecrypter([]byte(randomString(i)))
		if i == 16 || i == 24 || i == 32 {
			require.NoError(err)
			require.NotNil(ed)
			testProcessorReadWrite(t, ed)
			continue
		}

		require.Error(err)
		require.Nil(ed)
	}
}

func TestAESEncrypterDecrypter_UniqueOutputValues(t *testing.T) {
	t.Run("aes_128", func(t *testing.T) {
		testAEsEncrypterDecrypterUniqueOutputValues(t, 16)
	})
	t.Run("aes_192", func(t *testing.T) {
		testAEsEncrypterDecrypterUniqueOutputValues(t, 24)
	})
	t.Run("aes_256", func(t *testing.T) {
		testAEsEncrypterDecrypterUniqueOutputValues(t, 32)
	})
}

func testAEsEncrypterDecrypterUniqueOutputValues(t *testing.T, keySize int) {
	require := require.New(t)

	for i := 0; i < 512; i++ {
		// generate 2 random keys
		key1 := randomString(keySize)
		require.Len(key1, keySize)
		key2 := key1
		for key2 == key1 {
			key2 = randomString(keySize)
		}
		require.NotEqual(key1, key2)

		// generate random input value
		inputData := make([]byte, mathRand.Int31n(512)+1)
		rand.Read(inputData)

		// create first EC, and encrypt, and also validate it can be decrypted
		ec1, err := NewAESEncrypterDecrypter([]byte(key1))
		require.NoError(err)
		require.NotNil(ec1)

		data1, err := ec1.WriteProcess(inputData)
		require.NoError(err)
		require.NotEqual(inputData, data1)

		outputData, err := ec1.ReadProcess(data1)
		require.NoError(err)
		require.Equal(inputData, outputData)

		// create second EC, and encrypt, and also validate it can be decrypted
		ec2, err := NewAESEncrypterDecrypter([]byte(key2))
		require.NoError(err)
		require.NotNil(ec2)

		data2, err := ec2.WriteProcess(inputData)
		require.NoError(err)
		require.NotEqual(inputData, data2)

		outputData, err = ec2.ReadProcess(data2)
		require.NoError(err)
		require.Equal(inputData, outputData)

		// ensure that the the encrypted values,
		// produces by ec1 and ec2 are different
		// as this should be the case in case the keys are different
		require.NotEqual(data1, data2)
	}
}

func TestAESEncrypterDecrypter_ReadWrite(t *testing.T) {
	t.Run("aes_128", func(t *testing.T) {
		require := require.New(t)
		keys := []string{
			"abcdefghijklmnop",
			"0123456789012345",
			randomString(16),
		}
		for _, key := range keys {
			ed, err := NewAESEncrypterDecrypter([]byte(key))
			require.NoError(err)
			require.NotNil(ed)
			testProcessorReadWrite(t, ed)
		}
	})
	t.Run("aes_192", func(t *testing.T) {
		require := require.New(t)
		keys := []string{
			"abcdefghijklmnopqrstuvwx",
			"012345678901234567890123",
			randomString(24),
		}
		for _, key := range keys {
			ed, err := NewAESEncrypterDecrypter([]byte(key))
			require.NoError(err)
			require.NotNil(ed)
			testProcessorReadWrite(t, ed)
		}
	})
	t.Run("aes_256", func(t *testing.T) {
		require := require.New(t)
		keys := []string{
			"abcdefghijklmnopqrstuvwxyzabcdef",
			"01234567890123467890123456789012",
			randomString(32),
		}
		for _, key := range keys {
			ed, err := NewAESEncrypterDecrypter([]byte(key))
			require.NoError(err)
			require.NotNil(ed)
			testProcessorReadWrite(t, ed)
		}
	})
}

func TestAESEncrypterDecrypter_ReadWrite_MultiLayer(t *testing.T) {
	t.Run("aes_128", func(t *testing.T) {
		require := require.New(t)
		keys := []string{
			"abcdefghijklmnop",
			"0123456789012345",
			randomString(16),
		}
		for _, key := range keys {
			ed, err := NewAESEncrypterDecrypter([]byte(key))
			require.NoError(err)
			require.NotNil(ed)
			testProcessorReadWriteMultiLayer(t, ed)
		}
	})
	t.Run("aes_192", func(t *testing.T) {
		require := require.New(t)
		keys := []string{
			"abcdefghijklmnopqrstuvwx",
			"012345678901234567890123",
			randomString(24),
		}
		for _, key := range keys {
			ed, err := NewAESEncrypterDecrypter([]byte(key))
			require.NoError(err)
			require.NotNil(ed)
			testProcessorReadWriteMultiLayer(t, ed)
		}
	})
	t.Run("aes_256", func(t *testing.T) {
		require := require.New(t)
		keys := []string{
			"abcdefghijklmnopqrstuvwxyzabcdef",
			"01234567890123467890123456789012",
			randomString(32),
		}
		for _, key := range keys {
			ed, err := NewAESEncrypterDecrypter([]byte(key))
			require.NoError(err)
			require.NotNil(ed)
			testProcessorReadWriteMultiLayer(t, ed)
		}
	})
}

func TestAESEncrypterDecrypter_ReadWrite_Async(t *testing.T) {
	t.Run("aes_128", func(t *testing.T) {
		require := require.New(t)
		keys := []string{
			"abcdefghijklmnop",
			"0123456789012345",
			randomString(16),
		}
		for _, key := range keys {
			c := func() Processor {
				ed, err := NewAESEncrypterDecrypter([]byte(key))
				require.NoError(err)
				require.NotNil(ed)
				return ed
			}
			testProcessorReadWriteAsync(t, c)
		}
	})
	t.Run("aes_192", func(t *testing.T) {
		require := require.New(t)
		keys := []string{
			"abcdefghijklmnopqrstuvwx",
			"012345678901234567890123",
			randomString(24),
		}
		for _, key := range keys {
			c := func() Processor {
				ed, err := NewAESEncrypterDecrypter([]byte(key))
				require.NoError(err)
				require.NotNil(ed)
				return ed
			}
			testProcessorReadWriteAsync(t, c)
		}
	})
	t.Run("aes_256", func(t *testing.T) {
		require := require.New(t)
		keys := []string{
			"abcdefghijklmnopqrstuvwxyzabcdef",
			"01234567890123467890123456789012",
			randomString(32),
		}
		for _, key := range keys {
			c := func() Processor {
				ed, err := NewAESEncrypterDecrypter([]byte(key))
				require.NoError(err)
				require.NotNil(ed)
				return ed
			}
			testProcessorReadWriteAsync(t, c)
		}
	})
}

// A simple test to ensure we can combine 2 AES Encrypter-Decrypter together,
// not that we're planning to do this, but it should work none the less
func TestAESEncryptionCombination(t *testing.T) {
	require := require.New(t)

	ed1, err := NewAESEncrypterDecrypter([]byte(randomString(16)))
	require.NoError(err)
	require.NotNil(ed1)

	ed2, err := NewAESEncrypterDecrypter([]byte(randomString(32)))
	require.NoError(err)
	require.NotNil(ed2)

	testCases := []string{
		"a",
		"foo",
		"Hello, World!",
		"大家好",
		"This... is my finger :)",
	}
	for _, testCase := range testCases {
		inputData := []byte(testCase)

		data, err := ed1.WriteProcess(inputData)
		require.NoError(err)
		require.NotEmpty(data)

		data, err = ed2.WriteProcess(data)
		require.NoError(err)
		require.NotEmpty(data)

		data, err = ed2.ReadProcess(data)
		require.NoError(err)
		require.NotEmpty(data)

		outputData, err := ed1.ReadProcess(data)
		require.NoError(err)
		require.NotEmpty(outputData)

		require.Equal(inputData, outputData)
	}
}

func TestAESEncryptionCombination_Async(t *testing.T) {
	require := require.New(t)

	ed1, err := NewAESEncrypterDecrypter([]byte(randomString(16)))
	require.NoError(err)
	require.NotNil(ed1)

	ed2, err := NewAESEncrypterDecrypter([]byte(randomString(32)))
	require.NoError(err)
	require.NotNil(ed2)

	testCases := []string{
		"a",
		"foo",
		"Hello, World!",
		"大家好",
		"This... is my finger :)",
	}
	for _, testCase := range testCases {
		inputData := []byte(testCase)

		data, err := ed1.WriteProcess(inputData)
		require.NoError(err)
		require.NotEmpty(data)

		data, err = ed2.WriteProcess(data)
		require.NoError(err)
		require.NotEmpty(data)

		data, err = ed2.ReadProcess(data)
		require.NoError(err)
		require.NotEmpty(data)

		outputData, err := ed1.ReadProcess(data)
		require.NoError(err)
		require.NotEmpty(outputData)

		require.Equal(inputData, outputData)
	}
}

func randomString(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return string(b)
}

// some tests to ensure a user can register its own encryption type,
// in a valid and logical way

func TestMyCustomEncrypterDecrypter(t *testing.T) {
	require := require.New(t)

	encrypterDecrypter, err := NewEncrypterDecrypter(myCustomEncryptionType, nil)
	require.NoError(err)
	require.NotNil(encrypterDecrypter)
	require.IsType(NopProcessor{}, encrypterDecrypter)

	data, err := encrypterDecrypter.ReadProcess(nil)
	require.NoError(err)
	require.Equal([]byte(nil), data)
	data, err = encrypterDecrypter.WriteProcess([]byte("foo"))
	require.NoError(err)
	require.Equal([]byte("foo"), data)
}

func TestRegisterEncryptionExplicitPanics(t *testing.T) {
	require := require.New(t)

	require.Panics(func() {
		RegisterEncrypterDecrypter(myCustomEncryptionType, myCustomEncryptionTypeStr, nil)
	}, "no constructor given")

	require.Panics(func() {
		RegisterEncrypterDecrypter(myCustomEncryptionTypeNumberTwo, "", newMyCustomEncrypterDecrypter)
	}, "no string version given for non-registered encrypter-decrypter")
}

func TestRegisterEncryptionIgnoreStringExistingEncryption(t *testing.T) {
	require := require.New(t)

	require.Equal(myCustomEncryptionTypeStr, myCustomEncryptionType.String())
	RegisterEncrypterDecrypter(myCustomEncryptionType, "foo", newMyCustomEncrypterDecrypter)
	require.Equal(myCustomEncryptionTypeStr, myCustomEncryptionType.String())

	// the given string to RegisterEncrypterDecrypter will force lower cases for all characters
	// as to make the string<->value mapping case insensitive
	RegisterEncrypterDecrypter(myCustomEncryptionTypeNumberTwo, "FOO", newMyCustomEncrypterDecrypter)
	require.Equal("foo", myCustomEncryptionTypeNumberTwo.String())
}

const (
	myCustomEncryptionType = iota + MaxStandardEncryptionType + 1
	myCustomEncryptionTypeNumberTwo

	myCustomEncryptionTypeStr = "bad_256"
)

func newMyCustomEncrypterDecrypter([]byte) (Processor, error) {
	return NopProcessor{}, nil
}

func init() {
	RegisterEncrypterDecrypter(
		myCustomEncryptionType, myCustomEncryptionTypeStr,
		newMyCustomEncrypterDecrypter)
}
