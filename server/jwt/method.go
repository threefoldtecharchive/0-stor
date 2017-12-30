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

package jwt

// Method represents the operation method
type Method uint8

// String implements Stringer.String
func (m Method) String() string {
	if str, ok := _MethodMap[m]; ok {
		return str
	}
	return ""
}

// Method enum
const (
	MethodWrite Method = 1 << iota
	MethodRead
	MethodDelete
	MethodAdmin
)

// contains all method strings
const _MethodStrings = "writereaddeleteadmin"

// slice mapping to the individual method strings
var _MethodMap = map[Method]string{
	MethodWrite:  _MethodStrings[0:5],
	MethodRead:   _MethodStrings[5:9],
	MethodDelete: _MethodStrings[9:15],
	MethodAdmin:  _MethodStrings[15:20],
}
