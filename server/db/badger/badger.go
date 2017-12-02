package badger

import (
	"context"
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
	badgerdb "github.com/dgraph-io/badger"
	"github.com/zero-os/0-stor/server/db"
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
		log.Errorf("Meta dir %q couldn't be created: %v", meta, err)
		return nil, err
	}

	if err := os.MkdirAll(data, 0774); err != nil {
		log.Errorf("Data dir %q couldn't be created: %v", data, err)
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

// Get implements DB.Get
func (bdb *DB) Get(key []byte) ([]byte, error) {
	if key == nil {
		return nil, db.ErrNilKey
	}

	var val []byte
	err := bdb.db.View(func(tx *badgerdb.Txn) error {
		item, err := tx.Get(key)
		if err != nil {
			return err
		}
		v, err := item.Value()
		if err != nil {
			return err
		}

		val = make([]byte, len(v))
		copy(val, v)
		return nil
	})
	if err != nil {
		return nil, mapBadgerError(err)
	}
	return val, nil
}

// Exists implements DB.Exists
func (bdb *DB) Exists(key []byte) (bool, error) {
	if key == nil {
		return false, db.ErrNilKey
	}

	var exists bool
	err := bdb.db.View(func(tx *badgerdb.Txn) error {
		_, err := tx.Get(key)
		if err != nil {
			if err == badgerdb.ErrKeyNotFound {
				return nil
			}
			return err
		}

		exists = true
		return nil
	})
	if err != nil {
		return false, mapBadgerError(err)
	}
	return exists, nil
}

// Set implements DB.Set
func (bdb *DB) Set(key []byte, val []byte) error {
	if key == nil {
		return db.ErrNilKey
	}

	err := bdb.db.Update(func(tx *badgerdb.Txn) error {
		err := tx.Set(key, val)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return mapBadgerError(err)
	}
	return nil
}

// Update implements interface DB.Update
func (bdb *DB) Update(key []byte, cb db.UpdateCallback) error {
	if cb == nil {
		panic("(*DB).Update expects a non-nil UpdateCallback")
	}
	if key == nil {
		return db.ErrNilKey
	}

	err := bdb.db.Update(
		func(tx *badgerdb.Txn) error {
			var val []byte
			if item, err := tx.Get(key); err == nil {
				v, err := item.Value()
				if err != nil {
					return err
				}
				val = make([]byte, len(v))
				copy(val, v)
			} else if err != badgerdb.ErrKeyNotFound {
				return err
			}

			val, err := cb(val)
			if err != nil {
				return err
			}
			err = tx.Set(key, val)
			if err != nil {
				return err
			}

			return nil
		},
	)
	if err != nil {
		return mapBadgerError(err)
	}
	return nil
}

// Delete implements DB.Delete
func (bdb *DB) Delete(key []byte) error {
	if key == nil {
		return db.ErrNilKey
	}

	err := bdb.db.Update(func(tx *badgerdb.Txn) error {
		err := tx.Delete(key)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return mapBadgerError(err)
	}
	return nil
}

// ListItems implements interface DB.ListItems
func (bdb *DB) ListItems(ctx context.Context, prefix []byte) (<-chan db.Item, error) {
	if ctx == nil {
		panic("(*DB).ListItems expects a non-nil context.Context")
	}

	ch := make(chan db.Item, 1)

	go func() {
		defer close(ch)

		// used to wait until item is closed
		closeCh := make(chan struct{}, 1)

		err := bdb.db.View(
			func(tx *badgerdb.Txn) error {
				opt := badgerdb.DefaultIteratorOptions
				opt.PrefetchValues = false
				it := tx.NewIterator(opt)
				defer it.Close()

				// define predicate to use for the for loop
				var pred func() bool
				if prefix != nil {
					it.Seek(prefix)
					pred = func() bool { return it.ValidForPrefix(prefix) }
				} else {
					it.Rewind()
					pred = it.Valid
				}

				for ; pred(); it.Next() {
					item := &Item{
						item:  it.Item(),
						close: func() { closeCh <- struct{}{} },
					}

					// send item,
					// because `ch` is (unary-)buffered, this line can never be dead-locked
					ch <- item

					// wait until either the item is closed,
					// or until the context finished somehow early (due to an error?!)
					select {
					case <-closeCh:
					case <-ctx.Done():
						// ensure we close item, so it becomes unusable
						err := item.Close()
						log.Warningf("context closed before item is closed, force-closing item (err: %v)", err)
						return nil // return early
					}
				}

				return nil
			},
		)
		if err != nil {
			err = mapBadgerError(err)
			ch <- &db.ErrorItem{Err: err}
		}
	}()

	return ch, nil
}

// Close implements DB.Close
func (bdb *DB) Close() error {
	bdb.cancelFunc()

	err := bdb.db.Close()
	if err != nil {
		return mapBadgerError(err)
	}

	return nil
}

// collectGarbage runs the garbage collection for Badger backend db
func (bdb *DB) collectGarbage() error {
	if err := bdb.db.PurgeOlderVersions(); err != nil {
		return err
	}

	return bdb.db.RunValueLogGC(discardRatio)
}

// runGC triggers the garbage collection for the Badger backend db.
// Should be run as a goroutine
func (bdb *DB) runGC() {
	ticker := time.NewTicker(cgInterval)
	for {
		select {
		case <-ticker.C:
			err := bdb.collectGarbage()
			if err != nil {
				// don't report error when gc didn't result in any cleanup
				if err == badgerdb.ErrNoRewrite {
					log.Debugf("Badger GC: %v", err)
				} else {
					log.Errorf("Badger GC failed: %v", err)
				}
			}
		case <-bdb.ctx.Done():
			return
		}
	}
}

// map badger errors, if we know about them
func mapBadgerError(err error) error {
	switch err {
	case badgerdb.ErrKeyNotFound:
		return db.ErrNotFound
	case badgerdb.ErrConflict:
		return db.ErrConflict
	case badgerdb.ErrEmptyKey:
		return db.ErrNilKey
	default:
		return err
	}
}

// Item contains key and value of a badger item
type Item struct {
	item  *badgerdb.Item
	close func()
}

// Key implements interface Item.Key
func (item *Item) Key() []byte {
	if item.item == nil {
		return nil
	}

	return item.item.Key()
}

// Value implements interface Item.Value
func (item *Item) Value() ([]byte, error) {
	if item.item == nil {
		return nil, db.ErrClosedItem
	}
	return item.item.Value()
}

// Error implements interface Item.Error
func (item *Item) Error() error { return nil }

// Close implements interface Item.Close
func (item *Item) Close() error {
	if item.item == nil {
		return db.ErrClosedItem
	}

	// notify master iterator it can continue
	item.close()

	// make item useless
	item.item, item.close = nil, nil
	return nil
}

// ensure DB/Item interfaces are implemented
var (
	_ db.DB   = (*DB)(nil)
	_ db.Item = (*Item)(nil)
)
