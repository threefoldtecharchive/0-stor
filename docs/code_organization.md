# 0-stor code organization

The 0-stor codebase can be divided in 2 different projects, which together form 0-stor:

- [Client (zstor)](#client)
- [Daemon](#daemon)

## Client

The client side of 0-stor consists out of 2 sub clients and 1 master client. All 3 clients can be used separately, and the master client contains all 2 sub clients. All clients are also used in [a CLI tool, zstor][zstor].

The [datastor client][datastor_godocs] is used to interact with a 0-db server. The [metastor client][metastor_godocs] is used to store and manage metadata, used by the [master client][client_godocs].

The [master client][client_godocs] is the most high-level client, and can be used to store data, optionally/ideally split in chunks, and identified by metadata, all managed for you.

## Daemon

The daemon is used to expose the different clients, as discussed in the [client section](#client), over a [GRPC][grpc] interface to a light client, written in any language, from one and the same machine or from a remote location.

[zstor]: /cmd/zstor/README.md
[datastor_godocs]: https://godoc.org/github.com/threefoldtech/0-stor/client/datastor
[metastor_godocs]: https://godoc.org/github.com/threefoldtech/0-stor/client/metastor
[client_godocs]: https://godoc.org/github.com/threefoldtech/0-stor/client
[grpc]: https://grpc.io/
