package db_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	dbp "github.com/zero-os/0-stor/server/db"
	"github.com/zero-os/0-stor/server/db/memory"
)

func BenchmarkSetReferenceList_Badger(b *testing.B) {
	db, cleanup := makeTestBadgerDB(b)
	defer cleanup()
	benchmarkSetReferenceList(b, db)
}

func BenchmarkSetReferenceList_Memory(b *testing.B) {
	db := memory.New()
	defer db.Close()
	benchmarkSetReferenceList(b, db)
}

func benchmarkSetReferenceList(b *testing.B, db dbp.DB) {
	sizes := []uint64{1024, 1024 * 4, 1024 * 10, 1024 * 1024}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("with data, size:%d", size), func(b *testing.B) {
			benchmarkSetReferenceList_size(b, db, size)
		})
	}
}

func benchmarkSetReferenceList_size(b *testing.B, db dbp.DB, size uint64) {
	// save object first cause we want to test speed when loading the object from db
	obj, _, _ := makeTestObj(b, size, db)
	err := obj.Save()
	require.NoError(b, err, "fail to save object")

	refList := make([]string, 50)
	for i := 0; i < 50; i++ {
		refList[i] = fmt.Sprintf("user%d", i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// create new object and benchmark
		tmp := dbp.NewObject(obj.Namespace, obj.Key, db)
		if err := tmp.SetReferenceList(refList); err != nil {
			b.Errorf("error set reference list: %v", err)
		}
	}
}
