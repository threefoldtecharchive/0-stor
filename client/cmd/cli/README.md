# Simple 0-stor client cli

Simple cli to store file to 0-stor.

## Example

To use this `cli`, you need to modify the itsyou.online credentials:
- organization
- namespace
- client ID
- client secret

### upload file

```
./cli conf_file upload file_name
```

upload file and use the `file_name` as 0-stor key

`conf_file` is the configuration file. See `config.yaml` here for the example

### download file

```
./cli conf_file download key result_file_name
```

Get value with key=`key` from 0-stor server and write it to `result_file_name`

`conf_file` is the configuration file. See `config.yaml` here for the example

### distribution / erasure encoding test

stop one of the 0-stor server and retry your download.
the download will still succeed.

