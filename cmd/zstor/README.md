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
organization: <IYO organization>    #itsyou.online organization of the 0-stor
namespace: <IYO namespace>          #itsyou.online namespace of the 0-stor
iyo_app_id: <an IYO app ID>         #itsyou.online app/user id
iyo_app_secret: <an IYO app secret> #itsyou.online app/user secret
# the address(es) of 0-stor data cluster 
data_shards:
    - 127.0.0.1:12345
    - 127.0.0.1:12346
    - 127.0.0.1:12347
    - 127.0.0.1:12348
# the address(es) of etcd server(s) for the metadata
meta_shards:
    - http://127.0.0.1:2379
    - http://127.0.0.1:22379
    - http://127.0.0.1:32379

block_size: 4096

replication_nr: 4
replication_max_size: 4096

distribution_data: 3
distribution_parity: 1

compress: true
encrypt: true
encrypt_key: ab345678901234567890123456789012
```

Make sure to set the `organization` and `namespace` to an existing [ItsYou.Online][iyo] organization that looks like : `organization`.0stor.`namespace`  
Also make sure the `data_shards` are set to existing addresses of 0-stor server instances (**Warning** do not use one instance multiple times to make up a cluster)  
and that the `meta_shards` are set to existing addresses of etcd server instances.

The config used in this example will also do the following data processing when uploading a file:
- Chunk the file into smaller blocks
- Compress all the blocks using snappy
- Encrypt the blocks with the supplied `encryption_key`
- Erasure code the blocks over the `data_shards` (into 3 data shards and 1 parity shard).

## Commands
The CLI expose three group of commands, file, namespace, daemon. Each group except the daemon contains sub commands.

- file
  - `upload`: Upload a file to the 0-stor(s)
  - `download`: Download a file from the 0-stor(s)
  - `delete`: Delete a file from the 0-stor(s)
  - `metadata`: Print the metadata of a key
  - `repair`: Repair a file on the 0-stor(s)
- namespace
  - `create`: Create a namespace by creation the required sub-organization on [ItsYou.Online][iyo]
  - `delete`: Delete a namespace by deleting the sub-organizations.
  - `permission get`: Print the permission of a user for a namespace
  - `permission set`: Set the permission on a namespace for a user


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

### Create a namespace

```
zstor --config conf_file.yaml namespace create namespace_test
```

This command will create the required sub-organization on [ItsYou.online][iyo].
This command uses the `organization` field from the configuration file. If the organization is set to `myorg` the created sub-org will be:
- `myorg.0stor.namespace_test`
- `myorg.0stor.namespace_test.read`
- `myorg.0stor.namespace_test.write`
- `myorg.0stor.namespace_test.delete`

### Delete a namespace
```
zstor --config conf_file.yaml namespace delete namespace_test
```

This command will delete the organization `myorg.0stor.namespace_test` and all its sub-organization

### Set rights of a user into a namespace

```
zstor --config conf_file.yaml namespace permission set johndoe@email.com namespace_test -rwd
```

This command will authorize the user with it's [ItsYou.online][iyo] user ID,
in this case the email address `johndoe@email.com` into the namespace `namespace_test`
with the right read, write and delete.

For [ItsYou.online's][iyo] user ID, following can be used reliably with 0-stor: email address

The different rights that can be set are:
- read
- write
- delete
- admin (has all the right plus can inspect namespaces stats and reservations)

Note that once you give access to a user, he will receive an invitation from ItsYou.online to join the organization. He needs to accept the invitation before being able to access the 0-stor(s).

To remove some right, just re-execute the command with rights the user would have after the desired removal.
if no right are passed, then the user is unauthorized on all levels.

### Get the rights of a user

```
zstor --config demo.yaml namespace permission get johndoe@email.com namespace_test
```

This command will print the rights that a user with provided [ItsYou.online][iyo] user ID has for the specified namespace:

```
Read: true
Write: true
Delete: true
Admin: true
```

You can also change the format to JSON by specifying the `--`json` flag:

```
zstor --config demo.yaml namespace permission get johndoe@email.com namespace_test --json
{"read":true,"write":true,"delete":true,"admin":true}
```

If you want prettified JSON, you can use the `--json-pretty` flag instead of the `--json` flag.

[iyo]: https://itsyou.online/