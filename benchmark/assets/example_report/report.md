# Benchmark report
[Timeplot collection is here](timeplots.md)

 # Report 1 
**Benchmark config:** 
```yaml 
benchmark:
  clients: 0
  duration: 10
  key_size: 96000
  method: write
  operations: 0
  result_output: per_second
  value_size: 4096000
zstor:
  datastor:
    pipeline:
      block_size: 4096
      compression:
        mode: ''
        type: snappy
      distribution:
        data_shards: 2
        parity_shards: 1
      encryption:
        private_key: ''
        type: aes

```
![Fig: throughput vs parameter](fig1.png)
 ### Throughput: 
| value_size | key_size = 48.0 kB |key_size = 96.0 kB | 
|---|---|---| 
 | 1.0 MB |8.9 MB/s |9.3 MB/s | 
| 2.0 MB |9.0 MB/s |9.3 MB/s | 
| 4.1 MB |6.9 MB/s |9.5 MB/s | 
