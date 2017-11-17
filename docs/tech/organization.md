# Code organization

The 0-stor is composed of 3 layers:
- Database layer
- Manager layer
- Server interface

## Database layer

This is mostly located in the [db package](../../../server/db).  
The [db.go](/server/db/db.go) file defines an interface that any key-value store needs to implement to be usable by the 0-stor.

Currently the only implementation present is [badger](https://github.com/dgraph-io/badger) : see [badger package package](../../../server/db/badger)


## Manager layer

This is where the business logic of how we store the data in the key-value store. This layer also gives a unify API that the server interface can reuse.


The code is located in the [manager package](../../../server/manager)

## Server interface

We currently support a [grpc](../../../server) server interface:

This interface is defined and implemented in the [server package](./../../server/server.go)
