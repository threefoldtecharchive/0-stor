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
	"fmt"
	"strings"

	"github.com/threefoldtech/0-stor/client/metastor/encoding/proto"
	"github.com/threefoldtech/0-stor/client/metastor/metatypes"
)

// MarshalMetadata returns the encoding of the metadata parameter.
// It is important to use this function with a matching `UnmarshalMetadata` function.
type MarshalMetadata func(md metatypes.Metadata) ([]byte, error)

// UnmarshalMetadata parses the encoded metadata
// and stores the result in the value pointed to by the data parameter.
// It is important to use this function with a matching `MashalMetadata` function.
type UnmarshalMetadata func(b []byte, md *metatypes.Metadata) error

// NewMarshalFuncPair returns a registered MarshalFuncPair,
// for a given MarshalType.
//
// An error is returned in case, and only if,
// there was no MarshalFuncPair registered for the given MarshalType.
func NewMarshalFuncPair(mt MarshalType) (*MarshalFuncPair, error) {
	pair, ok := _MarshalTypeValueToFuncPairMapping[mt]
	if !ok {
		return nil, fmt.Errorf("%d is not a valid MarshalType value", mt)
	}
	return &pair, nil
}

// MarshalFuncPair composes a pair of MarshalMetadata and UnmarshalMetadata
// functions which are meant to be used together.
type MarshalFuncPair struct {
	Marshal   MarshalMetadata
	Unmarshal UnmarshalMetadata
}

// MarshalType represents the type of
// a pair of Marshal/Unmarshal Metadata functions.
type MarshalType uint8

const (
	// MarshalTypeProtobuf is the enum value which identifies,
	// the protobuf (un)marshalling functions pair provided by the proto subpackage.
	//
	// It makes use of the (slick) gogoprotobuf generator,
	// to generate the code used to marshal/unmarshal
	// metadata using this (un)marshalling pair.
	//
	// It is also the default Marshal type and the one recommended to be used,
	// as it is fast, lightweight and produces a compact output to top it all off.
	MarshalTypeProtobuf MarshalType = iota

	// DefaultMarshalType represents the default (un)marshalling format,
	// promoted by this package. Currently this is using Protobuf.
	//
	// This package reserves the right to change the
	// underlying algorithm at any time,
	// but this constant will always be available and up to date.
	DefaultMarshalType = MarshalTypeProtobuf

	// MaxStandardMarshalType defines the (un)marshalling type,
	// which has the greatest defined/used enum value.
	// When defining your custom MarshalType you can do so as follows:
	//
	//    const (
	//         MyMarshalType = iota + MaxStandardMarshalType + 1
	//         MyOtherMarshalType
	//         // ...
	//    )
	//
	// Or when you only have one custom Marshal Type you can simply do:
	//
	//    const MyMarshalType = MaxStandardMarshalType + 1
	//
	// Just make sure that you don't use the same value for 2 different
	// marshal type constants, as that would mean you'll overwrite
	// a previous registered hash when registering a (un)marshalling pair
	// using this "duplicate" constant.
	//
	// The maximum allowed value of a custom hash type is 255,
	// due to the underlying uint8 type.
	MaxStandardMarshalType = MarshalTypeProtobuf
)

// String implements Stringer.String
func (mt MarshalType) String() string {
	str, ok := _MarshalTypeValueToStringMapping[mt]
	if !ok {
		return ""
	}
	return str
}

// MarshalText implements encoding.TextMarshaler.MarshalText
func (mt MarshalType) MarshalText() ([]byte, error) {
	str := mt.String()
	if str == "" {
		return nil, fmt.Errorf("%d is not a valid MarshalType value", mt)
	}
	return []byte(str), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.UnmarshalText
func (mt *MarshalType) UnmarshalText(text []byte) error {
	var ok bool
	*mt, ok = _MarshalTypeStringToValueMapping[strings.ToLower(string(text))]
	if !ok {
		return fmt.Errorf("%q is not a valid MarshalType string", text)
	}
	return nil
}

// RegisterMarshalFuncPair registers a new or overwrite an existing
// marshal/unmarshal pair of functions, used to (un)marshal metadata.
// The given str will be used in a case-insensitive manner,
// if the registered marshal type however overwrites an existing marshal type,
// the str parameter will be ignored and instead the already used string version will be used.
// This is intended to be called from the init function in packages that implement marshal functions.
func RegisterMarshalFuncPair(mt MarshalType, str string, pair MarshalFuncPair) {
	if pair.Marshal == nil {
		panic("no marshal function given")
	}
	if pair.Unmarshal == nil {
		panic("no unmarshal function given")
	}

	if s, ok := _MarshalTypeValueToStringMapping[mt]; ok {
		str = s // ignoring given string
	} else if str == "" {
		panic("no string version defined for new MarshalType")
	}

	_MarshalTypeValueToStringMapping[mt] = str
	_MarshalTypeStringToValueMapping[str] = mt
	_MarshalTypeValueToFuncPairMapping[mt] = pair
}

// MarshalMetadata pairs mappings used to create MarshalMetadata pairs,
// based on their enum value, as well as to
// convert between the string and enum type values.
var (
	_MarshalTypeValueToStringMapping   = make(map[MarshalType]string)
	_MarshalTypeStringToValueMapping   = make(map[string]MarshalType)
	_MarshalTypeValueToFuncPairMapping = make(map[MarshalType]MarshalFuncPair)
)

func init() {
	RegisterMarshalFuncPair(
		MarshalTypeProtobuf, "protobuf", MarshalFuncPair{
			Marshal:   proto.MarshalMetadata,
			Unmarshal: proto.UnmarshalMetadata,
		})
}
