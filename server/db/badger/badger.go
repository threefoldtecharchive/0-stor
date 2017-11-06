package badger

import (
	"os"

	log "github.com/Sirupsen/logrus"
	badgerdb "github.com/dgraph-io/badger"
	"github.com/zero-os/0-stor/server/errors"
)

// BadgerDB implements the db.DB interace
type BadgerDB struct {
	db *badgerdb.DB
	// Config *config.Settings
}

// New creates new badger DB with default options
func New(data, meta string) (*BadgerDB, error) {
	opts := badgerdb.DefaultOptions
	opts.SyncWrites = true
	return NewWithOpts(data, meta, opts)
}

// NewWithOpts creates new badger DB with own options
func NewWithOpts(data, meta string, opts badgerdb.Options) (*BadgerDB, error) {
	if err := os.MkdirAll(meta, 0774); err != nil {
		log.Errorf("\t\tMeta dir: %v [ERROR]", meta)
		return nil, err
	}

	if err := os.MkdirAll(data, 0774); err != nil {
		log.Errorf("\t\tData dir: %v [ERROR]", data)
		return nil, err
	}

	opts.Dir = meta
	opts.ValueDir = data

	db, err := badgerdb.Open(opts)

	return &BadgerDB{
		db: db,
	}, err
}

func (b BadgerDB) Close() error {
	err := b.db.Close()
	if err != nil {
		log.Errorln(err.Error())
	}
	return err
}

func (b BadgerDB) Delete(key []byte) error {
	return b.db.Update(func(tx *badgerdb.Txn) error {
		err := tx.Delete(key)
		if err != nil {
			log.Errorf("badger delete error: %v", err)
		}
		return err
	})
}

func (b BadgerDB) Set(key []byte, val []byte) error {
	return b.db.Update(func(tx *badgerdb.Txn) error {
		err := tx.Set(key, val)
		if err != nil {
			log.Errorf("bager set error: %v", err)
		}
		return err
	})
}

func (b BadgerDB) Get(key []byte) (val []byte, err error) {

	err = b.db.View(func(tx *badgerdb.Txn) error {
		item, err := tx.Get(key)
		if err != nil {
			log.Errorf("badger get error: %v", err)
			return err
		}
		v, err := item.Value()
		if err != nil {
			log.Errorf("badger get error: %v", err)
			return err
		}

		val = make([]byte, len(v))
		copy(val, v)
		return nil
	})

	if err == badgerdb.ErrKeyNotFound {
		err = errors.ErrNotFound
	}

	return
}

func (b BadgerDB) Exists(key []byte) (exists bool, err error) {
	err = b.db.View(func(tx *badgerdb.Txn) error {
		_, err := tx.Get(key)
		if err != nil && err != badgerdb.ErrKeyNotFound {
			log.Errorf("badger exists error: %v", err)
			return err
		}

		exists = !(err == badgerdb.ErrKeyNotFound)
		return nil
	})

	return
}

// Pass count = -1 to get all elements starting from the provided index
func (b BadgerDB) Filter(prefix []byte, start int, count int) (result [][]byte, err error) {
	opt := badgerdb.DefaultIteratorOptions
	result = make([][]byte, 0, count)
	var buf []byte

	err = b.db.View(func(tx *badgerdb.Txn) error {
		it := tx.NewIterator(opt)
		defer it.Close()

		counter := 0 // Number of namespaces encountered

		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			counter++

			// Skip until starting index
			if counter < start {
				continue
			}

			b, err := it.Item().Value()
			if err != nil {
				log.Errorf("badger filter error: %v", err)
				return err
			}

			buf = make([]byte, len(b))
			n := copy(buf, b)
			result = append(result, buf[:n])

			if count > 0 && len(result) == count {
				break
			}
		}

		return nil
	})

	return result, err
}

func (b BadgerDB) List(prefix []byte) (result [][]byte, err error) {

	opt := badgerdb.DefaultIteratorOptions
	opt.PrefetchValues = false
	result = [][]byte{}
	var buf []byte

	err = b.db.View(func(tx *badgerdb.Txn) error {
		it := tx.NewIterator(opt)
		defer it.Close()

		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			key := it.Item().Key()

			buf = make([]byte, len(key))

			n := copy(buf, key)
			result = append(result, buf[:n])
		}

		return nil
	})

	return result, err
}
