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
	"github.com/zero-os/0-stor/server/db/memory"
)

func makeTestBadgerDB(t testing.TB) (DB, func()) {
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

func TestDBReuseOfInputValueSlice_Badger(t *testing.T) {
	db, cleanup := makeTestBadgerDB(t)
	defer cleanup()
	testDBReuseOfInputValueSlice(t, db)
}

func TestDBReuseOfInputValueSlice_Memory(t *testing.T) {
	db := memory.New()
	defer db.Close()
	testDBReuseOfInputValueSlice(t, db)
}

func testDBReuseOfInputValueSlice(t *testing.T, db DB) {
	require := require.New(t)

	// set 2 different values, using the same input buffer
	value := []byte("ping")
	require.NoError(db.Set([]byte{0}, value))
	value[1] = 'o'
	require.NoError(db.Set([]byte{1}, value))

	// now ensure that the 2 different values are indeed different
	value, err := db.Get([]byte{0})
	require.NoError(err)
	require.Equal("ping", string(value))
	value, err = db.Get([]byte{1})
	require.NoError(err)
	require.Equal("pong", string(value))

	// now set a lot more values (using the shared input value)
	for i := 0; i < 255; i++ {
		value[1] = byte(i)
		require.NoError(db.Set(value[1:2], value))
	}
	// now ensure all these values are still correctly defined
	for i := 0; i < 255; i++ {
		value, err = db.Get([]byte{byte(i)})
		require.NoError(err)
		expected := []byte("ping")
		expected[1] = byte(i)
		require.Equal(expected, value)
	}
}

func TestDBModificationOfOutput_Badger(t *testing.T) {
	db, cleanup := makeTestBadgerDB(t)
	defer cleanup()
	testDBModificationOfOutput(t, db)
}

func TestDBModificationOfOutput_Memory(t *testing.T) {
	db := memory.New()
	defer db.Close()
	testDBModificationOfOutput(t, db)
}

func testDBModificationOfOutput(t *testing.T, db DB) {
	require := require.New(t)

	// test that values retrieved aren't affected in the database
	value := []byte("ping")
	require.NoError(db.Set([]byte{0}, value))
	// ensure that that any value returned can be freely modified by the user
	output, err := db.Get([]byte{0})
	require.NoError(err)
	require.Equal("ping", string(output))
	output[1] = 'o'
	output, err = db.Get([]byte{0})
	require.NoError(err)
	require.Equal("ping", string(output))

	value[1] = 'o'
	require.NoError(db.Set([]byte{1}, value))
	// test that keys listed can be modified without consequences
	keys, err := db.List(nil)
	require.NoError(err)
	require.Len(keys, 2)
	require.Subset(keys, [][]byte{[]byte{0}, []byte{1}})
	// modify output keys
	keys[0][0] = 42
	keys[1][0] = 92
	// ensure that keys are still untouched in the database itself
	keys, err = db.List(nil)
	require.NoError(err)
	require.Len(keys, 2)
	require.Subset(keys, [][]byte{[]byte{0}, []byte{1}})

	// test that filtered values can be modified without consequences
	values, err := db.Filter(nil, 0, -1)
	require.NoError(err)
	require.Len(values, 2)
	require.Subset(values, [][]byte{[]byte("ping"), []byte("pong")})
	// modify output values
	values[0][0] = 'b'
	values[1][0] = 'b'
	// ensure that values are still untouched in the database itself
	values, err = db.Filter(nil, 0, -1)
	require.NoError(err)
	require.Len(values, 2)
	require.Subset(values, [][]byte{[]byte("ping"), []byte("pong")})
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

func testObjectSaveLoad(t *testing.T, db DB) {
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

func testObjectValidateCRC(t *testing.T, db DB) {
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

func TestDBFilter_Badger(t *testing.T) {
	db, cleanup := makeTestBadgerDB(t)
	defer cleanup()
	testDBFilter(t, db)
}

func TestDBFilter_Memory(t *testing.T) {
	db := memory.New()
	defer db.Close()
	testDBFilter(t, db)
}

func testDBFilter(t *testing.T, db DB) {
	require := require.New(t)

	values, err := db.Filter([]byte("foo"), 0, -1)
	require.NoError(err)
	require.Empty(values)

	err = db.Set([]byte("bar_one"), []byte("one"))
	require.NoError(err)

	values, err = db.Filter([]byte("foo"), 0, -1)
	require.NoError(err)
	require.Empty(values)

	values, err = db.Filter([]byte("bar"), 0, -1)
	require.NoError(err)
	require.Len(values, 1)
	require.Subset(values, [][]byte{[]byte("one")})

	err = db.Set([]byte("foo_one"), []byte("one"))
	require.NoError(err)

	values, err = db.Filter([]byte("foo"), 0, -1)
	require.NoError(err)
	require.Len(values, 1)
	require.Subset(values, [][]byte{[]byte("one")})

	err = db.Set([]byte("foo_two"), []byte("two"))
	require.NoError(err)
	err = db.Set([]byte("foo_three"), []byte("three"))
	require.NoError(err)

	values, err = db.Filter([]byte("foo"), 0, -1)
	require.NoError(err)
	require.Len(values, 3)
	require.Subset(values, [][]byte{[]byte("one"), []byte("two"), []byte("three")})

	values, err = db.Filter([]byte("foo"), 0, 3)
	require.NoError(err)
	require.Len(values, 3)
	require.Subset(values, [][]byte{[]byte("one"), []byte("two"), []byte("three")})

	values, err = db.Filter([]byte("foo"), 0, 2)
	require.NoError(err)
	require.Len(values, 2)
	require.Subset([][]byte{[]byte("one"), []byte("two"), []byte("three")}, values)

	values, err = db.Filter([]byte("foo"), 1, 2)
	require.NoError(err)
	require.Len(values, 2)
	require.Subset([][]byte{[]byte("one"), []byte("two"), []byte("three")}, values)

	values, err = db.Filter([]byte("foo"), 2, 1)
	require.NoError(err)
	require.Len(values, 1)
	require.Subset([][]byte{[]byte("one"), []byte("two"), []byte("three")}, values)

	values, err = db.Filter([]byte("foo"), 2, 42)
	require.NoError(err)
	require.Len(values, 1)
	require.Subset([][]byte{[]byte("one"), []byte("two"), []byte("three")}, values)

	values, err = db.Filter([]byte("foo"), 0, 1)
	require.NoError(err)
	require.Len(values, 1)
	require.Subset([][]byte{[]byte("one"), []byte("two"), []byte("three")}, values)

	values, err = db.Filter([]byte("foo"), 3, 1)
	require.NoError(err)
	require.Empty(values)

	values, err = db.Filter([]byte("foo"), 4, 1)
	require.NoError(err)
	require.Empty(values)
}

func TestDBList_Badger(t *testing.T) {
	db, cleanup := makeTestBadgerDB(t)
	defer cleanup()
	testDBList(t, db)
}

func TestDBList_Memory(t *testing.T) {
	db := memory.New()
	defer db.Close()
	testDBList(t, db)
}

func testDBList(t *testing.T, db DB) {
	require := require.New(t)

	keys, err := db.List([]byte("foo"))
	require.NoError(err)
	require.Empty(keys)

	err = db.Set([]byte("bar_one"), []byte("one"))
	require.NoError(err)

	keys, err = db.List([]byte("foo"))
	require.NoError(err)
	require.Empty(keys)

	keys, err = db.List([]byte("bar"))
	require.NoError(err)
	require.Len(keys, 1)
	require.Subset(keys, [][]byte{[]byte("bar_one")})

	err = db.Set([]byte("foo_one"), []byte("one"))
	require.NoError(err)

	keys, err = db.List([]byte("foo"))
	require.NoError(err)
	require.Len(keys, 1)
	require.Subset(keys, [][]byte{[]byte("foo_one")})

	err = db.Set([]byte("foo_two"), []byte("two"))
	require.NoError(err)
	err = db.Set([]byte("foo_three"), []byte("three"))
	require.NoError(err)

	keys, err = db.List([]byte("bar"))
	require.NoError(err)
	require.Len(keys, 1)
	require.Subset(keys, [][]byte{[]byte("bar_one")})

	keys, err = db.List([]byte("foo"))
	require.NoError(err)
	require.Len(keys, 3)
	require.Subset(keys, [][]byte{[]byte("foo_one"), []byte("foo_two"), []byte("foo_three")})
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
