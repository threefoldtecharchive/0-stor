package db_test

import (
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	dbp "github.com/zero-os/0-stor/server/db"
	"github.com/zero-os/0-stor/server/db/badger"
	"github.com/zero-os/0-stor/server/db/memory"
)

func makeTestBadgerDB(t testing.TB) (dbp.DB, func()) {
	tmpDir, err := ioutil.TempDir("", "0-stor-test")
	require.NoError(t, err)

	db, err := badger.New(path.Join(tmpDir, "data"), path.Join(tmpDir, "meta"))
	require.NoError(t, err)
	cleanup := func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}
	return db, cleanup
}

func makeTestObj(t testing.TB, size uint64, db dbp.DB) (*dbp.Object, []byte, []string) {
	obj := dbp.NewObject("testns", []byte("key"), db)

	data := make([]byte, size)
	_, err := rand.Read(data)
	require.NoError(t, err)
	obj.SetData(data)

	refList := make([]string, dbp.RefIDCount)
	for i := 0; i < dbp.RefIDCount; i++ {
		refList[i] = fmt.Sprintf("user%d", i)
	}
	err = obj.SetReferenceList(refList)
	require.NoError(t, err, "fail to set reference list")

	return obj, data, refList
}

func TestObjectSaveLoad_Badger(t *testing.T) {
	db, cleanup := makeTestBadgerDB(t)
	defer cleanup()
	testObjectSaveLoad(t, db)
}

func TestObjectSaveLoad_Memory(t *testing.T) {
	db := memory.New()
	defer db.Close()
	testObjectSaveLoad(t, db)
}

func testObjectSaveLoad(t *testing.T, db dbp.DB) {
	obj, data, refList := makeTestObj(t, 1024*4, db)

	err := obj.Save()
	require.NoError(t, err)

	obj2 := dbp.NewObject("testns", []byte("key"), db)

	data2, err := obj2.Data()
	require.NoError(t, err, "fail to read data")
	refList2, err := obj.GetreferenceListStr()
	require.NoError(t, err, "fail to read reference list")

	assert.Equal(t, refList, refList2, "reference list differs")
	assert.Equal(t, data, data2, "data differes")
}

/*
func TestObjectValidateCRC_Badger(t *testing.T) {
	db, cleanup := makeTestBadgerDB(t)
	defer cleanup()
	testObjectValidateCRC(t, db)
}

func TestObjectValidateCRC_Memory(t *testing.T) {
	db := memory.New()
	defer db.Close()
	testObjectValidateCRC(t, db)
}
*/

/*
func testObjectValidateCRC(t *testing.T, db dbp.DB) {
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
}*/
