## config

```go
type Settings struct {
	DebugLog    bool   `json:"debug"`
	BindAddress string `json:"bind"`

	DB struct {
		Dirs struct {
			Meta string `json:"meta"`
			Data string `json:"data"`
		} `json:"dirs"`
	} `json:"db"`

	ServerType string `json:"server_type"`
}
```


**Change config through command line options**

- DebugLog
    - Enables debugging.
    - Default is False
    - To enable degugging, use ```./server  --debug``` or ```./server -d```

- BindAddress
    - By default 0stor listens on all interfaces on port 8080
    - To change bind address ```./server --bind :9090``` or ```./server -b :9090```

- DB
    - Meta
        - Database Metadata files location
        - Change Metadata dir using ```./server --meta {PATH}```
    - Data
        - Database Data files location
        - ```./server --data {PATH}```


- ServerType
    - Change server listening interface REST/GRPC
    - ```./server --interface rest``` or ```./server --interface grpc```

**Load config from JSON file**

- ```./server --config {PATH}```
- Config file example
```config.json
    {
        "debug" : true,
        "bind": ":8000",
        "server_type" : "rest",
        "db": {
            "meta": ".db/meta",
            "data": ".db/data"
        },

    }
 ```