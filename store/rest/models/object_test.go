package models

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestObjectEncodeDecode(t *testing.T) {
	data := make([]byte, 0, 4096)
	_, err := rand.Read(data)
	require.NoError(t, err)

	obj := Object{
		Data: string(data),
		Id:   "Foooo",
		Tags: []Tag{{
			Key:   "key",
			Value: "value",
		}},
	}

	f, err := obj.ToFile("ns1")
	require.NoError(t, err)

	b, err := f.Encode()
	require.NoError(t, err)

	f2 := &File{}
	err = f2.Decode(b)
	require.NoError(t, err)

	assert.EqualValues(t, f, f2)
}
