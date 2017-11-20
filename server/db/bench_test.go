package db

import (
	"fmt"
	"testing"

	log "github.com/Sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func BenchmarkSetReferenceList(b *testing.B) {
	log.SetLevel(log.DebugLevel)
	db, cleanup := makeTestBadgerDB(b)
	defer cleanup()

	sizes := []uint64{1024, 1024 * 4, 1024 * 10, 1024 * 1024}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("with data, size:%d", size), func(b *testing.B) {
			benchmarkSetReferenceList(b, db, size)
		})
	}
}

func benchmarkSetReferenceList(b *testing.B, db DB, size uint64) {
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
		tmp := NewObject(obj.Namespace, obj.Key, db)
		if err := tmp.SetReferenceList(refList); err != nil {
			b.Errorf("error set reference list: %v", err)
		}
	}
}
