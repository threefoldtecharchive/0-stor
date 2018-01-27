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

package memory

import (
	"testing"

	"github.com/zero-os/0-stor/server/db"
	"github.com/zero-os/0-stor/server/db/test"

	"github.com/stretchr/testify/require"
)

func TestConstantErrors(t *testing.T) {
	require := require.New(t)

	mdb := New()
	defer mdb.Close()

	// nil-key errors
	require.Equal(db.ErrNilKey, mdb.Delete(nil))
	require.Equal(db.ErrNilKey, mdb.Set(nil, nil))
	_, err := mdb.Exists(nil)
	require.Equal(db.ErrNilKey, err)
	_, err = mdb.Get(nil)
	require.Equal(db.ErrNilKey, err)

	// explicit panics
	require.Panics(func() {
		mdb.ListItems(nil, nil)
	}, "panics because callback is required")
}

func TestSetGet(t *testing.T) {
	mdb := New()
	defer mdb.Close()

}

func TestSetIncremented(t *testing.T) {
	mdb := New()
	defer mdb.Close()
	test.SetIncremented(t, mdb)
}

func TestSetIncremented_Async(t *testing.T) {
	mdb := New()
	defer mdb.Close()
	test.SetIncrementedAsync(t, mdb)
}

func TestReuseOfInputValueSlice(t *testing.T) {
	mdb := New()
	defer mdb.Close()
	test.ReuseOfInputValueSlice(t, mdb)
}

func TestListItemsSimple(t *testing.T) {
	mdb := New()
	defer mdb.Close()
	test.ListItemsSimple(t, mdb)
}

func TestListItemsComplete(t *testing.T) {
	mdb := New()
	defer mdb.Close()
	test.ListItemsComplete(t, mdb)
}

func TestListItemsAbruptEnding(t *testing.T) {
	mdb := New()
	defer mdb.Close()
	test.ListItemsAbruptEnding(t, mdb)
}

func TestSetExistsDelete(t *testing.T) {
	mdb := New()
	defer mdb.Close()
	test.SetExistsDelete(t, mdb)
}
