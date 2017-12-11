package db

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDataPrefix(t *testing.T) {
	require := require.New(t)

	require.Panics(func() {
		DataPrefix(nil)
	}, "should panic when no label is given")

	require.Equal("foo:d", s(DataPrefix(bs("foo"))))
	require.Equal("大家好:d", s(DataPrefix(bs("大家好"))))
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
	}, "should panic when no label and key is given")

	require.Equal("foo:d:bar", s(DataKey(bs("foo"), bs("bar"))))
	require.Equal("42:d:大家好", s(DataKey(bs("42"), bs("大家好"))))
}

func TestDataKeyPrefixLength(t *testing.T) {
	require := require.New(t)

	require.Panics(func() {
		DataKeyPrefixLength(nil)
	}, "should panic when no label is given")

	require.Equal(6, DataKeyPrefixLength(bs("foo")))
	require.Equal(12, DataKeyPrefixLength(bs("大家好")))
}

func TestReferenceListPrefix(t *testing.T) {
	require := require.New(t)

	require.Panics(func() {
		ReferenceListPrefix(nil)
	}, "should panic when no label is given")

	require.Equal("foo:rl", s(ReferenceListPrefix(bs("foo"))))
	require.Equal("大家好:rl", s(ReferenceListPrefix(bs("大家好"))))
}

func TestReferenceListKey(t *testing.T) {
	require := require.New(t)

	require.Panics(func() {
		ReferenceListKey(nil, bs("foo"))
	}, "should panic when no label is given")
	require.Panics(func() {
		ReferenceListKey(bs("foo"), nil)
	}, "should panic when no key is given")
	require.Panics(func() {
		ReferenceListKey(nil, nil)
	}, "should panic when no label and key is given")

	require.Equal("foo:rl:bar", s(ReferenceListKey(bs("foo"), bs("bar"))))
	require.Equal("42:rl:大家好", s(ReferenceListKey(bs("42"), bs("大家好"))))
}

func TestReferenceListKeyPrefixLength(t *testing.T) {
	require := require.New(t)

	require.Panics(func() {
		ReferenceListKeyPrefixLength(nil)
	}, "should panic when no label is given")

	require.Equal(7, ReferenceListKeyPrefixLength(bs("foo")))
	require.Equal(13, ReferenceListKeyPrefixLength(bs("大家好")))
}

func TestNamespaceKey(t *testing.T) {
	require := require.New(t)

	require.Panics(func() {
		NamespaceKey(nil)
	}, "should panic when no label is given")

	require.Equal("@:foo", s(NamespaceKey(bs("foo"))))
	require.Equal("@:大家好", s(NamespaceKey(bs("大家好"))))
}

func s(bs []byte) string { return string(bs) }
func bs(s string) []byte { return []byte(s) }
