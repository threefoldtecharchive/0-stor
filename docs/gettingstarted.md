## Installation

The server is go-getable:
```
go get -u github.com/zero-os/0-stor/cmd/zstordb
```

## Running the 0-stor server

Start the server with grpc listening on all interfaces and port `12345`:

```
zstordb --listen :12345 --data-dir /path/to/data --meta-dir /path/to/meta
```

Execute `zstordb help` for more information:

```
A generic object store server used by zero-os.

Usage:
  zstordb [flags]
  zstordb [command]

Available Commands:
  help        Help about any command
  version     Output the version information

Flags:
      --async-write           Enable asynchronous writes in BadgerDB.
      --data-dir string       Directory path used to store the data. (default ".db/data")
  -D, --debug                 Enable debug logging.
  -h, --help                  help for zstordb
      --max-msg-size int      Configure the maximum size of the message GRPC server can receive, in MiB (default 32)
      --meta-dir string       Directory path used to store the meta data. (default ".db/meta")
      --no-auth               Disable JWT authentication.
  -L, --listen string         Bind the server to the given host and port. Format has to be host:port, with host optional (default ":8080")
      --profile-addr string   Enables profiling of this server as an http service.

Use "zstordb [command] --help" for more information about a command.
```