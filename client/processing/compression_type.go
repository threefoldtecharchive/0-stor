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
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
)

// CompressionType represents a compression algorithm.
type CompressionType uint8

const (
	// CompressionTypeSnappy is the enum constant which identifies Snappy,
	// a compression algorithm, implemented by Google, and also the compression algorithm,
	// promoted by this package and used by default.
	//
	// See github.com/golang/snappy for more information about the
	// technical details beyind this compression type.
	CompressionTypeSnappy CompressionType = iota
	// CompressionTypeLZ4 is the enum constant which identifies lz4,
	// a compression algorithm, implemented by Pierre Curto.
	//
	// See github.com/pierrec/lz4 for more information about the
	// technical details beyind this compression type.
	CompressionTypeLZ4
	// CompressionTypeGZip is the enum constant which identifies gz4,
	// a compression algorithm, part of the Golang std.
	//
	// See compress/gzip (Golang STD package) for more information about the
	// technical details beyind this compression type.
	CompressionTypeGZip

	// DefaultCompressionType represents the default
	// compression algorithm as promoted by this package.
	//
	// This package reserves the right to change the
	// default compression algorithm at any time,
	// but this constant will always be available and up to date.
	DefaultCompressionType = CompressionTypeSnappy

	// MaxStandardCompressionType defines the compression type,
	// which has the greatest defined/used enum value.
	// When defining your custom CompressionType you can do so as follows:
	//
	//    const (
	//         MyCompressionType = iota + processing.MaxStandardCompressionType + 1
	//         MyOtherCompressionType
	//         // ...
	//    )
	//
	// The maximum allowed value of a custom compression type is 255,
	// due to the underlying uint8 type.
	MaxStandardCompressionType = CompressionTypeGZip
)

// String implements Stringer.String
func (ct CompressionType) String() string {
	str, ok := _CompressionTypeValueToStringMapping[ct]
	if !ok {
		return fmt.Sprint(uint8(ct))
	}
	return str
}

// MarshalText implements encoding.TextMarshaler.MarshalText
func (ct CompressionType) MarshalText() ([]byte, error) {
	str := ct.String()
	if str == "" {
		return nil, fmt.Errorf("'%s' is not a valid CompressionType value", ct)
	}
	return []byte(str), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.UnmarshalText
func (ct *CompressionType) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		*ct = DefaultCompressionType
		return nil
	}

	var ok bool
	*ct, ok = _CompressionTypeStringToValueMapping[strings.ToLower(string(text))]
	if !ok {
		return fmt.Errorf("'%s' is not a valid CompressionType string", text)
	}
	return nil
}

// CompressorDecompressorConstructor defines a function which can be used to create an compressor-decompressor.
//
// The given compression mode serves as a hint from the user to the constructor,
// about what the user is expecting from its desired compressor-decompressor.
// The constructor is however free to interpret this value as it sees fit,
// however it is recommended to try to respect the choice as much as possible.
type CompressorDecompressorConstructor func(mode CompressionMode) (Processor, error)

// RegisterCompressorDecompressor registers a new or overwrite an existing compression algorithm.
// The given str will be used in a case-insensitive manner,
// if the registered compressor-decompressor however overwrites an existing compression type,
// the str parameter will be ignored and instead the already used string version will be used.
// This is intended to be called from the init function in packages that implement the compressor-decompressor.
func RegisterCompressorDecompressor(ct CompressionType, str string, cdc CompressorDecompressorConstructor) {
	if cdc == nil {
		panic("no compressor-decompressor constructor given")
	}

	if s, ok := _CompressionTypeValueToStringMapping[ct]; ok {
		log.Infof("overwriting CompressorDecompressorConstructor for compression type %s", ct)
		str = s // ignoring given string
	} else if str == "" {
		panic("no string version defined for new compression type")
	} else {
		// enforce lower cases
		// as to make the string<->value mapping case insensitive
		str = strings.ToLower(str)
	}

	_CompressionTypeValueToStringMapping[ct] = str
	_CompressionTypeStringToValueMapping[str] = ct
	_CompressionTypeValueToConstructorMapping[ct] = cdc
}

// compression algorithms mapping used to create compressor-decompressor instances,
// as processors, based on their enum value, as well as to
// convert between the string and enum type values.
var (
	_CompressionTypeValueToStringMapping      = make(map[CompressionType]string)
	_CompressionTypeStringToValueMapping      = make(map[string]CompressionType)
	_CompressionTypeValueToConstructorMapping = make(map[CompressionType]CompressorDecompressorConstructor)
)
