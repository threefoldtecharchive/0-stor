# Distribution / Erasure Coding

Creation takes the erasure coding policy (data and parity number) and a list of outputs where to distribute the generated n blocks.
Input data is splitted into n blocks, and then written to the ouputs. This component is most probably going to be the last one of the pipeline.

There are two kind of distribution:
- distribution (Distributor & Restorer) : the input/output is io.Reader/io.Writer
- stor distribution (StorDistributor & StorRestorer) : the input/output is 0-stor client

stor distribution is the one going to be used in pipeline.

## Example

```
	conf := Config{
		Data:   4,
		Parity: 2,
	}

	// create list of writers
	var writers []io.Writer
	for i := 0; i < conf.NumPieces(); i++ {
		buf := new(bytes.Buffer)
		writers = append(writers, buf)
	}
	
	data := make([]byte, 4096)

	// distribute
	d, _ := NewDistributor(writers, conf)


	_, err = d.Write(data)

	// restore
	var readers []io.Reader

	for i := 0; i < conf.NumPieces(); i++ {
		var reader io.Reader
		if i < conf.Parity {
			// simulate losing pieces here
			// we can lost up to `m` pieces
			reader = bytes.NewReader(nil)
		} else {
			reader = bytes.NewReader(buffs[i].Bytes())
		}
		readers = append(readers, reader)
	}

	r, _ := NewRestorer(readers, conf)

	decoded := make([]byte, len(data))

	n, _ := r.Read(decoded)

	if bytes.Compare(decoded, data) != 0 {
		log.Fatalf("restore failed")
	}
```

## example of stor distribution

The example can be found in [cli example](../../cmd/cli)
