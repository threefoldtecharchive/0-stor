# zstordb design

The 0-stor server can be seen as 2 layers, and some extra sub-modules
which will aid the development of those layers. The two layers are:

+ [`db/badger`][godocs_db_badger] (database): it provides a simple storage API to create and manage data;
+ [`api/grpc`][godocs_api_grpc]: provides a [GRPC][grpc] API to manage objects and namespaces;

While we currently only have one implementation per layer,
[badger][badger] as the DB layer and [GRPC][grpc] as the API layer,
it should be possible to swap out an implementation in a layer without affecting the other layer. This should allows us to provide different database and/or APIs for our 0-stor server, should we (or others) ever desire this.

The server binary itself consists out of single "main" file, which handles all the flags/option parsing and creates all the layers accordingly, to eventually serve the API over a TCP port.

> See: [/cmd/zstordb/commands/root.go](/cmd/zstordb/commands/root.go)

On top of these two layers, we also have some other packages:

+ [`jwt`][godocs_jwt]: handles all the JWT logic (e.g. JWT verification);
+ [`fs`][godocs_fs]: utility code about the file system (e.g. [`func FreeSpace`][godocs_func_free_space]);
+ [`stats`][godocs_stats]: provides a simple API to track —–in-memory—– and fetch statistics;
+ [`encoding`][godocs_encoding]: provides the encoding/decoding logic to use and transform the raw byte data;

These packages are used in the two layers mentioned earlier.

### Layer Packages

#### Database

The database is the core and first layer of any 0-stor server.

All database implementations implement [the DB interface][godocs_interface_db].

As you can see, the database layer does really only manage raw/binary data, mapped to a given key. It does not care about what that data represents, neither does it have any idea about any higher level concept such as namespaces and objects.

It does also contain functionality to store some data using an auto-generated key within a given scope. Which is used to store all data objects within 0-stor, freeing the user/client from the burden of ensuring all data keys are unique, as to prevent any kind of conflict that would otherwise arise.

##### Badger

> Package: [`/server/db/badger`][godocs_db_badger]

A server Database implementation using [badger][badger].

##### Memory

> Package: [`/server/db/memory`][godocs_db_memory]

An in-memory database implementation to be used for testing purposes only.

#### API

The API is the second and public layer of any 0-stor server.

All API implementations implement [the Server interface][godocs_interface_server], which is used to serve the API (of a zstordb server) over a given network Listener.

It is up to the implementation to ensure that all functionality/methods are implemented and correct.

##### GRPC

> Package: [`/server/api/grpc`][godocs_api_grpc]

The only supported API implementation, using [GRPC][grpc] as the communication protocol and used to expose the 0-stor server to a client, using remote procedure calls.

[godocs_db_badger]: https://godoc.org/github.com/zero-os/0-stor/server/db/badger
[godocs_db_memory]: https://godoc.org/github.com/zero-os/0-stor/server/db/memory
[godocs_api_grpc]: https://godoc.org/github.com/zero-os/0-stor/server/api/grpc
[godocs_jwt]: https://godoc.org/github.com/zero-os/0-stor/server/jwt
[godocs_fs]: https://godoc.org/github.com/zero-os/0-stor/server/fs
[godocs_fs_func_free_space]: https://godoc.org/github.com/zero-os/0-stor/server/fs#FreeSpace
[godocs_stats]: https://godoc.org/github.com/zero-os/0-stor/server/stats
[godocs_encoding]: https://godoc.org/github.com/zero-os/0-stor/server/encoding
[godocs_interface_db]: https://godoc.org/github.com/zero-os/0-stor/server/db#DB
[godocs_interface_server]: https://godoc.org/github.com/zero-os/0-stor/server/api#Server

[grpc]: https://grpc.io
[badger]: http://github.com/dgraph-io/badger