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

Use "zstordb [command] --help" for more information about a command.
```