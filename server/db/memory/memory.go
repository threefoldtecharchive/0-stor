package memory

import (
	"bytes"
	"context"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/zero-os/0-stor/server/db"
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
	if key == nil {
		return nil, db.ErrNilKey
	}

	mdb.mux.RLock()
	val, exists := mdb.m[string(key)]
	if !exists {
		mdb.mux.RUnlock()
		return nil, db.ErrNotFound
	}

	b := make([]byte, len(val))
	copy(b, val)
	mdb.mux.RUnlock()

	return b, nil
}

// Exists implements DB.Exists
func (mdb *DB) Exists(key []byte) (bool, error) {
	if key == nil {
		return false, db.ErrNilKey
	}

	mdb.mux.RLock()
	_, exists := mdb.m[string(key)]
	mdb.mux.RUnlock()
	return exists, nil
}

// Set implements DB.Set
func (mdb *DB) Set(key []byte, value []byte) error {
	if key == nil {
		return db.ErrNilKey
	}

	b := make([]byte, len(value))
	copy(b, value)
	mdb.mux.Lock()
	mdb.m[string(key)] = b
	mdb.mux.Unlock()
	return nil
}

// Delete implements DB.Delete
func (mdb *DB) Delete(key []byte) error {
	if key == nil {
		return db.ErrNilKey
	}

	mdb.mux.Lock()
	delete(mdb.m, string(key))
	mdb.mux.Unlock()
	return nil
}

// Update implements interface DB.Update
func (mdb *DB) Update(key []byte, cb db.UpdateCallback) error {
	if cb == nil {
		panic("(*DB).Update expects a non-nil UpdateCallback")
	}
	if key == nil {
		return db.ErrNilKey
	}

	mdb.mux.Lock()
	defer mdb.mux.Unlock()

	v := mdb.m[string(key)]
	val := make([]byte, len(v))
	copy(val, v)

	v, err := cb(val)
	if err != nil {
		log.Errorf("(*DB).Update callback returned an error: %v\n", err)
		return err
	}

	val = make([]byte, len(v))
	copy(val, v)
	mdb.m[string(key)] = val
	return nil
}

// ListItems implements interface DB.ListItems
func (mdb *DB) ListItems(ctx context.Context, prefix []byte) (<-chan db.Item, error) {
	if ctx == nil {
		panic("(*DB).ListItems expects a non-nil context.Context")
	}

	// already read-lock, in the main routine,
	// such that for sure the channel has read-access as soon as it starts,
	// and this way nobody can be writing after this call returned (unless there are no keys found)
	mdb.mux.RLock()

	ch := make(chan db.Item, 1)
	go func() {
		defer mdb.mux.RUnlock() // unlock (our) read-lock at the end of this goroutine)

		// close channel when we return
		// (early or because all pairs have been fetched)
		// this is important so that the channel can be used as a range iterator.
		defer close(ch)

		// because of the fact that we want to read-lock until we're finished,
		// we only want to stop a range iteration of a pair,
		// until the returned item has actually been closed.
		closeCh := make(chan struct{}, 1)

		// go through all items
		for k, val := range mdb.m {
			select {
			case <-ctx.Done():
				return

			default:
				key := []byte(k)
				if prefix != nil && !bytes.HasPrefix(key, prefix) {
					continue
				}

				item := &Item{
					key:  key,
					val:  val,
					done: func() { closeCh <- struct{}{} },
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
					return // return early
				}
			}
		}
	}()

	return ch, nil
}

// Close implements DB.Close
func (mdb *DB) Close() error {
	mdb.mux.Lock()
	mdb.m = make(map[string][]byte)
	mdb.mux.Unlock()
	return nil
}

// Item contains key and value of a badger item
type Item struct {
	key  []byte
	val  []byte
	done func()
}

// Key implements interface Item.Key
func (item *Item) Key() []byte {
	return item.key
}

// Value implements interface Item.Value
func (item *Item) Value() ([]byte, error) {
	if item.key == nil {
		return nil, db.ErrClosedItem
	}

	return item.val, nil
}

// Error implements interface Item.Error
func (item *Item) Error() error { return nil }

// Close implements interface Item.Close
func (item *Item) Close() error {
	if item.key == nil {
		return db.ErrClosedItem
	}

	// make this item unusable
	item.key, item.val = nil, nil

	// notifies the DB (item owner),
	// that the item is no longer needed by the user,
	// and we can move on to the next item or finish the update.
	item.done()
	item.done = nil

	return nil
}

// ensure DB/Item interfaces are implemented
var (
	_ db.DB   = (*DB)(nil)
	_ db.Item = (*Item)(nil)
)
