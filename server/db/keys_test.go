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
