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

package badger

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/zero-os/0-stor/server/db"
	"github.com/zero-os/0-stor/server/db/test"

	"github.com/stretchr/testify/require"
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

	// explicit panics
	require.Panics(func() {
		ddb.ListItems(nil, nil)
	}, "panics because callback is required")
}

func TestSetGet(t *testing.T) {
	ddb, cleanup := makeTestBadgerDB(t)
	defer cleanup()
	test.SetGet(t, ddb)
}

func TestSetIncremented(t *testing.T) {
	ddb, cleanup := makeTestBadgerDB(t)
	defer cleanup()
	test.SetIncremented(t, ddb)
}

func TestSetIncremented_Async(t *testing.T) {
	ddb, cleanup := makeTestBadgerDB(t)
	defer cleanup()
	test.SetIncrementedAsync(t, ddb)
}

func TestReuseOfInputValueSlice(t *testing.T) {
	ddb, cleanup := makeTestBadgerDB(t)
	defer cleanup()
	test.ReuseOfInputValueSlice(t, ddb)
}

func TestListItemsSimple(t *testing.T) {
	ddb, cleanup := makeTestBadgerDB(t)
	defer cleanup()
	test.ListItemsSimple(t, ddb)
}

func TestListItemsComplete(t *testing.T) {
	ddb, cleanup := makeTestBadgerDB(t)
	defer cleanup()
	test.ListItemsComplete(t, ddb)
}

func TestListItemsAbruptEnding(t *testing.T) {
	ddb, cleanup := makeTestBadgerDB(t)
	defer cleanup()
	test.ListItemsAbruptEnding(t, ddb)
}

func TestSetExistsDelete(t *testing.T) {
	ddb, cleanup := makeTestBadgerDB(t)
	defer cleanup()
	test.SetExistsDelete(t, ddb)
}
