# 0-stor

[![Build Status](https://travis-ci.com/threefoldtech/0-stor.svg?branch=master)](https://travis-ci.com/threefoldtech/0-stor) [![GoDoc](https://godoc.org/github.com/threefoldtech/0-stor?status.svg)](https://godoc.org/github.com/threefoldtech/0-stor) [![codecov](https://codecov.io/gh/threefoldtech/threefoldtech/branch/master/graph/badge.svg)](https://codecov.io/gh/threefoldtech/0-stor) [![Go Report Card](https://goreportcard.com/badge/github.com/threefoldtech/0-stor)](https://goreportcard.com/report/github.com/threefoldtech/0-stor) [![license](https://img.shields.io/github/license/threefoldtech/0-stor.svg)](https://github.com/threefoldtech/blob/master/LICENSE)

0-stor is client library to process and store data to 0-db server.

[link to group on telegram](https://t.me/joinchat/BrOCOUGHeT035il_qrwQ2A)

## Minimum requirements

| Requirements   | Notes                                                                                                                     |
| -------------- | ------------------------------------------------------------------------------------------------------------------------- |
| Go version     | [**Go 1.8**][min-release-go] or any higher **stable** release (it is recommended to always use the latest Golang release) |
| protoc version | [**protoc 3.4.0** (protoc-3.4.0)][min-release-protoc] (only required when needing to regenerate any proto3 schemas)       |


Developed on Linux and MacOS, [CI Tested on Linux][ci-tested-travis]. Ready for usage in production on both Linux and MacOS.

While 0-stor probably works on Windows and FreeBSD, this is not officially supported nor tested. Should it not work out of the box and you require it to work for whatever reason, feel free to open [a pull request](https://github.com/threefoldtech/0-stor/pulls) for it.

[min-release-go]: (https://github.com/golang/go/releases/tag/go1.8)
[min-release-etcd]: (https://github.com/coreos/etcd/releases/tag/v3.2.4)
[min-release-protoc]: (https://github.com/google/protobuf/releases/tag/v3.4.0)
[ci-tested-travis]: https://travis-ci.org/threefoldtech/0-stor

## Components

For a quick introduction checkout the [intro docs](/docs/intro.md).

For a full overview check out the [code organization docs](/docs/code_organization.md).

## Server

0-stor uses 0-db as storage server.

See [0-db page](https://github.com/rivine/0-db) for more information.

0-db server need to run in [diret mode](https://github.com/rivine/0-db#direct-key).


## Client

The client contains all the logic to communicate with the 0-db servers.

The client provides some basic storage primitives to process your data before sending it to the 0-db servers:
- chunking
- compression
- encryption
- replication or distribution/forward looking error correcting codes

All of these primitives are configurable and you can decide how your data will be processed before being sent to the 0-stor.

### Metadata

Client's [Write](https://godoc.org/github.com/threefoldtech/0-stor/client#Client.Write) returns metadata that need to be stored in safe place for future data
retrieval.
If 0-stor client created with metadata storage, then the metadata is going to stored on the Write operation.

#### Provided metadata DB storage

0-stor also provides two metadata storage packages to be used by user:
- [etcd](https://godoc.org/github.com/threefoldtech/0-stor/client/metastor/db/etcd)
- [badger](https://godoc.org/github.com/threefoldtech/0-stor/client/metastor/db/badger)

Here is the example to use `etcd` storage
```go
// creates metadata DB storage
etcdDB, err := etcd.New([]string{"127.0.0.1:2379"})
if err != nil {
	log.Fatal(err)
}

// creates metadata client with default encryption using the given key as private key
metaCli, err := metastor.NewClient("mynamespace", etcdDB, "ab345678901234567890123456789012")
if err != nil {
	log.Fatal(err)
}

// creates 0-stor client
c, err := client.NewClientFromConfig(config, metaCli, -1) // use default job count
if err != nil {
	log.Fatal(err)
}
```

User could also use `badger` as metadata DB storage by replacing
```go
etcdDB, err := etcd.New([]string{"127.0.0.1:2379"})
```
line above with the respective `badger` code, see [badger godoc](https://godoc.org/github.com/threefoldtech/0-stor/client/metastor/db/badger) for more details.

#### Own metadata DB storage

User could also use own implementation of metadata DB storage by implementing
the [DB interface](https://godoc.org/github.com/threefoldtech/0-stor/client/metastor/db#DB).

And then replace the
```go
etcdDB, err := etcd.New([]string{"127.0.0.1:2379"})
```
line above with the code to creates the metadata DB storage.


### Client API

Client API documentation can be found in the godocs:

[![godoc](https://godoc.org/github.com/threefoldtech/0-stor/client?status.svg)](https://godoc.org/github.com/threefoldtech/0-stor/client)

### Client CLI

You can find [a CLI for the client in `cmd/zstor`](cmd/zstor/README.md).

To install
```
go get -u github.com/threefoldtech/0-stor/cmd/zstor
```


# Repository Owners:

* [zaibon](https://github.com/zaibon)
