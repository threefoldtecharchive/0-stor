package badger

import (
	"github.com/dgraph-io/badger"
)

func getTestOptions(dir string) *badger.Options {
	opt := new(badger.Options)
	*opt = badger.DefaultOptions
	opt.MaxTableSize = 1 << 15 // Force more compaction.
	opt.LevelOneSize = 4 << 15 // Force more compaction.
	opt.Dir = dir
	opt.ValueDir = dir
	opt.SyncWrites = true // Some tests seem to need this to pass.
	opt.ValueGCThreshold = 0.0
	return opt
}

//
// func TestExists(t *testing.T) {
// 	dir, err := ioutil.TempDir("/tmp", "badger")
// 	require.NoError(t, err)
// 	defer os.RemoveAll(dir)
// 	kv, err := badger.NewKV(getTestOptions(dir))
// 	require.NoError(t, err)
// 	defer kv.Close()
//
// 	// populate kv
// 	err = kv.Set([]byte("foo"), []byte("bar"))
// 	require.NoError(t, err)
// 	kv.Close()
//
// 	db, err := New(config.Settings{})
// 	require.NoError(t, err)
// 	defer kv.Close()
//
// 	tt := []struct {
// 		key     string
// 		exists  bool
// 		message string
// 	}{
// 		{
// 			key:     "foo",
// 			exists:  true,
// 			message: "Should exists",
// 		},
// 		{
// 			key:     "noexists",
// 			exists:  false,
// 			message: "Should not exists",
// 		},
// 	}
//
// 	for _, test := range tt {
// 		t.Run(test.message, func(t *testing.T) {
// 			result, err := db.Exists(test.key)
// 			require.NoError(t, err)
// 			assert.Equal(t, test.exists, result, test.message)
// 		})
// 	}
// }
