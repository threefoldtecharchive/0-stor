package badger

import (
	"context"
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
	badgerdb "github.com/dgraph-io/badger"
	"github.com/zero-os/0-stor/server/errors"
)

const (
	// discardRatio represents the discard ratio for the badger GC
	// https://godoc.org/github.com/dgraph-io/badger#DB.RunValueLogGC
	discardRatio = 0.5

	// GC interval
	cgInterval = 10 * time.Minute
)

// DB implements the db.DB interace
type DB struct {
	db         *badgerdb.DB
	ctx        context.Context
	cancelFunc context.CancelFunc
}

// New creates new badger DB with default options
func New(data, meta string) (*DB, error) {
	opts := badgerdb.DefaultOptions
	opts.SyncWrites = true
	return NewWithOpts(data, meta, opts)
}

// NewWithOpts creates new badger DB with own options
func NewWithOpts(data, meta string, opts badgerdb.Options) (*DB, error) {
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

	ctx, cancel := context.WithCancel(context.Background())

	badger := &DB{
		db:         db,
		ctx:        ctx,
		cancelFunc: cancel,
	}

	go badger.runGC()

	return badger, err
}

// Close implements DB.Close
func (b *DB) Close() error {
	b.cancelFunc()

	err := b.db.Close()
	if err != nil {
		log.Errorln(err.Error())
	}
	return err
}

// Delete implements DB.Delete
func (b *DB) Delete(key []byte) error {
	return b.db.Update(func(tx *badgerdb.Txn) error {
		err := tx.Delete(key)
		if err != nil {
			log.Errorf("badger delete error: %v", err)
		}
		return err
	})
}

// Set implements DB.Set
func (b *DB) Set(key []byte, val []byte) error {
	return b.db.Update(func(tx *badgerdb.Txn) error {
		err := tx.Set(key, val)
		if err != nil {
			log.Errorf("badger set error: %v", err)
		}
		return err
	})
}

// Get implements DB.Get
func (b *DB) Get(key []byte) (val []byte, err error) {
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

// Exists implements DB.Exists
func (b *DB) Exists(key []byte) (exists bool, err error) {
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

// Filter implements DB.Filter
func (b *DB) Filter(prefix []byte, start int, count int) (result [][]byte, err error) {
	if count == 0 {
		return
	}

	opt := badgerdb.DefaultIteratorOptions
	var buf []byte

	err = b.db.View(func(tx *badgerdb.Txn) error {
		it := tx.NewIterator(opt)
		defer it.Close()

		var counter int // Number of namespaces encountered (only used for start cursor)

		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			// Skip until starting index
			if start > counter {
				counter++
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

			if len(result) == count {
				break
			}
		}

		return nil
	})

	return result, err
}

// List implements DB.List
func (b *DB) List(prefix []byte) (result [][]byte, err error) {
	opt := badgerdb.DefaultIteratorOptions
	opt.PrefetchValues = false
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

// collectGarbage runs the garbage collection for Badger backend db
func (b *DB) collectGarbage() error {
	if err := b.db.PurgeOlderVersions(); err != nil {
		return err
	}

	return b.db.RunValueLogGC(discardRatio)
}

// runGC triggers the garbage collection for the Badger backend db.
// Should be run as a goroutine
func (b *DB) runGC() {
	ticker := time.NewTicker(cgInterval)
	for {
		select {
		case <-ticker.C:
			err := b.collectGarbage()
			if err != nil {
				// don't report error when gc didn't result in any cleanup
				if err == badgerdb.ErrNoRewrite {
					log.Debugf("Badger GC: %v", err)
				} else {
					log.Errorf("Badger GC errored: %v", err)
				}
			}
		case <-b.ctx.Done():
			return
		}
	}
}
