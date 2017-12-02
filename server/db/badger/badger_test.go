package badger

import (
	"context"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zero-os/0-stor/server/db"
)

func makeTestBadgerDB(t *testing.T) (*DB, func()) {
	tmpDir, err := ioutil.TempDir("", "0-stor-test")
	require.NoError(t, err)

	badgerDB, err := New(path.Join(tmpDir, "data"), path.Join(tmpDir, "meta"))
	require.NoError(t, err)
	cleanup := func() {
		badgerDB.Close()
		os.RemoveAll(tmpDir)
	}
	return badgerDB, cleanup
}

func TestConstantErrors(t *testing.T) {
	require := require.New(t)

	ddb, cleanup := makeTestBadgerDB(t)
	defer cleanup()

	// nil-key errors
	require.Equal(db.ErrNilKey, ddb.Delete(nil))
	require.Equal(db.ErrNilKey, ddb.Set(nil, nil))
	_, err := ddb.Exists(nil)
	require.Equal(db.ErrNilKey, err)
	_, err = ddb.Get(nil)
	require.Equal(db.ErrNilKey, err)
	require.Equal(db.ErrNilKey, ddb.Update(nil,
		func([]byte) ([]byte, error) { return nil, nil }))

	// explicit panics
	require.Panics(func() {
		ddb.ListItems(nil, nil)
	}, "panics because context is required")
	require.Panics(func() {
		ddb.Update(nil, nil)
	}, "panics because callback is required")
}

func TestSetGet(t *testing.T) {
	ddb, cleanup := makeTestBadgerDB(t)
	defer cleanup()

	require := require.New(t)

	// set 2 different values, using the same input buffer
	value := []byte("ping")
	require.NoError(ddb.Set([]byte{0}, value))
	value[1] = 'o'
	require.NoError(ddb.Set([]byte{1}, value))

	// now ensure that the 2 different values are indeed different
	value, err := ddb.Get([]byte{0})
	require.NoError(err)
	require.Equal("ping", string(value))

	value, err = ddb.Get([]byte{1})
	require.NoError(err)
	require.Equal("pong", string(value))

	// now set a lot more values (using the shared input value)
	for i := 0; i < 255; i++ {
		value[1] = byte(i)
		require.NoError(ddb.Set(value[1:2], value))
	}
	// now ensure all these values are still correctly defined
	for i := 0; i < 255; i++ {
		value, err = ddb.Get([]byte{byte(i)})
		require.NoError(err)
		expected := []byte("ping")
		expected[1] = byte(i)
		require.Equal(expected, value)
	}
}

func TestReuseOfInputValueSlice(t *testing.T) {
	require := require.New(t)

	ddb, cleanup := makeTestBadgerDB(t)
	defer cleanup()

	// set 2 different values, using the same input buffer
	value := []byte("ping")
	require.NoError(ddb.Set([]byte{0}, value))
	value[1] = 'o'
	require.NoError(ddb.Set([]byte{1}, value))

	// now ensure that the 2 different values are indeed different
	value, err := ddb.Get([]byte{0})
	require.NoError(err)
	require.Equal("ping", string(value))
	value, err = ddb.Get([]byte{1})
	require.NoError(err)
	require.Equal("pong", string(value))

	// now set a lot more values (using the shared input value)
	for i := 0; i < 255; i++ {
		value[1] = byte(i)
		require.NoError(ddb.Set(value[1:2], value))
	}
	// now ensure all these values are still correctly defined
	for i := 0; i < 255; i++ {
		value, err = ddb.Get([]byte{byte(i)})
		require.NoError(err)
		expected := []byte("ping")
		expected[1] = byte(i)
		require.Equal(expected, value)
	}
}

func TestListItemsSimple(t *testing.T) {
	ddb, cleanup := makeTestBadgerDB(t)
	defer cleanup()

	require := require.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// when no items have been found, the channel will simply return nothing
	ch, err := ddb.ListItems(ctx, nil)
	require.NoError(err)
	for item := range ch {
		t.Fatal(item) // shouldn't happen, as there are no items
	}

	// add one item
	require.NoError(ddb.Set([]byte("a"), []byte("foo")))

	// should still have no items found, as there are no keys with the desired prefix
	ch, err = ddb.ListItems(ctx, []byte{'_'})
	require.NoError(err)
	for item := range ch {
		t.Fatal(item) // shouldn't happen, as there are no keys (with this prefix)
	}

	// when looking for no prefix, this one should be found though
	ch, err = ddb.ListItems(ctx, nil)
	require.NoError(err)
	var count int
	for item := range ch {
		if count > 0 {
			t.Fatal(item) // shouldn't happen, as there is only one key to be expected
		}
		require.NoError(item.Error())
		require.Equal([]byte("a"), item.Key())

		value, err := item.Value()
		require.NoError(err)
		require.Equal([]byte("foo"), value)

		count++ // increase count, so we can fail if we receive more then 1 item
		require.NoError(item.Close())
	}

	// when looking for a prefix, for which we do have keys, it should succeed
	ch, err = ddb.ListItems(ctx, []byte{'a'})
	require.NoError(err)
	count = 0 // reset count
	for item := range ch {
		if count > 0 {
			t.Fatal(item) // shouldn't happen, as there is only one key to be expected
		}
		require.NoError(item.Error())
		require.Equal([]byte("a"), item.Key())

		value, err := item.Value()
		require.NoError(err)
		require.Equal([]byte("foo"), value)

		count++ // increase count, so we can fail if we receive more then 1 item
		require.NoError(item.Close())
	}

	require.NoError(ddb.Set([]byte("_b"), []byte("bar")))
	require.NoError(ddb.Set([]byte("_c"), []byte("baz")))

	// when looking for no prefix, all keys should be returned
	ch, err = ddb.ListItems(ctx, nil)
	require.NoError(err)
	expectedKeys := []string{"a", "_b", "_c"}
	expectedValues := map[string]string{
		"a":  "foo",
		"_b": "bar",
		"_c": "baz",
	}
	for item := range ch {
		if len(expectedKeys) == 0 {
			t.Fatal(item) // shouldn't happen, as there is only one key to be expected
		}
		require.NoError(item.Error())
		k := item.Key()
		require.NotNil(k)
		key := string(k)
		require.Subset(expectedKeys, []string{key})
		expectedKeys = removeStringFromSlice(key, expectedKeys)

		value, err := item.Value()
		require.NoError(err)
		require.Equal(expectedValues[key], string(value))
		delete(expectedValues, key)
		require.NoError(item.Close())
	}
	require.Len(expectedKeys, 0, "all keys should have been found")

	// when looking for a prefix, only the prefixed keys should be returned
	ch, err = ddb.ListItems(ctx, []byte{'_'})
	require.NoError(err)
	expectedKeys = []string{"_b", "_c"}
	expectedValues = map[string]string{
		"_b": "bar",
		"_c": "baz",
	}
	for item := range ch {
		if len(expectedKeys) == 0 {
			t.Fatal(item) // shouldn't happen, as there is only one key to be expected
		}
		require.NoError(item.Error())
		k := item.Key()
		require.NotNil(k)
		key := string(k)
		require.Subset(expectedKeys, []string{key})
		expectedKeys = removeStringFromSlice(key, expectedKeys)

		value, err := item.Value()
		require.NoError(err)
		require.Equal(expectedValues[key], string(value))
		delete(expectedValues, key)
		require.NoError(item.Close())
	}
	require.Len(expectedKeys, 0, "all keys should have been found")
}

func removeStringFromSlice(str string, slice []string) []string {
	for i, it := range slice {
		if it == str {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}

// We want to test that:
//   + The function is thread-safe
//     ( we can spawn multiple iterators async );
//   + An item is no longer usuable after it has been closed;
//   + An iterator only continues/stops after the current Item has been closed;
//   + We can List without a prefix (the prefix is optional);
// NOTE: there is no guarantee that keys cannot be modified in the meanwhile
func TestListItemsComplete(t *testing.T) {
	ddb, cleanup := makeTestBadgerDB(t)
	defer cleanup()

	require := require.New(t)

	const prefix = byte(0)

	// create keys (data == key, for this test)
	var allKeys, keysPrefixed [][]byte
	for i := 1; i < 256; i++ {
		key, keyPrefixed := []byte{byte(i)}, []byte{prefix, byte(i)}
		allKeys = append(allKeys, key, keyPrefixed)
		keysPrefixed = append(keysPrefixed, keyPrefixed)
	}
	// add all keys to database
	for _, key := range allKeys {
		require.NoError(ddb.Set(key, key))
	}

	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const listThreadCount = 100
	wg.Add(listThreadCount)
	// now list all/some keys, multiple times, async
	for i := 0; i < listThreadCount; i++ {
		if i%2 == 0 {
			// test filtered list version
			ch, err := ddb.ListItems(ctx, []byte{prefix})
			require.NoError(err)
			require.NotNil(ch)
			go func() {
				defer wg.Done()
				testListItemsResult(t, ch, keysPrefixed)
			}()
		} else {
			// test non-filtered list version
			ch, err := ddb.ListItems(ctx, nil)
			require.NoError(err)
			require.NotNil(ch)
			go func() {
				defer wg.Done()
				testListItemsResult(t, ch, allKeys)
			}()
		}
	}

	// wait until all is finished
	wg.Wait()
}

func testListItemsResult(t *testing.T, ch <-chan db.Item, expectedKeys [][]byte) {
	require := require.New(t)

	var receivedKeys [][]byte
	var receivedValues [][]byte

	for item := range ch {
		// ensure no error has occurred
		require.NoError(item.Error())

		// get+copy+collect key
		key := item.Key()
		require.NotNil(key)
		keyCopy := make([]byte, len(key))
		copy(keyCopy, key)
		receivedKeys = append(receivedKeys, keyCopy)

		// get+copy+collect value
		value, err := item.Value()
		require.NoError(err)
		require.NotNil(value)
		valueCopy := make([]byte, len(value))
		copy(valueCopy, value)
		receivedValues = append(receivedValues, valueCopy)

		// close item, so we can continue our iteration
		require.NoError(item.Close())
	}

	// validate that the received keys and values is correct
	require.Len(receivedKeys, len(expectedKeys))
	require.Subsetf(expectedKeys, receivedKeys, "%v Vs. %v", expectedKeys, receivedKeys)
	require.Len(receivedValues, len(expectedKeys))
	require.Subsetf(expectedKeys, receivedValues, "%v Vs. %v", expectedKeys, receivedValues)
}

// tests if in an async environment,
// where multiple different listing get stopped before the ending,
// by means of the context,
// if all is still fine
func TestListItemsAbruptEnding(t *testing.T) {
	ddb, cleanup := makeTestBadgerDB(t)
	defer cleanup()

	require := require.New(t)

	var keys [][]byte
	// create keys (data == key, for this test)
	// and store them in the db
	for i := 0; i < 256; i++ {
		key := []byte{byte(i)}
		keys = append(keys, key)
		require.NoError(ddb.Set(key, key))
	}

	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const listThreadCount = 128
	wg.Add(listThreadCount)
	// now list all/some keys, multiple times, async
	for i := 0; i < listThreadCount; i++ {
		stopIndex := rand.Intn(listThreadCount)
		go func() {
			defer wg.Done()

			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			ch, err := ddb.ListItems(ctx, nil)
			require.NoError(err)
			require.NotNil(ch)

			var index int
			for item := range ch {
				exit, err := func() (bool, error) {
					defer func() {
						require.NoError(item.Close())
					}()

					if index == stopIndex {
						return true, nil // exit early, as to simulate failure at the user side
					}
					index++

					err := item.Error()
					if err != nil {
						return false, err
					}

					require.Subset(keys, [][]byte{item.Key()})
					value, err := item.Value()
					if err != nil {
						return false, err
					}
					require.Subset(keys, [][]byte{value})

					return false, nil
				}()

				require.NoError(err)
				if exit {
					return
				}
			}
		}()
	}

	// wait until all is finished
	wg.Wait()
}

// test to ensure that updating a (non-)existing key is possible
func TestUpdate(t *testing.T) {
	ddb, cleanup := makeTestBadgerDB(t)
	defer cleanup()

	require := require.New(t)

	var (
		key   = []byte("key")
		value = []byte("value")
	)

	require.NoError(ddb.Update(key, func(data []byte) ([]byte, error) {
		return value, nil
	}), "updating an item should always be possible for memory DB (even if the key doesn't exist)")

	v, err := ddb.Get(key)
	require.NoError(err)
	require.Equal(value, v)

	value = []byte("new value")
	require.NoError(ddb.Update(key, func(data []byte) ([]byte, error) {
		return value, nil
	}), "updating an item should always be possible for memory DB")

	v, err = ddb.Get(key)
	require.NoError(err)
	require.Equal(value, v)
}

func TestUpdateExistsDelete(t *testing.T) {
	ddb, cleanup := makeTestBadgerDB(t)
	defer cleanup()

	require := require.New(t)

	key := []byte("key")
	value := []byte("old value")

	require.NoError(ddb.Set(key, value), "Setting a value should be always possible for memory DB")

	// test Update interface
	require.NoError(ddb.Update(key, func(data []byte) ([]byte, error) {
		return []byte("new value"), nil
	}), "updating an item should always be possible for memory DB")

	newValue, err := ddb.Get(key)

	require.NoError(err, "getting interface shouldn't fail here")
	require.NotEqual(value, newValue)

	// test Exists interface
	exists, err := ddb.Exists(key)
	require.NoError(err, "Calling Exists shouldn't fail here")
	require.True(exists, "fail to find the item")

	exists, err = ddb.Exists([]byte("wrong key"))
	require.NoError(err, "Calling Exists shouldn't fail here")
	require.False(exists, "false item detection")

	// test Delete interface
	require.NoError(ddb.Delete(key), "Deleting a non-existing key shouldn't fail")
}

func TestRaceCondition(t *testing.T) {
	ddb, cleanup := makeTestBadgerDB(t)
	defer cleanup()

	require := require.New(t)

	key := []byte("key")
	value := []byte("A")

	require.NoError(ddb.Set(key, value), "setting the item should be OK here")
	val, err := ddb.Get(key)
	require.NoError(err)
	require.Equal(value, val)

	nThreads := 25
	var wg sync.WaitGroup

	// updating a value with multiple threads
	wg.Add(nThreads)
	for i := 0; i < nThreads; i++ {
		go func() {
			defer wg.Done()
			for {
				err := ddb.Update(key, func(data []byte) ([]byte, error) {
					data[0]++
					return data, nil
				})
				if err == nil {
					return // we're done!
				}
				if err == db.ErrConflict {
					continue // simply try again
				}
				t.Fatal(err) // shouldn't ever happen
			}
		}()
	}
	wg.Wait()

	// check the last updated value
	value, err = ddb.Get(key)

	require.NoError(err, "Getting the item shouldn't fail here")
	require.Equal(string(value), "Z", "unexpected result, race condition possible")
}
