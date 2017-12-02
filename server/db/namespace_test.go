package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
