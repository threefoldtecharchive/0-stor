package compress_test

import (
	"bytes"
	"fmt"
	"time"

	"github.com/zero-os/0-stor/client/components/compress"
	"github.com/zero-os/0-stor/client/metastor"

	"github.com/zero-os/0-stor/client/components/block"
)

func Example() {
	// given payload ...
	payload := []byte("aaaabbaaabbbbbaabbababa")
	fmt.Printf("Payload:\n%v\n", string(payload))

	// given compress config
	conf := compress.Config{
		Type: compress.TypeSnappy,
	}

	// we define metadata for 0-stor
	md := &metastor.Data{Key: []byte("foo"), Epoch: time.Now().UnixNano()}

	// we can compress the payload and
	// write it to block.BytesBuffer buf
	buf := block.NewBytesBuffer()
	w, _ := compress.NewWriter(conf, buf)
	_, err := w.WriteBlock(nil, payload, md)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Compressed Payload:\n%v\n", string(buf.Bytes()))

	// then we can read and decompress data
	r, _ := compress.NewReader(conf)
	decompressed, _ := r.ReadBlock(buf.Bytes())

	if bytes.Compare(payload, decompressed) != 0 {
		panic("x != decompress(compress(x))")
	}
	// Output:
	// Payload:
	// aaaabbaaabbbbbaabbababa
	// Compressed Payload:
	// aaaabb,bbbaabbababa
}
