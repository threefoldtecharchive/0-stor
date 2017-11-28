# 0-stor server

This spec is a bit different than other specs,
as at the time of writing this spec the 0-stor server already exists.
The aim of this spec isn't to rewrite the server from scratch,
but to simplify and improve it into a rock-solid 0-stor server.

Our design principles at hand that will guide is through the work to be done are:

+ Code should be as minimalistic and simple as possible;
+ Allocations should only happen when absolutely required;
+ Responses should (start to) be sent the callee (over GRPC) as soon as possible;

## Index

1. [Design](#design): the new/improved design of the 0-stor server:
    + 1.1. [Layer Packages](#layer-packages): the design of the two layer packages and their implementations as sub-packages:
      + 1.1.1. [Database](#database): the first (and core) server layer:
        + 1.1.1.1. [Badger](#badger): the production db implementation;
        + 1.1.1.2. [Memory](#memory): a db implementation for dev/test purposes;
      + 1.1.2. [API](#API): the second (last and public) server layer:
        + 1.1.2.1. [GRPC](#GRPC): the production API layer, exposing 0-stor server behind a [GRPC][grpc] interface;
    + 1.2. [Server Package](#server-package): the code `/server` package with;
    + 1.3. [Other packages](#other-packages): other server packages, defining the core of the business logic:
      + 1.3.1. [JWT](#jwt): contains all JWT-related logic;
      + 1.3.1. [FS](#fs): contains all local FileSystem related logic;
      + 1.3.1. [Stats](#stats): a global server statistics tracker;
      + 1.3.1. [Encoding](#encoding): provides all server encoding/decoding logic;
    + 1.4. [zstordb](#zstordb): the 0-stor server binary;
    + 1.5. [Final Notes](#final-notes): some final design notes;

## Design

The 0-stor server can be seen as 2 layers, and some extra sub-modules
which will aid the development of those layers. The two layers are:

+ `db/badger` (database): it provides a simple storage API to fetch/store data from/to a key;
+ `api/grpc`: provides a GRPC API to manage objects and namespaces;

While we currently only have 1 implementation per layer,
badger as the DB layer and GRPC as the api layer,
it should be possible to swap out an implementation in a layer without affecting the other layer.
This should allows us to provide different database and/or API's for our 0-stor server.

The server will simply be a main file that handles all the flags/option parsing,
create the layers accordingly and serves using the API layer.

On top of these two layer packages, we also have some other packages:

+ `jwt`: handles all the JWT logic (e.g. JWT verification);
+ `fs`: utility code about the file system (e.g. `func FreeSpace`);
+ `stats`: provides a simple API to track —–in-memory—– and fetch statistics;
+ `encoding`: provides the encoding/decoding logic to use and transform the raw byte data;

This is all the 0-stor server package provides.
The actual binary can be defined in a single file, `/cmd/zstordb/main.go`,
and does only handle flag/option parsing and the creation of the server layers accordingly,
to finally serve using the desired API layer.

### Layer Packages

#### Database

The database is the core and first layer of any 0-stor server.

All database implementations will have to implement the following `DB` interface:

```go
// location: /server/db/db.go

// DB defines the interface of a Database implementation,
// and will handle all logic for the database layer (one of two server layers).
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
    Update(key []byte, cb func(data []byte) ([]byte, error)) error

    // Delete deletes data from a database mapped to the given key.
    Delete(key []byte) error

    // ListItems lists all available key-value pairs,
    // which key equals or starts with the given prefix.
    // The items are returned over the returned channel.
    // The returned channel remains open until all items are returned,
    // or until the given context is done.
    // Each returned Item _has_ to be closed,
    // the channel won't receive a new Item until the previous returned Item has been Closed!
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

    // Close this item, freeing up its resources,
    // and making it invalid for further use.
    Close() error
}
```

As you can see, the database layer does really only manage raw data,
mapped to a given key. It does not care about what that data represents,
neither does it have any idea about any higher level concept such as namespaces and objects.

All Database-related public errors (e.g. ErrNotFound) are directly defined
in the database core package (`/server/db`), as the `errors` package no longer exist.

##### Badger

> Package: `/server/db/badger`

Some pointers on how to implement the badger implementation:

+ `func (db *DB) Get(key []byte) ([]byte, error)`:
    + ViewTransaction -> Get -> Return copied data
+ `func (db *DB) Exists(key []byte) (bool, error)`:
    + ViewTransaction -> Get -> Return (ErrKeyNotFound != err, err)
+ `func (db *DB) Set(key, data []byte) error`:
    + UpdateTransaction -> Set
+ `func (db *DB) Update(key []byte, cb func(data []byte) ([]byte, error)) error`:
    + UpdateTransaction -> Get
    + Call callback using uncopied data
    + Set with non-error callback data result
+ `func (db *DB) Delete(key []byte) error`:
    + UpdateTransaction -> Delete
+ `func (db *DB) ListItems(ctx context.Context, prefix []byte) (<-chan Item, error)`:
    + create output channel -> open ViewTransaction -> create iterator
    + for each iteration -> `item := &iteratorItem{it}`
    + send item over channel OR exit callback if context is done
+ `func (db *DB) Close() error`:
    + Stop Background Goroutines
    + Close internal badger DB

for the `db.Item` you can create following struct that will implement that interface:

```go
// itemIterator implements db.Item
type iteratorItem struct {
    // whenever `it == nil` we'll return ErrItemInvalid
    // it will be set to `nil` when `(iteratorItem).Close` is called
    it *badger.Iterator
}
```

+ `func (item iteratorItem) Key() []byte`:
  + if `item.it == nil` -> return ErrItemInvalid
  + `return item.it.Item().Key()`
+ `func (it iteratorItem) Data() ([]byte, error):
  + if `item.it == nil` -> return ErrItemInvalid
  + `return item.it.Item().Data()`
+ `func (it iteratorItem) Close() error`:
  + if `item.it == nil` -> return ErrItemInvalid
  + `item.it.Next()` (moves the iterator and makes the Value and Key of our previous iteration invalid)
  + `item.it = nil`

##### Memory

> Package: `/server/db/memory`

+ Can be implemented pretty much as the current memory.DB is;
+ Adapt the API where needed;
+ Create `struct iteratorItem { key, data []byte }
  + both the key and value should be copied when constructing the iteratorItem;

#### API

The API is the second and public layer of any 0-stor server.

All API implementations will have to implement the following `API` interface:

```go
// location: /server/api/api.go

// API defines the interface of an api implementation (second and last layer),
// and will compose all 0-stor business logic and exposing it
// using a protocol of choice over a given local network address.
type API interface {
    // Bind binds the API to the given address,
    // as to serve the API and execute the business logic through that API.
    // This method returns the actual (final) address the API is bound onto.
    // This method is non-blocking and should return immediatly.
    Bind(address string) string

    // Close all open resources (including the embedded db.DB object)
    Close()
}
```

As you see the interface is really simple for this one,
as all the magic happens in the bindings to the given (network) address.
That does mean that we do also not validate (on compile time) if a given API implementation
implements the full API. This is on purpose and it is up to the API implementor
to ensure and communicate that the desired functionality is exposed.
The specific API should be in detail documented within the package where the API implementation lives.

##### GRPC

> Package: `/server/api/grpc`

+ All existing code in the following files has to move to the GRPC package:
  + https://github.com/zero-os/0-stor/blob/master/server/server.go
  + https://github.com/zero-os/0-stor/blob/master/server/object_api.go
  + https://github.com/zero-os/0-stor/blob/master/server/namespace_api.go
+ The code has to be cleaned up and properly documented;
+ The code can be improved, and will also have to be adapted to use the new API;
+ An API object will have to be implemented to provide a GRPC API Object (reusing a lot of the `server.go` code);

### Server package

This package defines some data structures shared among other packages:

```go
type (
    Object struct {
        // CRC32 checksum of the data
        CRC uint32
        // Data in its raw encoded form
        Data []byte
    }

    Namespace struct {
        // Reserved (total) space in this namespace (in bytes)
        Reserved uint64
        // Label (or name) of the namespace
        Label []byte
    }

    StoreStat struct {
        // Available disk space in bytes
        SizeAvailable uint64 
        // Space used in bytes
        SizeUsed uint64
    }
)
```

These structures do not have any logic attached to them,
and act as a coherent collection of data.
Further would it be a good idea if these data structures
are passed by value, as it will make everything safer and easier to use.
Safer to use because there is no deref-panic possible,
and easier to use as the function never has to check for a `nil`-reference.

For now there are also no public functions in this package.

There are however also a couple of constants:

```go
const (
    // Prefix used for all Namespace keys
    PrefixNamespace = "@"

    // Key used to store the global store stats in the DB
    KeyGlobalStoreStats  = "$"
)
```

### Other Packages

#### JWT

The JWT package has already been specced out and is being worked on already.
See https://github.com/zero-os/0-stor/issues/310 for more information.

#### FS

We can simply rename and move `/server/disk/disk.go` to `/server/fs/space.go`,
adapt the package name in that single Go file, update and improve the documentation and be done with it.

#### Stats

The stats package can remain as it is for now.

#### Encoding

Provides encoding and decoding logic for the data structures available in the `/server` package.
In a functional way it will allow you to encode/decode objects, namespaces, reference lists and stats.
On top of that it will allow you to append to reference lists without having to decode it.

The specific details of the implementation aren't really too important.
However here is how I expect the API to be more or less:

```go
// Encode an object to to a byte representation.
// MAKE SURE TO DOCUMENT THE ENCODING FORMAT
func EncodeObject(obj server.Object) ([]byte, error) {}
// Decode a byte slice as an Object (if possible)
func DecodeObject(data []byte) (server.Object, error) {}

// Encode a namespace to to a byte representation.
// MAKE SURE TO DOCUMENT THE ENCODING FORMAT
func EncodeNamespace(ns server.Namespace) ([]byte, error) {}
// Decode a byte slice as a namespace (if possible)
func DecodeNamespace(data []byte) (server.Namespace, error) {}

// Encode store stats to to a byte representation.
// MAKE SURE TO DOCUMENT THE ENCODING FORMAT
func EncodeStoreStats(stats server.StoreStat) ([]byte, error) {}
// Decode a byte slice as store stats (if possible)
func DecodeStoreStats(data []byte) (server.StoreStat, error) {}

// Encode a reference list to a semi-bencoded byte representation.
// MAKE SURE TO DOCUMENT THE ENCODING FORMAT
// For the encoding the following can be done:
//    + write each string one by one in the bencode format:
//        + binary.Write(buf, binary.LittleEndian, len(str))
//        + buf.WriteByte(':')
//        + binary.Write(buf, binary.LittleEndian, []byte(str))
func EncodeReferenceList(list []string) ([]byte, error) {}
// Decode a semi-bencoded byte slice as a reference list
// For the decoding the following can be done:
//   + read each string one by one, by decoding the bencode strings:
//        + binary.Read(buf, binary.LittleEndian, &length)
//        + if `buf.ReadByte() != ':'` -> ERROR
//        + binary.Read(buf, binary.LittleEndian, &raw[:length])
//        + refList = append(refList, string(raw))
func DecodeReferenceList(data []byte) ([]string, error) {}

// There is no seperate function for Appending a ReferenceList,
// as it is guranteed that you can safely append two encoded lists, 
// such that the following works:
//    listData, _ := EncodeReferenceList(refList)
//    data = append(data, listData)

// always make sure that the both during encoding and
// decoding the input is validated!
// never trust user input(!) and avoid bugs
// such as repported in https://github.com/zero-os/0-stor/issues/307
```

It is however left to the implementer to decide whether or not the proposed API needs
some modifications here and there. Or to extend the API where needed.
However, KISS!!!

### zstordb

> defined as a single-file binary: `/cmd/zstordb/main.go` (the 0-Stor server binary)

+ Parse flags/options, version and app execution code;
   + should we decide to use `spf13/cobra` we'll have to redefine the flag/option and command/execute code;
   + if we stick with `codegangsta/cli` can pretty much copy that code from the already existing code base;
   + see https://github.com/zero-os/0-stor/issues/318 for more information about this;
+ Create (Badger) DB Layer;
+ Create (GRPC) API layer using the earlier created (Badger) DB Layer;
+ Do the ensureStoreStat thing as we do now already as well, we would now implement it however as:
   + `globStats.available = fs.FreeSpace(dataPath)`
   + `for item := range kv.List(context.Background(), []byte(server.PrefixNamespace))`:
       + `ns := encoding.DecodeNamespace(kv.Get(item.Value()))`
       + `globStats.totalReserved += ns.Reserved`
   + `globStats.available -= globStats.totalReserved`
   + if `available <= 0` -> return error
   + kv.Set(server.KeyGlobalStoreStats, encoding.EncodeStoreStats(globStats))
   + log space reserved/available

### Final Notes

+ All non-used code should be deleted;
+ All commented-out code should be deleted (e.g. old reservation code);
+ Document all code properly;
+ When moving/copying/reusing existing code, make sure:
  + to improve the code where needed;
  + do add any missing documentation and improve existing documentation if needed;
  + stay critical and don't be afraid to reimplement something if needed;
+ Provide unit tests for all different functions and packages of the 0-stor server code;
+ The `config`, `errors` and `manager` packages should be deleted;

[grpc]: http://grpc.io