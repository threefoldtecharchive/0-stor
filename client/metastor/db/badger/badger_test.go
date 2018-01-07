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

	"github.com/zero-os/0-stor/client/metastor/db/test"

	"github.com/stretchr/testify/require"
)

func makeTestDB(t *testing.T) (*DB, func()) {
	tmpDir, err := ioutil.TempDir("", "0-stor-test")
	require.NoError(t, err)

	db, err := New(path.Join(tmpDir, "data"), path.Join(tmpDir, "meta"))
	require.NoError(t, err)
	require.NotNil(t, db)
	cleanup := func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}
	return db, cleanup
}

func TestInMemoryDB_RoundTrip(t *testing.T) {
	db, cleanup := makeTestDB(t)
	defer cleanup()
	test.RoundTrip(t, db)
}

func TestInMemoryDB_SyncUpdate(t *testing.T) {
	db, cleanup := makeTestDB(t)
	defer cleanup()
	test.SyncUpdate(t, db)
}

func TestInMemoryDB_AsyncUpdate(t *testing.T) {
	db, cleanup := makeTestDB(t)
	defer cleanup()
	test.AsyncUpdate(t, db)
}
