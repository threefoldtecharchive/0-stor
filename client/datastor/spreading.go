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

// Package datastor defines the clients and other types,
// to be used to interface with a zstordb server.
package datastor

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
)

// SpreadingType represent a spreading algorithm
// used by the ShardIterator.
type SpreadingType uint8

const (
	// SpreadingTypeRandom is the enum constant that identifies
	// a ShardIterator that will walk over the shards in a purely
	// random way.
	SpreadingTypeRandom SpreadingType = iota

	// SpreadingTypeLeastUsed is the enum constant that identifies
	// a ShardIterator that will walk over the shards by returning first
	// the shards that are the least used (have the more storage available)
	SpreadingTypeLeastUsed

	// DefaultSpreadingType represent the default value
	// for the ShardIterator
	//
	// This package reserves the right to change the
	// default value at any time,
	// but this constant will always be available and up to date.
	DefaultSpreadingType = SpreadingTypeRandom

	// MaxStandardSpreadingType defines the spreading type,
	// which has the greatest defined/used enum value.
	// When defining your custom SpreadingType you can do so as follows:
	//
	//    const (
	//         MySpreadingType = iota + datastor.MaxStandardSpreadingType + 1
	//         MyOtherSpreadingType
	//         // ...
	//    )
	//
	MaxStandardSpreadingType = SpreadingTypeLeastUsed
)

// String implements Stringer.String
func (ht SpreadingType) String() string {
	str, ok := _SpreadingTypeValueToStringMapping[ht]
	if !ok {
		return fmt.Sprint(uint8(ht))
	}
	return str
}

// MarshalText implements encoding.TextMarshaler.MarshalText
func (ht SpreadingType) MarshalText() ([]byte, error) {
	str := ht.String()
	if str == "" {
		return nil, fmt.Errorf("'%s' is not a valid SpreadingType value", ht)
	}
	return []byte(str), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.UnmarshalText
func (ht *SpreadingType) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		*ht = DefaultSpreadingType
		return nil
	}

	var ok bool
	*ht, ok = _SpreadingTypeStringToValueMapping[strings.ToLower(string(text))]
	if !ok {
		return fmt.Errorf("'%s' is not a valid SpreadingType string", text)
	}
	return nil
}

// SpreadingConstructor defines a function which can be used to create
// a ShardIterator
type SpreadingConstructor func() (ShardIterator, error)

// RegisterSpreadingType registers a new or overwrite an existing spreading algorithm.
// The given str will be used in a case-insensitive manner,
// if the registered hash however overwrites an existing spreading type,
// the str parameter will be ignored and instead the already used string version will be used.
// This is intended to be called from the init function in packages that implement spread functions.
func RegisterSpreadingType(ht SpreadingType, str string) {
	if s, ok := _SpreadingTypeValueToStringMapping[ht]; ok {
		log.Infof("overwriting HasherConstructor for hash type %s", ht)
		str = s // ignoring given string
	} else if str == "" {
		panic("no string version defined for new hash type")
	} else {
		// enforce lower cases
		// as to make the string<->value mapping case insensitive
		str = strings.ToLower(str)
	}

	_SpreadingTypeValueToStringMapping[ht] = str
	_SpreadingTypeStringToValueMapping[str] = ht
}

// Spreading algorithms mapping used to create SpreadingType instances,
// based on their enum value, as well as to
// convert between the string and enum type values.
var (
	_SpreadingTypeValueToStringMapping = make(map[SpreadingType]string)
	_SpreadingTypeStringToValueMapping = make(map[string]SpreadingType)
)
