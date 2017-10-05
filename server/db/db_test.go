package db

import (
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zero-os/0-stor/server/db/badger"
)

func makeTestDB(t testing.TB) (DB, func()) {
	tmpDir, err := ioutil.TempDir("", "0stortest")
	require.NoError(t, err)

	db, err := badger.New(path.Join(tmpDir, "data"), path.Join(tmpDir, "meta"))
	require.NoError(t, err)
	cleanup := func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}
	return db, cleanup
}

func makeTestObj(t testing.TB, size uint64, db DB) (*Object, []byte, []string) {
	obj := NewObject("testns", []byte("key"), db)

	data := make([]byte, size)
	_, err := rand.Read(data)
	require.NoError(t, err)
	obj.SetData(data)

	refList := make([]string, len(obj.referenceList))
	for i := range obj.referenceList {
		refList[i] = fmt.Sprintf("user%d", i)
	}
	err = obj.SetReferenceList(refList)
	require.NoError(t, err, "fail to set reference list")

	return obj, data, refList
}

func TestObjectSaveLoad(t *testing.T) {
	db, cleanup := makeTestDB(t)
	defer cleanup()

	obj, data, refList := makeTestObj(t, 1024*4, db)

	err := obj.Save()
	require.NoError(t, err)

	obj2 := NewObject("testns", []byte("key"), db)

	data2, err := obj2.Data()
	require.NoError(t, err, "fail to read data")
	refList2, err := obj.GetreferenceListStr()
	require.NoError(t, err, "fail to read reference list")

	assert.Equal(t, refList, refList2, "reference list differs")
	assert.Equal(t, data, data2, "data differes")
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

func TestStoreStatEncodeDecode(t *testing.T) {
	stat := StoreStat{
		SizeAvailable: 100,
		SizeUsed:      45,
	}

	b, err := stat.Encode()
	require.NoError(t, err, "fail to encode StoreStat")

	stat2 := StoreStat{}
	err = stat2.Decode(b)
	require.NoError(t, err, "fail to decode StoreStat")

	assert.Equal(t, stat, stat2, "two object should be the same")
}

func TestObjectValidateCRC(t *testing.T) {
	db, cleanup := makeTestDB(t)
	defer cleanup()

	obj, _, _ := makeTestObj(t, 1024*4, db)

	valid, err := obj.Validcrc()
	require.NoError(t, err, "fail to validate crc")

	assert.True(t, valid, "CRC should be valid")
	for i := 0; i < 10; i++ {
		obj.data[i] = -obj.data[i] // corrupte the data
	}
	valid, err = obj.Validcrc()
	require.NoError(t, err, "fail to validate crc")
	assert.False(t, valid, "CRC should be different")
}
