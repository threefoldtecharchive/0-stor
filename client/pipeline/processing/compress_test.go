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
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSnappyCompressorDecompressor_ReadWrite(t *testing.T) {
	testCompressorDecompressorReadWrite(t,
		func(mode CompressionMode) (Processor, error) {
			return NewSnappyCompressorDecompressor(mode)
		})
}

func TestSnappyCompressorDecompressor_ReadWrite_MultiLayer(t *testing.T) {
	testCompressorDecompressorReadWriteMultiLayer(t,
		func(mode CompressionMode) (Processor, error) {
			return NewSnappyCompressorDecompressor(mode)
		})
}

func TestSnappyCompressorDecompressor_ReadWrite_Async(t *testing.T) {
	testCompressorDecompressorReadWriteAsync(t,
		func(mode CompressionMode) (Processor, error) {
			return NewSnappyCompressorDecompressor(mode)
		})
}

func TestLZ4CompressorDecompressor_ReadWrite(t *testing.T) {
	testCompressorDecompressorReadWrite(t,
		func(mode CompressionMode) (Processor, error) {
			return NewLZ4CompressorDecompressor(mode)
		})
}

func TestLZ4CompressorDecompressor_ReadWrite_MultiLayer(t *testing.T) {
	testCompressorDecompressorReadWriteMultiLayer(t,
		func(mode CompressionMode) (Processor, error) {
			return NewLZ4CompressorDecompressor(mode)
		})
}

func TestLZ4CompressorDecompressor_ReadWrite_Async(t *testing.T) {
	testCompressorDecompressorReadWriteAsync(t,
		func(mode CompressionMode) (Processor, error) {
			return NewLZ4CompressorDecompressor(mode)
		})
}

func TestGZipCompressorDecompressor_ReadWrite(t *testing.T) {
	testCompressorDecompressorReadWrite(t,
		func(mode CompressionMode) (Processor, error) {
			return NewGZipCompressorDecompressor(mode)
		})
}

func TestGZipCompressorDecompressor_ReadWrite_MultiLayer(t *testing.T) {
	testCompressorDecompressorReadWriteMultiLayer(t,
		func(mode CompressionMode) (Processor, error) {
			return NewGZipCompressorDecompressor(mode)
		})
}

func TestGZipCompressorDecompressor_ReadWrite_Async(t *testing.T) {
	testCompressorDecompressorReadWriteAsync(t,
		func(mode CompressionMode) (Processor, error) {
			return NewGZipCompressorDecompressor(mode)
		})
}

func testCompressorDecompressorReadWrite(t *testing.T, c CompressorDecompressorConstructor) {
	modes := []CompressionMode{
		CompressionModeDefault,
		CompressionModeBestSpeed,
		CompressionModeBestCompression,
		CompressionMode(math.MaxUint8),
	}
	for _, mode := range modes {
		t.Run(mode.String(), func(t *testing.T) {
			compressorDecompressor, err := c(mode)
			require.NoError(t, err)
			require.NotNil(t, compressorDecompressor)
			testProcessorReadWrite(t, compressorDecompressor)
		})
	}
}

func testCompressorDecompressorReadWriteMultiLayer(t *testing.T, c CompressorDecompressorConstructor) {
	modes := []CompressionMode{
		CompressionModeDefault,
		CompressionModeBestSpeed,
		CompressionModeBestCompression,
		CompressionMode(math.MaxUint8),
	}
	for _, mode := range modes {
		t.Run(mode.String(), func(t *testing.T) {
			compressorDecompressor, err := c(mode)
			require.NoError(t, err)
			require.NotNil(t, compressorDecompressor)
			testProcessorReadWriteMultiLayer(t, compressorDecompressor)
		})
	}
}

func testCompressorDecompressorReadWriteAsync(t *testing.T, c CompressorDecompressorConstructor) {
	modes := []CompressionMode{
		CompressionModeDefault,
		CompressionModeBestSpeed,
		CompressionModeBestCompression,
		CompressionMode(math.MaxUint8),
	}
	for _, mode := range modes {
		t.Run(mode.String(), func(t *testing.T) {
			testProcessorReadWriteAsync(t, func() Processor {
				compressorDecompressor, err := c(mode)
				require.NoError(t, err)
				require.NotNil(t, compressorDecompressor)
				return compressorDecompressor
			})
		})
	}
}

// some tests to ensure a user can register its own compression type,
// in a valid and logical way

func TestMyCustomCompressorDecompressor(t *testing.T) {
	require := require.New(t)

	compressorDecompressor, err := NewCompressorDecompressor(
		myCustomCompressionType, CompressionModeDefault)
	require.NoError(err)
	require.NotNil(compressorDecompressor)
	require.IsType(NopProcessor{}, compressorDecompressor)

	data, err := compressorDecompressor.ReadProcess(nil)
	require.NoError(err)
	require.Equal([]byte(nil), data)
	data, err = compressorDecompressor.WriteProcess([]byte("foo"))
	require.NoError(err)
	require.Equal([]byte("foo"), data)
}

func TestRegisterCompressionExplicitPanics(t *testing.T) {
	require := require.New(t)

	require.Panics(func() {
		RegisterCompressorDecompressor(myCustomCompressionType, myCustomCompressionTypeStr, nil)
	}, "no constructor given")

	require.Panics(func() {
		RegisterCompressorDecompressor(myCustomCompressionTypeNumberTwo, "", newMyCustomCompressorDecompressor)
	}, "no string version given for non-registered compressor-decompressor")
}

func TestRegisterCompressionIgnoreStringExistingCompression(t *testing.T) {
	require := require.New(t)

	require.Equal(myCustomCompressionTypeStr, myCustomCompressionType.String())
	RegisterCompressorDecompressor(myCustomCompressionType, "foo", newMyCustomCompressorDecompressor)
	require.Equal(myCustomCompressionTypeStr, myCustomCompressionType.String())

	// the given string to RegisterCompressorDecompressor will force lower cases for all characters
	// as to make the string<->value mapping case insensitive
	RegisterCompressorDecompressor(myCustomCompressionTypeNumberTwo, "FOO", newMyCustomCompressorDecompressor)
	require.Equal("foo", myCustomCompressionTypeNumberTwo.String())
}

const (
	myCustomCompressionType = iota + MaxStandardCompressionType + 1
	myCustomCompressionTypeNumberTwo

	myCustomCompressionTypeStr = "bad_256"
)

func newMyCustomCompressorDecompressor(CompressionMode) (Processor, error) {
	return NopProcessor{}, nil
}

func init() {
	RegisterCompressorDecompressor(
		myCustomCompressionType, myCustomCompressionTypeStr,
		newMyCustomCompressorDecompressor)
}
