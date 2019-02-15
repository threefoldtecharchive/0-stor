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

package datastor

import (
	"errors"
)

// Errors that get returned in case the server returns
// an unexpected results. The client can return these errors,
// but if any of these errors get returned,
// it means there is a bug in the zstordb code,
// and it should be reported at:
// http://github.com/threefoldtech/0-stor/issues
var (
	ErrMissingKey    = errors.New("zstor: missing object key (zstordb bug?)")
	ErrMissingData   = errors.New("zstor: missing object data (zstordb bug?)")
	ErrInvalidStatus = errors.New("zstor: invalid object status (zstordb bug?)")
	ErrInvalidLabel  = errors.New("zstor: invalid namespace label (zstordb bug?)")
)

// Errors that can be expected to be returned by a zstordb server,
// in "normal" scenarios.
var (
	ErrKeyNotFound     = errors.New("zstordb: key is no found")
	ErrObjectCorrupted = errors.New("zstordb: object is corrupted")
	ErrNamespaceFull   = errors.New("zstordb: namespace if full")
)

type (
	// Namespace contains information about a namespace.
	// None of this information is directly stored somewhere,
	// and instead it is gathered upon request.
	Namespace struct {
		Label               string
		ReadRequestPerHour  int64
		WriteRequestPerHour int64
		NrObjects           int64
	}

	// Object contains the information stored for an object.
	Object struct {
		Key  []byte
		Data []byte
	}

	// ObjectKeyResult is the (stream) data type,
	// used as the result data type, when fetching the keys
	// of all objects stored in the current namespace.
	//
	// Only in case of an error, the Error property will be set,
	// in all other cases only the Key property will be set.
	ObjectKeyResult struct {
		Key   []byte
		Error error
	}
)

// ObjectStatus defines the status of an object,
// it can be retrieved using the Check Method of the Client API.
type ObjectStatus uint8

// ObjectStatus enumeration values.
const (
	// The Object is missing.
	ObjectStatusMissing ObjectStatus = iota
	// The Object is OK.
	ObjectStatusOK
	// The Object is corrupted.
	ObjectStatusCorrupted
)

// String implements Stringer.String
func (status ObjectStatus) String() string {
	return _ObjectStatusEnumToStringMapping[status]
}

// private constants for the string

const _ObjectStatusStrings = "missingokcorrupted"

var _ObjectStatusEnumToStringMapping = map[ObjectStatus]string{
	ObjectStatusMissing:   _ObjectStatusStrings[:7],
	ObjectStatusOK:        _ObjectStatusStrings[7:9],
	ObjectStatusCorrupted: _ObjectStatusStrings[9:],
}
