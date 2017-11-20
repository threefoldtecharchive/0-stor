package memory

import (
	"strings"
	"sync"

	"github.com/zero-os/0-stor/server/errors"
)

// DB implements the db.DB interace
type DB struct {
	m   map[string][]byte
	mux sync.RWMutex
}

// New creates a new in-memory DB,
// useful for testing and development purposes only.
func New() *DB {
	return &DB{
		m: make(map[string][]byte),
	}
}

// Get implements DB.Get
func (mdb *DB) Get(key []byte) ([]byte, error) {
	mdb.mux.RLock()
	val, exists := mdb.m[string(key)]
	if !exists {
		mdb.mux.RUnlock()
		return nil, errors.ErrNotFound
	}

	b := make([]byte, len(val))
	copy(b, val)
	mdb.mux.RUnlock()

	return b, nil
}

// Exists implements DB.Exists
func (mdb *DB) Exists(key []byte) (bool, error) {
	mdb.mux.RLock()
	_, exists := mdb.m[string(key)]
	mdb.mux.RUnlock()
	return exists, nil
}

// Filter implements DB.Filter
func (mdb *DB) Filter(prefix []byte, start int, count int) ([][]byte, error) {
	mdb.mux.RLock()
	defer mdb.mux.RUnlock()

	if count == 0 {
		return nil, nil
	}

	var (
		counter int
		out     [][]byte
		buf     []byte
	)

	sprefix := string(prefix)

	for k, v := range mdb.m {
		if strings.HasPrefix(k, sprefix) {
			// Skip until starting index
			if start > counter {
				counter++
				continue
			}

			buf = make([]byte, len(v))
			copy(buf, v)
			out = append(out, buf)
			// stop if we have reached our limit
			if len(out) == count {
				break
			}
		}
	}

	return out, nil
}

// List implements DB.List
func (mdb *DB) List(prefix []byte) ([][]byte, error) {
	mdb.mux.RLock()
	defer mdb.mux.RUnlock()

	out := make([][]byte, 0)
	sprefix := string(prefix)

	for k := range mdb.m {
		if strings.HasPrefix(k, sprefix) {
			out = append(out, []byte(k))
		}
	}
	return out, nil
}

// Set implements DB.Set
func (mdb *DB) Set(key []byte, value []byte) error {
	b := make([]byte, len(value))
	copy(b, value)
	mdb.mux.Lock()
	mdb.m[string(key)] = b
	mdb.mux.Unlock()
	return nil
}

// Delete implements DB.Delete
func (mdb *DB) Delete(key []byte) error {
	mdb.mux.Lock()
	delete(mdb.m, string(key))
	mdb.mux.Unlock()
	return nil
}

// Close implements DB.Close
func (mdb *DB) Close() error {
	mdb.mux.Lock()
	mdb.m = make(map[string][]byte)
	mdb.mux.Unlock()
	return nil
}
