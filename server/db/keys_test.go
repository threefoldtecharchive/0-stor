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

package db

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScopedSequenceKey(t *testing.T) {
	require := require.New(t)

	require.Panics(func() {
		ScopedSequenceKey(nil, 0)
	}, "should panic when no label is given")

	require.Equal("大家好"+string([]byte{0, 0, 0, 0, 0, 0, 0, 0}),
		s(ScopedSequenceKey(bs("大家好"), 0)))
	require.Equal("max-uint64"+string([]byte{255, 255, 255, 255, 255, 255, 255, 255}),
		s(ScopedSequenceKey(bs("max-uint64"), math.MaxUint64)))
	require.Equal("order-sequence-check"+string([]byte{1, 2, 3, 4, 5, 6, 7, 8}),
		s(ScopedSequenceKey(bs("order-sequence-check"), 578437695752307201)))
}

func TestDataKey(t *testing.T) {
	require := require.New(t)

	require.Panics(func() {
		DataKey(nil, bs("foo"))
	}, "should panic when no label is given")
	require.Panics(func() {
		DataKey(bs("foo"), nil)
	}, "should panic when no key is given")
	require.Panics(func() {
		DataKey(nil, nil)
	}, "should panic when nothing is given")

	require.Equal("d:foo:bar", s(DataKey(bs("foo"), bs("bar"))))
	require.Equal("d:42:大家好", s(DataKey(bs("42"), bs("大家好"))))
}

func TestDataScopeKey(t *testing.T) {
	require := require.New(t)

	require.Panics(func() {
		DataScopeKey(nil)
	}, "should panic when no label is given")

	require.Equal("d:foo:", s(DataScopeKey(bs("foo"))))
	require.Equal("d:大家好:", s(DataScopeKey(bs("大家好"))))
}

func TestNamespaceKey(t *testing.T) {
	require := require.New(t)

	require.Panics(func() {
		NamespaceKey(nil)
	}, "should panic when no label is given")

	require.Equal("@:foo", s(NamespaceKey(bs("foo"))))
	require.Equal("@:大家好", s(NamespaceKey(bs("大家好"))))
}

func TestUnlistedKey(t *testing.T) {
	require := require.New(t)

	require.Panics(func() {
		UnlistedKey(nil)
	}, "should panic when no key is given")

	require.Equal("__foo", s(UnlistedKey(bs("foo"))))
	require.Equal("__大家好", s(UnlistedKey(bs("大家好"))))
}

func s(bs []byte) string { return string(bs) }
func bs(s string) []byte { return []byte(s) }
