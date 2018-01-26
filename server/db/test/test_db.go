/*
 * Copyright (C) 2017-2018 GIG Technology NV and Contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package test

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"testing"

	"github.com/zero-os/0-stor/server/db"

	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

// SetGet tests the Set and Get methods of a database implementation.
func SetGet(t *testing.T, ddb db.DB) {
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

// SetIncremented tests the SetIncremented method of a database
func SetIncremented(t *testing.T, ddb db.DB) {
	require := require.New(t)

	label := []byte("foo")

	key1, err := ddb.SetScoped(label, []byte("foo"))
	require.NoError(err)
	require.NotEqual(label, key1)
	require.NotEmpty(key1)

	value1, err := ddb.Get(key1)
	require.NoError(err)
	require.Equal([]byte("foo"), value1)

	key2, err := ddb.SetScoped(label, []byte("bar"))
	require.NoError(err)
	require.NotEqual(label, key2)
	require.NotEqual(key1, key2)
	require.NotEmpty(key2)

	value2, err := ddb.Get(key2)
	require.NoError(err)
	require.Equal([]byte("bar"), value2)

	value1, err = ddb.Get(key1)
	require.NoError(err)
	require.Equal([]byte("foo"), value1)

	expectedKeys := []string{
		string(db.ScopedSequenceKey([]byte("foo"), 0)),
		string(db.ScopedSequenceKey([]byte("foo"), 1)),
	}
	expectedValues := map[string]string{
		expectedKeys[0]: "foo",
		expectedKeys[1]: "bar",
	}
	err = ddb.ListItems(func(item db.Item) error {
		if len(expectedKeys) == 0 {
			t.Fatal(item) // shouldn't happen, as there is only one key to be expected
		}
		k, err := item.Key()
		require.NoError(err)
		require.NotNil(k)
		key := string(k)
		require.Subset(expectedKeys, []string{key})
		expectedKeys = removeStringFromSlice(key, expectedKeys)

		value, err := item.Value()
		require.NoError(err)
		require.Equal(expectedValues[key], string(value))
		delete(expectedValues, key)
		return nil
	}, label)
	require.NoError(err)
	require.Len(expectedKeys, 0, "all keys should have been found")

	err = ddb.Delete(key2)
	require.NoError(err)

	_, err = ddb.Get(key2)
	require.Equal(db.ErrNotFound, err)

	key3, err := ddb.SetScoped(label, []byte("42"))
	require.NoError(err)
	require.NotEqual(label, key3)
	require.NotEqual(key1, key3)
	require.NotEqual(key1, key2)
	require.NotEmpty(key3)

	value1, err = ddb.Get(key1)
	require.NoError(err)
	require.Equal([]byte("foo"), value1)

	value3, err := ddb.Get(key3)
	require.NoError(err)
	require.Equal([]byte("42"), value3)

	expectedKeys = []string{
		string(db.ScopedSequenceKey([]byte("foo"), 0)),
		string(db.ScopedSequenceKey([]byte("foo"), 2)),
	}
	expectedValues = map[string]string{
		expectedKeys[0]: "foo",
		expectedKeys[1]: "42",
	}
	err = ddb.ListItems(func(item db.Item) error {
		if len(expectedKeys) == 0 {
			t.Fatal(item) // shouldn't happen, as there is only one key to be expected
		}
		k, err := item.Key()
		require.NoError(err)
		require.NotNil(k)
		key := string(k)
		require.Subset(expectedKeys, []string{key})
		expectedKeys = removeStringFromSlice(key, expectedKeys)

		value, err := item.Value()
		require.NoError(err)
		require.Equal(expectedValues[key], string(value))
		delete(expectedValues, key)
		return nil
	}, label)
	require.NoError(err)
	require.Len(expectedKeys, 0, "all keys should have been found")
}

// SetIncrementedAsync tests the SetIncremented function of a given database,
// in an asynchronous fashion.
func SetIncrementedAsync(t *testing.T, ddb db.DB) {
	// generate keys
	const (
		keyCount     = 128
		keyMaxSize   = 32
		keyMinSize   = 8
		valueMaxSize = 256
		valueMinSize = 32
	)
	keys := make([][]byte, keyCount)
	const chars = "abcdefghijklmnopqrtsuvwxyzABCDEFGHIJKLMNOPQRSTUVXZYZ-@_+"
	for i := range keys {
		id := make([]byte, rand.Int31n(keyMaxSize-keyMinSize)+keyMinSize)
		for u := range id {
			id[u] = chars[rand.Int31n(int32(len(chars)))]
		}
		keys[i] = []byte(fmt.Sprintf("d:%d:%s", i, id))
	}

	// increment all counters

	const (
		incrementCount = 256
		workerCount    = keyCount * 2
		countPerWorker = incrementCount / 2
	)
	type output struct {
		ScopeKey string
		Key      string
		Value    string
	}
	outputCh := make(chan output, workerCount*2)

	group, ctx := errgroup.WithContext(context.Background())
	// create multiple workers per key
	for i := 0; i < workerCount; i++ {
		index := i % keyCount
		group.Go(func() error {
			for i := 0; i < countPerWorker; i++ {
				value := make([]byte, rand.Int31n(valueMaxSize-valueMinSize)+valueMinSize)
				for u := range value {
					value[u] = chars[rand.Int31n(int32(len(chars)))]
				}
				key, err := ddb.SetScoped(keys[index], value)
				if err != nil {
					return err
				}

				output := output{
					ScopeKey: string(keys[index]),
					Key:      string(key),
					Value:    string(value),
				}
				select {
				case outputCh <- output:
				case <-ctx.Done():
					return nil
				}
			}
			return nil
		})
	}
	go func() {
		err := group.Wait()
		close(outputCh)
		require.NoError(t, err)
	}()

	allOutput := make(map[string]map[string]string, keyCount)
	for _, key := range keys {
		allOutput[string(key)] = make(map[string]string)
	}
	for output := range outputCh {
		require.True(t, strings.HasPrefix(output.Key, output.ScopeKey))

		m, ok := allOutput[output.ScopeKey]
		require.True(t, ok)

		_, ok = m[output.Key]
		require.False(t, ok)

		value, err := ddb.Get([]byte(output.Key))
		require.NoErrorf(t, err, "key: %s", output.Key)
		require.Equal(t, output.Value, string(value))

		m[output.Key] = output.Value
	}
	for _, m := range allOutput {
		require.Len(t, m, incrementCount)
		for k, v := range m {
			value, err := ddb.Get([]byte(k))
			require.NoError(t, err)
			require.Equal(t, v, string(value))
		}
	}
}

// ReuseOfInputValueSlice tests that a given database
// correctly handles the input variables it receives,
// and doesn't try to take ownership over them.
func ReuseOfInputValueSlice(t *testing.T, ddb db.DB) {
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

// ListItemsSimple tests the ListItems method of a database,
// in a very simple scenario.
func ListItemsSimple(t *testing.T, ddb db.DB) {
	require := require.New(t)

	// when no items have been found, the channel will simply return nothing
	err := ddb.ListItems(func(item db.Item) error {
		t.Fatal(item) // shouldn't happen, as there are no items
		return nil
	}, nil)
	require.NoError(err)

	// add one item
	require.NoError(ddb.Set([]byte("a"), []byte("foo")))

	// should still have no items found, as there are no keys with the desired prefix
	err = ddb.ListItems(func(item db.Item) error {
		t.Fatal(item) // shouldn't happen, as there are no keys (with this prefix)
		return nil
	}, []byte{'_'})
	require.NoError(err)

	// when looking for no prefix, this one should be found though
	var count int
	err = ddb.ListItems(func(item db.Item) error {
		if count > 0 {
			t.Fatal(item) // shouldn't happen, as there is only one key to be expected
		}
		key, err := item.Key()
		require.NoError(err)
		require.Equal([]byte("a"), key)

		value, err := item.Value()
		require.NoError(err)
		require.Equal([]byte("foo"), value)

		count++ // increase count, so we can fail if we receive more then 1 item
		return nil
	}, nil)
	require.NoError(err)

	// when looking for a prefix, for which we do have keys, it should succeed
	count = 0 // reset count
	err = ddb.ListItems(func(item db.Item) error {
		if count > 0 {
			t.Fatal(item) // shouldn't happen, as there is only one key to be expected
		}
		key, err := item.Key()
		require.NoError(err)
		require.Equal([]byte("a"), key)

		value, err := item.Value()
		require.NoError(err)
		require.Equal([]byte("foo"), value)

		count++ // increase count, so we can fail if we receive more then 1 item
		return nil
	}, []byte{'a'})
	require.NoError(err)

	require.NoError(ddb.Set([]byte("_b"), []byte("bar")))
	require.NoError(ddb.Set([]byte("_c"), []byte("baz")))

	// when looking for no prefix, all keys should be returned
	expectedKeys := []string{"a", "_b", "_c"}
	expectedValues := map[string]string{
		"a":  "foo",
		"_b": "bar",
		"_c": "baz",
	}
	err = ddb.ListItems(func(item db.Item) error {
		if len(expectedKeys) == 0 {
			t.Fatal(item) // shouldn't happen
		}
		k, err := item.Key()
		require.NoError(err)
		require.NotNil(k)
		key := string(k)
		require.Subset(expectedKeys, []string{key})
		expectedKeys = removeStringFromSlice(key, expectedKeys)

		value, err := item.Value()
		require.NoError(err)
		require.Equal(expectedValues[key], string(value))
		delete(expectedValues, key)
		return nil
	}, nil)
	require.NoError(err)
	require.Len(expectedKeys, 0, "all keys should have been found")

	// when looking for a prefix, only the prefixed keys should be returned
	expectedKeys = []string{"_b", "_c"}
	expectedValues = map[string]string{
		"_b": "bar",
		"_c": "baz",
	}
	err = ddb.ListItems(func(item db.Item) error {
		if len(expectedKeys) == 0 {
			t.Fatal(item) // shouldn't happen
		}
		k, err := item.Key()
		require.NoError(err)
		require.NotNil(k)
		key := string(k)
		require.Subset(expectedKeys, []string{key})
		expectedKeys = removeStringFromSlice(key, expectedKeys)

		value, err := item.Value()
		require.NoError(err)
		require.Equal(expectedValues[key], string(value))
		delete(expectedValues, key)
		return nil
	}, []byte{'_'})
	require.NoError(err)
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

// ListItemsComplete tests te ListItems method of a given database.
// It tests that:
//   + The function is thread-safe
//     ( we can spawn multiple iterators async );
//   + An item is no longer usuable after it has been closed;
//   + An iterator only continues/stops after the current Item has been closed;
//   + We can List without a prefix (the prefix is optional);
// NOTE: there is no guarantee that keys cannot be modified in the meanwhile
func ListItemsComplete(t *testing.T, ddb db.DB) {
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

	const listThreadCount = 100
	wg.Add(listThreadCount)
	// now list all/some keys, multiple times, async
	for i := 0; i < listThreadCount; i++ {
		if i%2 == 0 {
			// test filtered list version
			go func() {
				defer wg.Done()
				testListItemsResult(t, ddb, []byte{prefix}, keysPrefixed)
			}()
		} else {
			// test non-filtered list version
			go func() {
				defer wg.Done()
				testListItemsResult(t, ddb, nil, allKeys)
			}()
		}
	}

	// wait until all is finished
	wg.Wait()
}

func testListItemsResult(t *testing.T, ddb db.DB, prefix []byte, expectedKeys [][]byte) {
	require := require.New(t)

	var receivedKeys [][]byte
	var receivedValues [][]byte

	err := ddb.ListItems(func(item db.Item) error {
		// get+copy+collect key
		key, err := item.Key()
		require.NoError(err)
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

		return nil
	}, prefix)
	require.NoError(err)

	// validate that the received keys and values is correct
	require.Len(receivedKeys, len(expectedKeys))
	require.Subsetf(expectedKeys, receivedKeys, "%v Vs. %v", expectedKeys, receivedKeys)
	require.Len(receivedValues, len(expectedKeys))
	require.Subsetf(expectedKeys, receivedValues, "%v Vs. %v", expectedKeys, receivedValues)
}

// ListItemsAbruptEnding tests if in an async environment,
// where multiple different listing get stopped before the ending,
// by means of the context,
// if all is still fine
func ListItemsAbruptEnding(t *testing.T, ddb db.DB) {
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

	const listThreadCount = 128
	wg.Add(listThreadCount)
	// now list all/some keys, multiple times, async
	for i := 0; i < listThreadCount; i++ {
		stopIndex := rand.Intn(listThreadCount)
		go func() {
			defer wg.Done()

			var index int
			err := ddb.ListItems(func(item db.Item) error {
				exit, err := func() (bool, error) {
					if index == stopIndex {
						return true, nil // exit early, as to simulate failure at the user side
					}
					index++

					key, err := item.Key()
					if err != nil {
						return false, err
					}
					require.Subset(keys, [][]byte{key})
					value, err := item.Value()
					if err != nil {
						return false, err
					}
					require.Subset(keys, [][]byte{value})

					return false, nil
				}()

				require.NoError(err)
				if exit {
					return errStopEarlyIteration
				}
				return nil
			}, nil)
			require.True(err == nil || err == errStopEarlyIteration)
		}()
	}

	// wait until all is finished
	wg.Wait()
}

var (
	errStopEarlyIteration = errors.New("stop early")
)

// SetExistsDelete tests the set, exists and delete methods of a given database.
func SetExistsDelete(t *testing.T, ddb db.DB) {
	require := require.New(t)

	key := []byte("key")
	value := []byte("old value")

	require.NoError(ddb.Set(key, value), "Setting a value should be always possible for memory DB")

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
