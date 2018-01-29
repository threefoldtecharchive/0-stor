# 0-stor

[![Build Status](https://travis-ci.org/zero-os/0-stor.png?branch=master)](https://travis-ci.org/zero-os/0-stor) [![GoDoc](https://godoc.org/github.com/zero-os/0-stor?status.svg)](https://godoc.org/github.com/zero-os/0-stor) [![codecov](https://codecov.io/gh/zero-os/0-stor/branch/master/graph/badge.svg)](https://codecov.io/gh/zero-os/0-stor) [![Go Report Card](https://goreportcard.com/badge/github.com/zero-os/0-stor)](https://goreportcard.com/report/github.com/zero-os/0-stor) [![license](https://img.shields.io/github/license/zero-os/0-stor.svg)](https://github.com/zero-os/0-stor/blob/master/LICENSE)

A Single device object store.

[link to group on telegram](https://t.me/joinchat/BrOCOUGHeT035il_qrwQ2A)

## Minimum requirements

Development requirements | Notes
--- | ---
Go version | [**Go 1.8**][min-release-go] or any higher **stable** release (it is recommended to always use the latest Golang release)
ETCD Version | [**etcd 3.2.4**][min-release-etcd] or any higher **stable** release (only required when requiring a metastor client with ETCD as its underlying database)
protoc version | [**protoc 3.4.0** (protoc-3.4.0)][min-release-protoc] (only required when needing to regenerate any proto3 schemas)

Production requirements | Notes
--- | ---
ETCD Version | [**etcd 3.2.4**][min-release-etcd] or any higher **stable** release

Developed on Linux and MacOS, [CI Tested on Linux][ci-tested-travis]. Ready for usage in production on both Linux and MacOS.

While 0-stor probably works on Windows and FreeBSD, this is not officially supported nor tested. Should it not work out of the box and you require it to work for whatever reason, feel free to open [a pull request](https://github.com/zero-os/0-stor/pulls) for it.

[min-release-go]: (https://github.com/golang/go/releases/tag/go1.8)
[min-release-etcd]: (https://github.com/coreos/etcd/releases/tag/v3.2.4)
[min-release-protoc]: (https://github.com/google/protobuf/releases/tag/v3.4.0)
[ci-tested-travis]: https://travis-ci.org/zero-os/0-stor

## Components

For a quick introduction checkout the [intro docs](/docs/intro.md).

For a full overview check out the [code organization docs](/docs/code_organization.md).

## Server

The 0-stor server is a generic object store that provide simple storage primitives, read, write, list, delete.

0-stor uses [badger](https://github.com/dgraph-io/badger) as the backend key value store. Badger allows storing the keys and the value onto separate devices. Because of this separation, the LSM (Log-Structured Merge) tree of keys can most of the time stay in memory. Typically the keys could be kept in memory and depending on the use case, the values could be served from an SSD or HDD.

See [the server docs][/docs/server/server.md] for more information.

### Installation

Install the 0-stor server

```
go get -u github.com/zero-os/0-stor/cmd/zstordb
```

### How to run the server

## Running the server

Here are the options of the server:

```
      --async-write                  Enable asynchronous writes in BadgerDB.
      --data-dir string              Directory path used to store the data. (default ".db/data")
  -D, --debug                        Enable debug logging.
  -h, --help                         help for zstordb
  -j, --jobs int                     amount of async jobs to run for heavy GRPC server commands (default $NUM_OF_CPUS_TIMES_TWO)
  -L, --listen listenAddress         Bind the server to the given unix socket path or tcp address. Format has to be either host:port, with host optional, or a valid unix (socket) path. (default :8080)
      --max-msg-size int             Configure the maximum size of the message GRPC server can receive, in MiB (default 32)
      --meta-dir string              Directory path used to store the meta data. (default ".db/meta")
      --no-auth                      Disable JWT authentication.
      --profile-addr string          Enables profiling of this server as an http service.
      --profile-mode profileMode     Enable profiling mode, one of [cpu, mem, block, trace]
      --profile-output string        Path of the directory where profiling files are written (default ".")
      --tls-cert string              TLS certificate used for this server, paired with the given key
      --tls-key string               TLS private key used for this server, paired with the given cert
      --tls-key-pass string          Passphrase of the given TLS private key file, only required if that file is encrypted
      --tls-live-reload              Enable in order to support the live reloading of TLS Cert/Key file pairs, when signaling a SIGHUP signal
      --tls-max-version TLSVersion   Maximum supperted/accepted TLS version (default TLS12)
      --tls-min-version TLSVersion   Minimum supperted/accepted TLS version (default TLS11)
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
- replication or distribution/erasure coding

All of these primitives are configurable and you can decide how your data will be processed before being sent to the 0-stor.

### etcd

Other then a 0-stor server cluster, 0-stor clients also needs an [etcd](https://github.com/coreos/etcd) server cluster running to store it's metadata onto.

To install and run an etcd cluster, check out the [etcd documentation](https://github.com/coreos/etcd#getting-etcd).

> NOTE: it is possible to avoid the usage of etcd, and use a badger-backed metastor client instead. See http://godoc.org/github.com/zero-os/0-stor/client/metastor/db/badger for more information.

### Client API

Client API documentation can be found in the godocs:

[![godoc](https://godoc.org/github.com/zero-os/0-stor/client?status.svg)](https://godoc.org/github.com/zero-os/0-stor/client)

### Client CLI

You can find [a CLI for the client in `cmd/zstor`](cmd/zstor/README.md).

To install
```
go get -u github.com/zero-os/0-stor/cmd/zstor
```
