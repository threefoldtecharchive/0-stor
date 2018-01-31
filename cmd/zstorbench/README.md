# zstorbench

Zstorbench is a benchmark client/tool to test the performance of `zstor` (`0-stor`).

## Getting started
Download the `zstor`repository.
```bash
git clone https://github.com/zero-os/0-stor.git

# or 
go get -u -d github.com/zero-os/0-stor/cmd/zstorbench
```

Install the `zstor` components by running following command in the zstor root of the repository:
```bash
cd $GOPATH/src/github.com/zero-os/0-stor
make install
```

Before starting a benchmark, make sure the necessary services for zstor are running.  
A guide to set up zstordb's can her found [here](https://github.com/zero-os/0-stor/blob/master/docs/gettingstarted.md).
To set up etcd metadata server, check the etcd [documentation](https://coreos.com/etcd/docs/latest/).
Zstor requires	etcd 3.2.4 or any higher stable release.

Start the benchmarking by providing zstorbench with a [config](#Benchmark-config) file.
``` bash
zstorbench -C config.yaml --out-benchmark benchmark.yaml
```

Start benchmarking with profiling
``` bash
zstorbench -C config.yaml --profile-mode cpu --out-profile "outputProfileInfo"
```

`zstorbench` has the following options:
```
  -C, --conf string                path to a config file (default "zstorbench_config.yaml")
  -D, --debug                      Enable debug logging.
  -h, --help                       help for zstorbench
      --out-benchmark string       path and filename where benchmarking results are written (default "benchmark.yaml")
      --out-profile string         path where profiling files are written (default "./profile")
      --profile-mode profileMode   Enable profiling mode, one of [cpu, mem, block, trace]
```

## Benchmark config

Client config contains a list of scenarios. 
Each scenario is associated with a corresponding scenarioID and contains 2 main fields: 
* `zstor` representing the config for the zstor client
* `benchmark` representing the configuration of the benchmarking

The `zstor` represents a zstor config. More details can be found [here](../zstor/README.md#Configuration).

The `iyo` field of the `zstor` contains the credentials for [`itsyou.online`](https://itsyou.online), used for authentication during the benchmark. If the sub fields are empty or the `iyo` field itself is omitted, the benchmarks will run without authentication.  
If authentification is enabled, the `namespace` fields needs to be a valid [`itsyou.online`](https://itsyou.online) namespace, if authentification is disabled, any name can be used or it can be omitted for a random one to be generated.

If the `db` `endpoints` of the `metastor` field in the `zstor` is empty or omitted, `zstorbench` will use in-memory metadata storage. This in-memory metadata storage is not meant for production, but allows for benchmarking without a remote metadata server being a potential bottleneck.

The `benchmark` represents the benchmarking parameters.

The `method` field of the `benchmark` defines which operation is benchmarked.  
Current supported operations are:
* `read` - for reading from zstor
* `write` - for writing to zstor

The `result_output` field of the `benchmark` specifies interval of the data collection (`perinterval` in the results) and can take values:  
* per_second
* per_minute
* per_hour
If empty or invalid, there will be no interval data collection.

The `duration` and `operations` sub fields describe the conditions that end the benchmark. One of them should be given for a valid configuration.  
If only `duration` is given, the benchmark will end when the time elapsed defined by this field (in seconds) has been reached.  
If only `operations` is given, the benchmark will end when the amount of operations defined by this field has been reached.  
If both of them are given, the benchmark will end when the first condition has been reached.

The `key_size` and `value_size` sub fields define the length of the key and value respectively which will be used during the benchmarks.

The `clients` sub field defines the amount of concurrent benchmarking clients that execute the benchmark on the zstor setup defined in the `zstor`.

Example of a config YAML file for zstorbench:

``` yaml
scenarios:
  bench1:
    zstor:
      iyo:  # If empty or omitted, the zstordb servers set up for the benchmark 
            # needs to be run with the no-auth flag.
            # For benching with authentication, provide it with valid itsyou.online credentials
        organization: "bench_org"
        app_id: "an_iyo_bench_app_id"
        app_secret: "an_iyo_bench_app_secret"
      namespace: adisk # If IYO credentials are provided,
                       # this needs to be a valid and existing IYO namespace,
                       # otherwise this can be any name or omitted
                       # and the namespace will be generated
      datastor:
        shards:
        - 127.0.0.1:45627
        - 127.0.0.1:49861
        - 127.0.0.1:37355
        pipeline:
          block_size: 4096
          compression:
            type: snappy # snappy is the default, other options: lz4, gzip
            mode: default # default is the default compression mode, for gzip other options: best_speed, best_compression
          encryption:
            private_key: ab345678901234567890123456789012
          distribution:
            data_shards: 2
            parity_shards: 1
      metastor:
        db:
          endpoints:  # If empty or omitted, an in memory metadata server will be used
                      # Otherwise it will presume to have etcd servers running on these addresses
            - 127.0.0.1:1300
            - 127.0.0.1:1301
        encryption:
          private_key: ab345678901234567890123456789012
    benchmark:
      method: write
      result_output: per_second # if empty or invalid, perinterval in the result will be empty
      duration: 5
      operations: 0
      key_size: 48
      value_size: 128
      clients: 1  # number of concurrent clients
```

## Output file

`Zstorbench` writes the results to a YAML file. It will contain a list of the supplied scenarios by the config file and adds a `result` field to each scenario. If a scenario has failed, there will be an `error` field instead of the `result` field.

``` yaml
scenarios:
  bench1:
    results:
    - count: 561            # total operations during this scenario
      duration: 5.0085373   # how long the scenario took in decimal seconds
      perinterval:          # operations per time interval defined by `result_output`
      - 110                 # ordered from first interval to last
      - 113
      - 114
      - 113
      - 111
    scenario:
      zstor:
        iyo:
          organization: ""
          app_id: ""
          app_secret: ""
        namespace: adisk
        datastor:
          shards:
          - 127.0.0.1:1200
          - 127.0.0.1:1201
          - 127.0.0.1:1202
          pipeline:
            block_size: 4096
            hashing:
              type: blake2b_256
              private_key: ""
            compression:
              mode: default
              type: snappy
            encryption:
              private_key: ab345678901234567890123456789012
              type: aes
            distribution:
              data_shards: 2
              parity_shards: 1
        metastor:
          db:
            endpoints:
            - 127.0.0.1:1300
            - 127.0.0.1:1301
          encryption:
            private_key: ab345678901234567890123456789012
            type: aes
          encoding: protobuf
      benchmark:
        method: write
        result_output: per_second
        duration: 5
        operations: 0
        clients: 1
        key_size: 48
        value_size: 128
```