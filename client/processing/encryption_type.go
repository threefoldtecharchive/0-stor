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

// EncryptionType represents an encryption algorithm.
type EncryptionType uint8

const (
	// EncryptionTypeAES is the enum constant which identifies AES,
	// an encryption algorithm, and also the encryption algorithm,
	// promoted by this package and used by default.
	//
	// What specific AES algorithm is used, depends on the given key size:
	//
	//    * 16 bytes: AES_128
	//    * 24 bytes: AES_192
	//    * 32 bytes: AES_256
	//
	// When giving a key of a size other than these 3,
	// while creating an encrypter-decrypter, the constructor will return an error.
	EncryptionTypeAES EncryptionType = iota

	// DefaultEncryptionType represents the default
	// encryption algorithm as promoted by this package.
	//
	// This package reserves the right to change the
	// default encryption algorithm at any time,
	// but this constant will always be available and up to date.
	DefaultEncryptionType = EncryptionTypeAES

	// MaxStandardEncryptionType defines the encryption type,
	// which has the greatest defined/used enum value.
	// When defining your custom EncryptionType you can do so as follows:
	//
	//    const (
	//         MyEncryptionType = iota + processing.MaxStandardEncryptionType + 1
	//         MyOtherEncryptionType
	//         // ...
	//    )
	//
	// The maximum allowed value of a custom encryption type is 255,
	// due to the underlying uint8 type.
	MaxStandardEncryptionType = EncryptionTypeAES
)

// String implements Stringer.String
func (et EncryptionType) String() string {
	str, ok := _EncryptionTypeValueToStringMapping[et]
	if !ok {
		return fmt.Sprint(uint8(et))
	}
	return str
}

// MarshalText implements encoding.TextMarshaler.MarshalText
func (et EncryptionType) MarshalText() ([]byte, error) {
	str := et.String()
	if str == "" {
		return nil, fmt.Errorf("'%s' is not a valid EncryptionType value", et)
	}
	return []byte(str), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.UnmarshalText
func (et *EncryptionType) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		*et = DefaultEncryptionType
		return nil
	}

	var ok bool
	*et, ok = _EncryptionTypeStringToValueMapping[strings.ToLower(string(text))]
	if !ok {
		return fmt.Errorf("'%s' is not a valid EncryptionType string", text)
	}
	return nil
}

// EncrypterDecrypterConstructor defines a function which can be used to create an encrypter-decrypter.
// The key parameter is a required private key which is used to encrypt and decrypt
// the data while processing it.
type EncrypterDecrypterConstructor func(key []byte) (Processor, error)

// RegisterEncrypterDecrypter registers a new or overwrite an existing encryption algorithm.
// The given str will be used in a case-insensitive manner,
// if the registered encrypter-decrypter however overwrites an existing encryption type,
// the str parameter will be ignored and instead the already used string version will be used.
// This is intended to be called from the init function in packages that implement the encrypter-decrypter.
func RegisterEncrypterDecrypter(et EncryptionType, str string, edc EncrypterDecrypterConstructor) {
	if edc == nil {
		panic("no encrypter-decrypter constructor given")
	}

	if s, ok := _EncryptionTypeValueToStringMapping[et]; ok {
		log.Infof("overwriting EncrypterDecrypterConstructor for encryption type %s", et)
		str = s // ignoring given string
	} else if str == "" {
		panic("no string version defined for new encryption type")
	} else {
		// enforce lower cases
		// as to make the string<->value mapping case insensitive
		str = strings.ToLower(str)
	}

	_EncryptionTypeValueToStringMapping[et] = str
	_EncryptionTypeStringToValueMapping[str] = et
	_EncryptionTypeValueToConstructorMapping[et] = edc
}

// encryption algorithms mapping used to create encrypter-decrypter instances,
// as processors, based on their enum value, as well as to
// convert between the string and enum type values.
var (
	_EncryptionTypeValueToStringMapping      = make(map[EncryptionType]string)
	_EncryptionTypeStringToValueMapping      = make(map[string]EncryptionType)
	_EncryptionTypeValueToConstructorMapping = make(map[EncryptionType]EncrypterDecrypterConstructor)
)
