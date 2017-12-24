# 0-stor

[![Build Status](https://travis-ci.org/zero-os/0-stor.png?branch=master)](https://travis-ci.org/zero-os/0-stor) [![GoDoc](https://godoc.org/github.com/zero-os/0-stor?status.svg)](https://godoc.org/github.com/zero-os/0-stor) [![codecov](https://codecov.io/gh/zero-os/0-stor/branch/master/graph/badge.svg)](https://codecov.io/gh/zero-os/0-stor) [![Go Report Card](https://goreportcard.com/badge/github.com/zero-os/0-stor)](https://goreportcard.com/report/github.com/zero-os/0-stor)

A Single device object store.

[link to group on telegram](https://t.me/joinchat/BrOCOUGHeT035il_qrwQ2A)

## Components

## Server

The 0-stor server is a generic object store that provide simple storage primitives, read, write, list, delete.

0-stor uses [badger](https://github.com/dgraph-io/badger) as the backend key value store. Badger allows storing the keys and the value onto separate devices. Because of this separation, the LSM (Log-Structured Merge) tree of keys can most of the time stay in memory. Typically the keys could be kept in memory and depending on the use case, the values could be served from an SSD or HDD.

### Installation

Install the 0-stor server

```
go get -u github.com/zero-os/0-stor/cmd/zstordb
```

### How to run the server

## Running the server

Here are the options of the server:

```
      --async-write           Enable asynchronous writes in BadgerDB.
      --data-dir string       Directory path used to store the data. (default ".db/data")
  -D, --debug                 Enable debug logging.
  -h, --help                  help for zstordb
      --max-msg-size int      Configure the maximum size of the message GRPC server can receive, in MiB (default 32)
      --meta-dir string       Directory path used to store the meta data. (default ".db/meta")
      --no-auth               Disable JWT authentication.
  -L, --listen string         Bind the server to the given host and port. Format has to be host:port, with host optional (default ":8080")
      --profile-addr string   Enables profiling of this server as an http service.
```

Start the server with listening on all interfaces and port 12345

```shell
zstordb --listen :12345 --data-dir /path/to/data --meta-dir /path/to/meta
```

## Client

The client contains all the logic to communicate with the 0-stor server.

The client provides some basic storage primitives to process your data before sending it to the 0-stor server:
- chunking
- compression
- encryption
- replication
- distribution/erasure coding

All of these primitives are configurable and you can decide how your data will be processed before being sent to the 0-stor.

### etcd

Other then a 0-stor server cluster, 0-stor clients also needs an [etcd](https://github.com/coreos/etcd) server cluster running to store it's metadata onto.

To install and run an etcd cluster, check out the [etcd documentation](https://github.com/coreos/etcd#getting-etcd).

### Client API

Client API documentation can be found in the godocs:

[![godoc](https://godoc.org/github.com/zero-os/0-stor/client?status.svg)](https://godoc.org/github.com/zero-os/0-stor/client)

### Client CLI

You can find a CLI for the client in `cmd/zerostorcli`.

To install
```
go get -u github.com/zero-os/0-stor/cmd/zerostorcli
```

### More documentation

You can find more information about the different components in the `/docs` folder of this repository:

* [Server docs](docs/README.md)
* [Client docs](client/README.md)
* [CLI docs](cmd/zerostorcli/README.md)
