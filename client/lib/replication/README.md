# Replication

A replicater is created by taking one input and specifying multiple outputs. 
All the data that comes in are replicated on all the configured outputs.

## Example

```
	var writers []block.Writer
	numWriter := 10
	data := make([]byte, 4096)

	// create block writers which do the replication
	for i := 0; i < numWriter; i++ {
		writers = append(writers, block.NewBytesBuffer())
	}

	// create writer
	w := NewWriter(writers, Config{Async: async})

	// replicate
	resp := w.WriteBlock(data)
	assert.Nil(t, resp.Err)

	// check replicated data
	for i := 0; i < numWriter; i++ {
		buff := writers[i].(*block.BytesBuffer)
		if bytes.Compare(data, buff.Bytes()) != 0 {
			log.Fatal("invalid replicated data")
		}
	}
```
```
