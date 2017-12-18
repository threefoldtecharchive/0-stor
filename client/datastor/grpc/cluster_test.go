package grpc

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewClusterExplicitErrors(t *testing.T) {
	require := require.New(t)

	cluster, err := NewCluster(nil, "foo", nil)
	require.Error(err, "no addresses given")
	require.Nil(cluster)

	cluster, err = NewCluster([]string{"foo"}, "", nil)
	require.Error(err, "no label given")
	require.Nil(cluster)

	cluster, err = NewCluster(nil, "", nil)
	require.Error(err, "no addresses given, nor a label given")
	require.Nil(cluster)
}
