package db

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestObjectEncodeDecode(t *testing.T) {
	obj := NewObjet()
	obj.Data = make([]byte, 1024)

	_, err := rand.Read(obj.Data)
	require.NoError(t, err)

	copy(obj.ReferenceList[0][:], []byte("user1"))
	copy(obj.ReferenceList[1][:], []byte("user2"))

	b, err := obj.Encode()
	require.NoError(t, err)

	obj2 := NewObjet()
	err = obj2.Decode(b)
	require.NoError(t, err)

	assert.Equal(t, obj.ReferenceList, obj2.ReferenceList, "reference list differs")
	assert.Equal(t, obj.Data, obj2.Data, "data differes")
}

func TestNamespaceEncodeDecode(t *testing.T) {
	ns := Namespace{
		Label:    "mynamespace",
		Reserved: 1024 * 1024,
	}

	b, err := ns.Encode()
	require.NoError(t, err)

	ns2 := Namespace{}
	err = ns2.Decode(b)
	require.NoError(t, err)

	assert.Equal(t, ns.Label, ns2.Label, "label differs")
	assert.Equal(t, ns.Reserved, ns2.Reserved, "reserved differs")

}
