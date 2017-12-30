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

func TestEncryptionTypeMarshalUnmarshal(t *testing.T) {
	require := require.New(t)

	types := []EncryptionType{
		EncryptionTypeAES,
	}
	for _, t := range types {
		b, err := t.MarshalText()
		require.NoError(err)
		require.NotNil(b)

		var o EncryptionType
		err = o.UnmarshalText(b)
		require.NoError(err)
		require.Equal(t, o)
	}
}

func TestEncryptionTypeMarshal(t *testing.T) {
	require := require.New(t)

	testCases := []struct {
		Type     EncryptionType
		Expected string
	}{
		{EncryptionTypeAES, "aes"},
		{math.MaxUint8, "255"},
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

func TestEncryptionTypeUnmarshal(t *testing.T) {
	require := require.New(t)

	testCases := []struct {
		String   string
		Expected EncryptionType
		Err      bool
	}{
		{"aes", EncryptionTypeAES, false},
		{"AES", EncryptionTypeAES, false},
		{"", DefaultEncryptionType, false},
		{"some invalid type", math.MaxUint8, true},
	}
	for _, tc := range testCases {
		var o EncryptionType
		err := o.UnmarshalText([]byte(tc.String))
		if tc.Err {
			require.Error(err)
			require.Equal(DefaultEncryptionType, o)
		} else {
			require.NoError(err)
			require.Equal(tc.Expected, o)
		}
	}
}
