# Code organization

The 0-stor server is composed of 3 layers:
- [Database layer](#database-layer)
- [Manager layer](#manager-layer)
- [Server interface](#server-interface)

## Database layer

This is mostly located in the [db package][pkg_db] and its subpackages.  
The [db.go][code_db_interface] file defines an interface that any key-value store needs to implement to be usable by the 0-stor.

Currently the only production-ready implementation present is using [badger][badger] as the underlying DB technology.
See the [badger package][pkg_db_badger] for more information.

## Manager layer

This layer provides a unified KV storage API, using the underlying [DB Layer](#database-layer), that the server interface can (re)use. Its purpose is to have to only define the business logic of the KV storage once, regardless of the underlying [DB Layer](#database-layer).

See the [manager package][pkg_manager] for more information.

## Server interface

We currently only support a [GRPC][grpc] server interface.

This interface is defined and implemented in the [server package][code_server_interface].

[pkg_db]: /server/db
[pkg_db_badger]: /server/db/badger
[pkg_manager]: /server/manager

[code_db_interface]: /server/db/db.go
[code_server_interface]: /server/server.go

[grpc]: http://grpc.io
[badger]: https://github.com/dgraph-io/badger