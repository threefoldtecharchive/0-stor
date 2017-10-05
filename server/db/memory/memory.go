package memory

import (
	"strings"

	"github.com/zero-os/0-stor/server/db"
	"github.com/zero-os/0-stor/server/errors"
)

var _ db.DB = (*memroyDB)(nil)

type memroyDB struct {
	m map[string][]byte
}

func New() db.DB {
	return &memroyDB{
		m: make(map[string][]byte),
	}
}

func (mdb *memroyDB) Get(key []byte) ([]byte, error) {
	val, exists := mdb.m[string(key)]
	if !exists {
		return nil, errors.ErrNotFound
	}
	return val, nil
}

func (mdb *memroyDB) Exists(key []byte) (bool, error) {
	_, exists := mdb.m[string(key)]
	return exists, nil
}

func (mdb *memroyDB) Filter(prefix []byte, start int, count int) ([][]byte, error) {

	i, n := 0, 0
	out := make([][]byte, 0, 100)
	sprefix := string(prefix)

	for k, v := range mdb.m {
		if start < i {
			i++
			continue
		}

		if n >= count {
			break
		}

		if strings.HasPrefix(k, sprefix) {
			out[n] = v
			n++
		}

		i++
	}

	return out, nil
}

func (mdb *memroyDB) List(prefix []byte) ([][]byte, error) {
	l := make([][]byte, len(mdb.m))
	i := 0
	for k := range mdb.m {
		l[i] = []byte(k)
		i++
	}
	return l, nil
}

func (mdb *memroyDB) Set(key []byte, value []byte) error {
	mdb.m[string(key)] = value
	return nil
}

func (mdb *memroyDB) Delete(key []byte) error {
	delete(mdb.m, string(key))
	return nil
}

func (mdb *memroyDB) Close() error {
	return nil
}
