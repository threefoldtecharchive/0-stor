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
	"bytes"
	"context"
	"fmt"
	"sort"
	"strconv"
	"testing"

	dbp "github.com/zero-os/0-stor/client/metastor/db"

	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

// RoundTrip simply tests test the client's set-get-delete cycle
// for all kinds of metadata.
func RoundTrip(t *testing.T, db dbp.DB) {
	require := require.New(t)
	require.NotNil(db)

	var (
		key  = []byte("foo")
		data = []byte("bar")
	)

	// ensure metadata is not there yet
	_, err := db.Get(key)
	require.Equal(dbp.ErrNotFound, err)

	// set the metadata
	err = db.Set(key, data)
	require.NoError(err)

	// get it back
	storedData, err := db.Get(key)
	require.NoError(err)
	// check stored value
	require.NotEmpty(storedData)
	require.Equal(data, storedData)

	// delete the metadata
	err = db.Delete(key)
	require.NoError(err)
	// make sure we can't get it back
	_, err = db.Get(key)
	require.Equal(dbp.ErrNotFound, err)
}

// SyncUpdate tests that the given database
// can Update existing metadata in a synchronous scenario.
func SyncUpdate(t *testing.T, db dbp.DB) {
	require := require.New(t)

	key := []byte("foo")

	err := db.Update(key, func(bs []byte) ([]byte, error) { return bs, nil })
	require.Equal(dbp.ErrNotFound, err)

	data := []byte("foo")
	err = db.Set(key, data)
	require.NoError(err)

	output, err := db.Get(data)
	require.NoError(err)
	require.Equal(data, output)

	err = db.Update(key, func(bs []byte) ([]byte, error) {
		bs[0] = 'b'
		return bs, nil
	})
	require.NoError(err)

	output, err = db.Get(key)
	require.NoError(err)
	require.NotEqual(data, output)
	data[0] = 'b'
	require.Equal(data, output)
}

// AsyncUpdate tests that the given database
// can Update existing metadata in an asynchronous scenario.
func AsyncUpdate(t *testing.T, db dbp.DB) {
	require := require.New(t)
	require.NotNil(db)

	const (
		jobs = 128
	)
	var (
		err  error
		key  = []byte("foo")
		data = []byte("l:")
	)

	err = db.Set(key, data)
	require.NoError(err)

	group, _ := errgroup.WithContext(context.Background())
	for i := 0; i < jobs; i++ {
		i := i
		group.Go(func() error {
			return db.Update(key, func(bs []byte) ([]byte, error) {
				bs = []byte(string(bs) + fmt.Sprintf("%d,", i))
				return bs, nil
			})
		})
	}
	require.NoError(group.Wait())

	data, err = db.Get(key)
	require.NoError(err)
	require.True(bytes.HasPrefix(data, []byte{'l', ':'}))

	rawIntegers := bytes.Split(data[2:len(data)-1], []byte{','})
	require.Len(rawIntegers, jobs)

	integers := make([]int, jobs)
	for i, raw := range rawIntegers {
		integer, err := strconv.Atoi(string(raw))
		require.NoError(err)
		integers[i] = integer
	}

	sort.Ints(integers)
	for i := 0; i < jobs; i++ {
		require.Equal(i, integers[i])
	}
}
