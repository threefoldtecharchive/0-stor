
# redis client interface to zero stor

- start zero-stor and connect to a redis & path (for the incoming queue (unixsocket preferred))
- for the incoming queue support set/get 
    - see data formats below
- the result is posted back on specified return queue

why 

- no need for grpc client (issue in e.g. crystal twin)
- queuing, can easily make multi user

## remarks

- no encryption (needs already to be done in advance if required)
- always compression


## data formats

### data format for set

is json format, but shown here for readability in toml

```toml
#make sure works for ipv6, example in ipv4
dest = [
    22.33.44.55:333:mynamespace:mysecret,
    22.33.44.57:333:mynamespace:mysecret3,
    22.33.44.58:333:mynamespace:mysecret1,
    22.33.44.59:333:mynamespace:mysecret2,
    22.33.44.51:333:mynamespace:mysecret3
    ]
#first nr is the nr of min nodes required, sum is always amount of nodes defined in dest
policy = "3/2"
path = "/storage/something/file
returnqueue = "zstor:results:myjob1"
```

will return required data for a stor

return

- json for file_meta

```toml
#below is the filemeta + return queue
#make sure works for ipv6, example in ipv4
source = [
    22.33.44.55:333:mynamespace:mysecret,
    22.33.44.57:333:mynamespace:mysecret3,
    22.33.44.58:333:mynamespace:mysecret1,
    22.33.44.59:333:mynamespace:mysecret2,
    22.33.44.51:333:mynamespace:mysecret3
    ]

blocks = [
    [1212,3434,3434], #id's corresponding to source 1
    [34,34,52],
    [23,25,22],
    [2342,52,22],
    [232,55,22]
]

returnqueue = "zstor:results:myjob1"

```

- data is the id's as used in the zdb keys
- in this example 3 blocks of data make up the file



### data format for get

```toml
#make sure works for ipv6, example in ipv4
source = [
    22.33.44.55:333:mynamespace:mysecret,
    22.33.44.57:333:mynamespace:mysecret3,
    22.33.44.58:333:mynamespace:mysecret1,
    22.33.44.59:333:mynamespace:mysecret2,
    22.33.44.51:333:mynamespace:mysecret3
    ]
blocks = [
    [1212,3434,3434], #id's corresponding to source 1
    [34,34,52],
    [23,25,22],
    [2342,52,22],
    [232,55,22]
    ]

path = "/storage/something/file
returnqueue = "zstor:results:myjob1"
```

return 

- ```OK``` when done
- ```ERROR: errormsg```

