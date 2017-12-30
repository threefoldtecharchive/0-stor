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
	"errors"
	"math"
	"testing"

	"github.com/zero-os/0-stor/client/metastor"
	"github.com/zero-os/0-stor/client/metastor/encoding/proto"

	"github.com/stretchr/testify/require"
)

func TestMarshalTypeMarshalUnmarshal(t *testing.T) {
	require := require.New(t)

	types := []MarshalType{
		MarshalTypeProtobuf,
	}
	for _, t := range types {
		b, err := t.MarshalText()
		require.NoError(err)
		require.NotNil(b)

		var o MarshalType
		err = o.UnmarshalText(b)
		require.NoError(err)
		require.Equal(t, o)
	}
}

func TestMarshalTypeTextMarshal(t *testing.T) {
	require := require.New(t)

	testCases := []struct {
		Type     MarshalType
		Expected string
	}{
		{MarshalTypeProtobuf, "protobuf"},
		{math.MaxUint8, ""},
	}
	for _, tc := range testCases {
		b, err := tc.Type.MarshalText()
		if tc.Expected == "" {
			require.Error(err)
			require.Nil(b)
		} else {
			require.NoError(err)
			require.Equal(tc.Expected, string(b))
		}
	}
}

func TestMarshalTypeTextUnmarshal(t *testing.T) {
	require := require.New(t)

	testCases := []struct {
		String   string
		Expected MarshalType
		Err      bool
	}{
		{"protobuf", MarshalTypeProtobuf, false},
		{"ProtoBuf", MarshalTypeProtobuf, false},
		{"PROTOBUF", MarshalTypeProtobuf, false},
		{"", math.MaxUint8, true},
	}
	for _, tc := range testCases {
		var o MarshalType
		err := o.UnmarshalText([]byte(tc.String))
		if tc.Err {
			require.Error(err)
			require.Equal(MarshalTypeProtobuf, o)
		} else {
			require.NoError(err)
			require.Equal(tc.Expected, o)
		}
	}
}

func TestNewHasher(t *testing.T) {
	require := require.New(t)

	testCases := []struct {
		Type     MarshalType
		Expected MarshalFuncPair
	}{
		{MarshalTypeProtobuf, MarshalFuncPair{proto.MarshalMetadata, proto.UnmarshalMetadata}},
		{math.MaxUint8, MarshalFuncPair{}},
	}

	for _, tc := range testCases {
		pair, err := NewMarshalFuncPair(tc.Type)
		if tc.Expected.Marshal == nil {
			require.Error(err)
			require.Nil(pair.Marshal)
			require.Nil(pair.Unmarshal)
		} else {
			require.NoError(err)
			require.NotNil(pair.Marshal)
			require.NotNil(pair.Unmarshal)
		}
	}
}

// some tests to ensure a user can register its own marshal func pair,
// without overwriting the existing (un)marshal algorithms

func TestMyCustomMarshalFuncPair(t *testing.T) {
	require := require.New(t)

	pair, err := NewMarshalFuncPair(myCustomMarshalType)
	require.NoError(err)
	require.NotNil(pair.Marshal)
	_, err = pair.Marshal(metastor.Metadata{})
	require.Equal(errMyMarshal, err)
	require.NotNil(pair.Unmarshal)
	err = pair.Unmarshal(nil, nil)
	require.Equal(errMyMarshal, err)
}

func TestRegiserMarshalFuncPairExplicitPanics(t *testing.T) {
	require := require.New(t)

	require.Panics(func() {
		RegisterMarshalFuncPair(myCustomMarshalType, myCustomMarshalTypeStr,
			MarshalFuncPair{nil, myErrorUnmarshalFunc})
	}, "no marshaler given")
	require.Panics(func() {
		RegisterMarshalFuncPair(myCustomMarshalType, myCustomMarshalTypeStr,
			MarshalFuncPair{myErrorMarsalFunc, nil})
	}, "no unmarshaler given")

	require.Panics(func() {
		RegisterMarshalFuncPair(myCustomMarshalTypeNumberTwo, "",
			MarshalFuncPair{myErrorMarsalFunc, myErrorUnmarshalFunc})
	}, "no string version given for non-registered MarshalFuncPair")
}

func TestRegisterHashIgnoreStringExistingHash(t *testing.T) {
	require := require.New(t)

	require.Equal(myCustomMarshalTypeStr, myCustomMarshalType.String())
	RegisterMarshalFuncPair(myCustomMarshalType, "foo",
		MarshalFuncPair{myErrorMarsalFunc, myErrorUnmarshalFunc})
	require.Equal(myCustomMarshalTypeStr, myCustomMarshalType.String())
}

const (
	myCustomMarshalType = iota + MaxStandardMarshalType + 1
	myCustomMarshalTypeNumberTwo

	myCustomMarshalTypeStr = "err_marshal"
)

var errMyMarshal = errors.New("my marshal error")

func myErrorMarsalFunc(metastor.Metadata) ([]byte, error) {
	return nil, errMyMarshal
}

func myErrorUnmarshalFunc([]byte, *metastor.Metadata) error {
	return errMyMarshal
}

func init() {
	RegisterMarshalFuncPair(myCustomMarshalType, myCustomMarshalTypeStr,
		MarshalFuncPair{myErrorMarsalFunc, myErrorUnmarshalFunc})
}
