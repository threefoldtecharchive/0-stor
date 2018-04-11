# Simple 0-stor client cli

Simple cli to store file to 0-stor.

## Installation

```
 go get -u github.com/zero-os/0-stor/cmd/zstor
```

## Configuration

By default, the cli client will look for a `config.yaml` file in the directory the client is called from.  
If a custom filename and/or directory needs to be provided for the config file, it can be set with the `--config`/`-C` flag. E.g.:  
`zstor --config config/aConfigFile.yaml`

A config file should look something like this:
```yaml
password: mypass       # 0-db namespace password
namespace: namespace1  # 0-db namespace (required)
datastor: # required
  # the address(es) of a 0-db cluster (required0)
  shards: # required
    - 127.0.0.1:12345
    - 127.0.0.1:12346
    - 127.0.0.1:12347
    - 127.0.0.1:12348
  pipeline:
    block_size: 4096
    compression: # optional, snappy by default
      type: snappy # snappy is the default, other options: lz4, gzip
      mode: default # default is the default, other options: best_speed, best_compression
    encryption: # optional, disabled by default
      type: aes # aes is the default and only standard option
      private_key: ab345678901234567890123456789012
    distribution: # optional, disabled by default
      data_shards: 3
      parity_shards: 1
metastor: # required
  db: # required
    # the address(es) of an etcd server cluster
    endpoints:
      - 127.0.0.1:2379
      - 127.0.0.1:22379
      - 127.0.0.1:32379
  encoding: protobuf # protobuf is the default and only standard option
  encryption:
    type: aes # aes is the default and only standard option
    private_key: ab345678901234567890123456789012
```

Make sure to set the `namespace` to a valid 0-db namespace.
Also make sure the `data_shards` are set to existing addresses of 0-db server instances (**Warning** do not use one instance multiple times to make up a cluster)  
and that the `meta_shards` are set to existing addresses of etcd server instances.

The config used in this example will also do the following data processing when uploading a file:
- Chunk the file into smaller blocks
- Compress all the blocks using snappy
- Encrypt the blocks with the supplied `encryption_key`
- Erasure code the blocks over the `data_shards` (into 3 data shards and 1 parity shard).

## Commands
The CLI expose two group of commands, file and daemon. File group contains sub commands.

- file
  - `upload`: Upload a file to the 0-stor(s)
  - `download`: Download a file from the 0-stor(s)
  - `delete`: Delete a file from the 0-stor(s)
  - `metadata`: Print the metadata of a key
  - `repair`: Repair a file on the 0-stor(s)

### Start client daemon

```
zstor --config conf_file.yaml daemon --listen 127.0.0.1:8000
```

Start the daemon with  `127.0.0.1:8000`  as listen address

More information about the daemon can be found at the daemon [README](/daemon/api/grpc/README.md)

### Upload a file

```
zstor --config conf_file.yaml file upload data/my_file.file
```

Upload the file and use the file's name (`my_file.file`) as the 0-stor key.  
If a custom key needs to be provided, the `--key` or `-k` flag can be used to set the desired key:

```
zstor --conf conf_file.yaml file upload -k myFile data/my_file.file
```

Now the key for the file will be set to `myFile`.

You can also upload a file directly from the STDIN:

```
zstor --config conf_file.yaml file upload -k myFile < data/my_file.file
```

When uploading a file directly from the STDIN the key has to be given.

### Download a file

```
zstor --config conf_file.yaml file download myFile
```

This will get the value with key =`myFile`
from the 0-stor server(s) and output it on the STDOUT.

If you want to output the file directly to a file instead,
you can define the `--output`/`-o` flag:

```
zstor --config conf_file.yaml file download myFile --output downloaded.file
```

This will get the value with key=`myFile`
from 0-stor server(s) and write it to `downloaded.file` instead of the STDOUT.

### Read metadata of a file

```
zstor --config config_file.yaml file metadata myFile --json-pretty
```

This will print the metadata of the object with the key `myFile` in a prettified JSON format.
You can also print it as the default/compact JSON format using the `--json` flag.
If None of these flags are given the metadata will be printed in a custom human-readable format (close to YAML).

### Repair a file

```
zstor --config config_file.yaml file repair myFile
```
This will repair the file with the key `myFile` in the 0-stor

### Delete a file

```
zstor --config config_file.yaml file delete myFile
```
This will delete the file with the key `myFile` in the 0-stor