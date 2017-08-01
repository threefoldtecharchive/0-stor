# Simple 0-stor client cli

Simple cli to store file to 0-stor.
WARNING : it only suitable for small file.
Big file need to be splitted using `chunker` package.

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

`conf_file` is the configuration file. See `simple.yaml` here for the example

### download file

```
./cli conf_file download key result_file_name
```

Get value with key=`key` from 0-stor server and write it to `result_file_name`

`conf_file` is the configuration file. See `simple.yaml` here for the example
