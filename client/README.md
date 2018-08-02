# 0-stor client   [![godoc](https://godoc.org/github.com/threefoldtech/0-stor/client?status.svg)](https://godoc.org/github.com/threefoldtech/0-stor/client)

[Specifications](specs)

API documentation : [https://godoc.org/github.com/threefoldtech/0-stor/client](https://godoc.org/github.com/threefoldtech/0-stor/client)


## Motivation

- Building a **secure** & **fast** object store client library with support for big files

## Basic idea

- Splitting a large file into smaller chunks.
- Compress each chunk using [snappy](https://github.com/google/snappy)
- Encrypt each chunk using Hash(content)
    - Hash functions supported: [blake2](https://blake2.net/)
    - Support for symmetric encryption
- Save each part in different 0-db server
- Replicate same chunk into different 0-db servers
- Save metadata about chunks and where they're in etcd server or other metadata db
- Assemble smaller chunks for a file upon retrieval
    - Get chunks location from metadata server
    - Assemble file from chunks

## Important

- 0-db server is JUST a simple key/value store
- splitting, compression, encryption, and replication is the responsibility of client
- you need at least etcd v3 if you want to use `etcd` as metadata DB storage

**Features**

- [Erasure coding](http://smahesh.com/blog/2012/07/01/dummies-guide-to-erasure-coding/)

**Reference list**

The `client.Write` method takes a third parameter other then the key and value, namely the reference list (`refList`).  
This reference list is also returned as the second value from the `client.Read` method.

As the 0-db server doesn't do anything with this list, it can be omitted and ignored if the client has no desire of using it.  
The reference list for example, can be used to allow the client to do deduplication.

***TODO: show example (https://github.com/threefoldtech/0-stor/issues/216)***

## Metadata

**Metadata format**
- Format for the metadata:
    ```
	type Chunk struct {
		Size   uint64    # Size of the chunk in bytes
		Key    []byte    # key used in the 0-stor
		Shards []string
	}
	type Meta struct {
		Epoch     int64  # creation epoch
		Key       []byte # key used in the 0-stor
		Chunks    []*Chunk # list of chunks of the files
		Previous  []byte   # Key to the previous metadata entry
		Next      []byte   # Key to the next metadata entry
		ConfigPtr []byte   # Key to the configuration used by the lib to set the data.
	}
    ```

**chunks**

Depending on the policy used to create the client, the data could be split into multiple chunks.  Which means that the metadata can be composed of minimum one up to n chunks.

Each chunks can then have one or multiple shards.

- If you use replication, each shards is the location of one of the replicate.
- If you use distribution, each shards is the location of one of the data or parity block.

**metadata linked list**

With `Previous` and `Next` fields in the metadata, we can build metadata linked list of sequential data
such as transaction log/history and block chain.
The linked list can then be used to walk over the data in forward or backward fashion.

Metadata linked list will be build if  user specify previous meta key when
calling `WriteWithMeta` or `WriteFWithMeta` methods.

## Getting started

- [Getting started](../cmd/zstor/README.md)

## Now into some technical details!

**Pre processing of data**

- Client does some preprocessing on each chunk of data before sending them to 0stor
- This is achieved by configuring a policy during client creation
- Supported Data Preprocessing:
    - [chunker](./components/chunker)
	- [compression](./components/compress/README.md)
    - [Hasher](./components/hash/README.md)
    - [encryption](./components/encrypt/README.md)
    - [distribution / erasure coding](./components/distribution/README.md)
    - [replication](./components/replication/README.md)

**walk over the metadata**

Thanks to our metadata linked list, we can walk over the linked list in forward and backward mode
using `Walk` or `WalkBack` API.

Those APIs returns `channel` of `WalkResult` which then can be iterated.
The WalkResult consist of these fields:

- key in metadata
- the metadata
- the data stored on 0-stor server
- error if exist

It can be used for example to reconstruct the stored sequential data.

## Using 0-stor client examples:

### Hello World

File: [/examples/hello_world/main.go](/examples/hello_world/main.go)

In this example, when we store the data, the data will be processed as follow:
plain data -> compress -> encrypt -> distribution/erasure encoding (which send to 0-stor server and write metadata)

When we get the data from 0-db server, the reverse process will happen:
distribution/erasure decoding (which reads metadata & Get data from 0-stor) -> decrypt -> decompress -> plain data.

To run this example, you need to run:
- 0-db server at port 12345
- 0-db server at port 12346
- 0-db server at port 12347
- etcd server at port 2379

Than you can run the example as follows:

```
go run examples/hello_world/main.go
```

Please check out the source code to see how this example works.

### Hello World: Config File Edition

File: [/examples/hello_config/main.go](/examples/hello_config/main.go)

In this file we are doing exactly the same,
and you'll need to run the same servers as were required before.

However this time we create the client, using a file-based config.

You can run the example as follows:

```
go run examples/hello_config/main.go
```

Please check out the source code to see how this example works.

## Configuration

Configuration file example can be found on [config.yaml](/cmd/zstor/config.yaml).

## Libraries

This client includes some components that can be used independently.
See [components](./components) directory for more details.

## CLI

A cli can be found in the [cli](./cmd/zstor) directory.

This command-line client includes a command to spawn it as a daemon.
