package db

import (
	"context"
	"errors"
)

// Constant DB errors
var (
	ErrNotFound   = errors.New("key not found")
	ErrNilKey     = errors.New("nil-key not allowed")
	ErrClosedItem = errors.New("item is already closed")
	ErrConflict   = errors.New("update conflicted, try again")
)

// UpdateCallback is the type of callback to be used,
// when wanting to update (race-condition free) an item in a database.
type UpdateCallback func(origData []byte) (newData []byte, err error)

// DB interface is the interface defining how to interact with the key value store.
// DB is threadsafe
type DB interface {
	// Get fetches data from a database mapped to the given key.
	// The returned data is allocated memory that can be used
	// and stored by the callee.
	Get(key []byte) ([]byte, error)

	// Exists checks if data exists in a database for the given key.
	// and returns true as the first return parameter if so.
	Exists(key []byte) (bool, error)

	// Set stores data in a database mapped to the given key.
	Set(key, data []byte) error

	// Update allows you to overwrite data, mapped to the given key,
	// by loading it in-memory and allowing for manipulation using a callback.
	// The callback receives the original data and is expected to return the modified data,
	// which will be stored in the database using the same key.
	// If the given callback returns an error it is returned from this function as well.
	// The returned data in the callback is only valid within the scope of that callback,
	// and should be copied if you want to retain the data beyond the scope of the callback.
	// When the requested key couldn't be found, the callback will receive a nil slice.
	Update(key []byte, cb UpdateCallback) error

	// Delete deletes data from a database mapped to the given key.
	Delete(key []byte) error

	// ListItems lists all available key-value pairs,
	// which key equals or starts with the given prefix.
	// The items are returned over the returned channel.
	// The returned channel remains open until all items are returned,
	// or until the given context is done.
	// Each returned Item _has_ to be closed,
	// the channel won't receive a new Item until the previous returned Item has been Closed!
	// The prefix is optional, and all items will be returned if no prefix is given.
	//
	// NOTE:
	// it is not guaranteed that all items returned are from the same generation
	// as when the ListItems was called. Meaning that some keys might be updated,
	// in-between the time this function is called and the item is returned.
	// Or if the key was deleted, one key that existed when this function was called,
	// might not be returned at all.
	ListItems(ctx context.Context, prefix []byte) (<-chan Item, error)

	// Close the DB connection and any other resources.
	Close() error
}

// Item is returned during iteration. Both the Key() and Value() output
// is only valid until Close is called.
// Every returned item has to be closed.
type Item interface {
	// Key returns the key.
	// Key is only valid as long as item is valid.
	// If you need to use it outside its validity, please copy it.
	Key() []byte

	// Value retrieves the value of the item.
	// The returned value is only valid as long as item is valid,
	// So, if you need to use it outside, please parse or copy it.
	Value() ([]byte, error)

	// Error retrieves the error,
	// which occurred while trying to fetch this item.
	Error() error

	// Close this item, freeing up its resources,
	// and making it invalid for further use.
	Close() error
}

// ErrorItem is an Item implementation,
// which can be used for returned items,
// that couldn't be fetched due to an error.
type ErrorItem struct {
	Err error
}

// Key implements Item.Key
func (item *ErrorItem) Key() []byte { return nil }

// Value implements Item.Value
func (item *ErrorItem) Value() ([]byte, error) { return nil, item.Err }

// Error implements Item.Error
func (item *ErrorItem) Error() error { return item.Err }

// Close implements Item.Close
func (item *ErrorItem) Close() error { return item.Err }

var (
	_ Item = (*ErrorItem)(nil)
)
