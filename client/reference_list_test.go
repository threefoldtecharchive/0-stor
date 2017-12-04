package client

import (
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/zero-os/0-stor/client/meta/embedserver"

	"github.com/stretchr/testify/require"
)

func TestReferenceList(t *testing.T) {
	etcd, err := embedserver.New()

	require.NoError(t, err, "fail to start embebed etcd server")
	defer etcd.Stop()

	servers, serverClean := testGRPCServer(t, 1)
	defer serverClean()

	shards := make([]string, len(servers))
	for i, server := range servers {
		shards[i] = server.Address()
	}

	policy := Policy{
		Organization:   "testorg",
		Namespace:      "namespace1",
		DataShards:     shards,
		MetaShards:     []string{etcd.ListenAddr()},
		IYOAppID:       "id",
		IYOSecret:      "secret",
		ReplicationNr:  0,
		DistributionNr: 0,
		BlockSize:      4096,
	}

	c, err := getTestClient(policy)
	require.NoError(t, err, "fail to create client")

	// initialize test data
	key := []byte("testkey")
	data := make([]byte, 1024*4)
	_, err = rand.Read(data)
	require.NoError(t, err, "fail to read random data")

	// initialize ref list

	allRefList := make([]string, 0, 160)
	for i := 0; i < 160; i++ {
		allRefList = append(allRefList, fmt.Sprintf("vdisk-%v", i))
	}

	// write data to the store with proper number of refList
	refList := allRefList[:]
	_, err = c.Write(key, data, refList)
	require.NoError(t, err, "fail to write data to the store")

	// check
	_, refListRead, err := c.Read(key)
	require.Equal(t, refList, refListRead)

	// remove reference list
	removeIndex := 160 / 2
	err = c.RemoveReferenceList(key, allRefList[removeIndex:])
	require.NoError(t, err)

	_, refListRead, err = c.Read(key)
	require.NoError(t, err)
	require.Len(t, refListRead, len(allRefList[:removeIndex]))
	require.Subset(t, allRefList[:removeIndex], refListRead)

	// append some of it
	appendIndex := removeIndex + (removeIndex / 2)
	err = c.AppendReferenceList(key, allRefList[removeIndex:appendIndex])
	require.NoError(t, err)

	_, refListRead, err = c.Read(key)
	require.NoError(t, err)
	require.Len(t, refListRead, len(allRefList[:appendIndex]))
	require.Subset(t, allRefList[:appendIndex], refListRead)

	// set again to full value
	err = c.SetReferenceList(key, allRefList[:160])
	require.NoError(t, err)

	_, refListRead, err = c.Read(key)
	require.NoError(t, err)
	require.Len(t, refListRead, len(allRefList[:160]))
	require.Subset(t, allRefList[:160], refListRead)
}
