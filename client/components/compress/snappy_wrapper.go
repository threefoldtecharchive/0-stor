package compress

import (
	"github.com/golang/snappy"

	"github.com/zero-os/0-stor/client/components/block"
	"github.com/zero-os/0-stor/client/metastor"
)

type snappyWriter struct {
	w block.Writer
}

func newSnappyWriter(w block.Writer) *snappyWriter {
	return &snappyWriter{
		w: w,
	}
}

func (sw snappyWriter) WriteBlock(key, val []byte, md *metastor.Data) (*metastor.Data, error) {
	encoded := snappy.Encode(nil, val)

	// update chunk size in metadata
	chunk := md.GetChunk(key)
	chunk.Size = int64(len(encoded))

	return sw.w.WriteBlock(key, encoded, md)
}

type snappyReader struct {
}

func newSnappyReader() *snappyReader {
	return &snappyReader{}
}

func (sr *snappyReader) ReadBlock(data []byte) ([]byte, error) {
	return snappy.Decode(nil, data)
}
