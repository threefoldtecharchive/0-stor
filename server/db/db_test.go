package db_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zero-os/0-stor/server/db"
	"github.com/zero-os/0-stor/server/db/memory"
)

func TestCountKeys(t *testing.T) {
	require := require.New(t)

	// CountKeys should panic in case no db is given
	require.Panics(func() {
		db.CountKeys(nil, nil)
	})

	mdb := memory.New()
	require.NotNil(mdb)
	defer mdb.Close()

	// should be 0 keys, as we haven't added anything yet
	n, err := db.CountKeys(mdb, nil)
	require.NoError(err)
	require.Zero(n)

	// let's add one value
	require.NoError(mdb.Set([]byte("a"), []byte("bar")))

	// count again
	n, err = db.CountKeys(mdb, nil)
	require.NoError(err)
	require.Equal(1, n)

	// let's add some more (prefixed) values)
	require.NoError(mdb.Set([]byte("_b"), []byte("baz")))
	require.NoError(mdb.Set([]byte("_f"), []byte("foo")))

	// count again (unfiltered)
	n, err = db.CountKeys(mdb, nil)
	require.NoError(err)
	require.Equal(3, n)

	// count again (filered)
	n, err = db.CountKeys(mdb, []byte{'_'})
	require.NoError(err)
	require.Equal(2, n)

	// count again (filered)
	n, err = db.CountKeys(mdb, []byte("_b"))
	require.NoError(err)
	require.Equal(1, n)
	n, err = db.CountKeys(mdb, []byte{'a'})
	require.NoError(err)
	require.Equal(1, n)
}

func TestErrorItem(t *testing.T) {
	require := require.New(t)

	myError := errors.New("error")
	item := db.ErrorItem{Err: myError}
	require.Equal(myError, item.Error())
	require.Equal(myError, item.Close())
	require.Nil(item.Key())
	value, err := item.Value()
	require.Nil(value)
	require.Equal(myError, err)
}
