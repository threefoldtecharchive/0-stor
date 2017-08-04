package compress

import (
	"github.com/golang/snappy"

	"github.com/zero-os/0-stor/client/lib/block"
)

type snappyWriter struct {
	w block.Writer
}

func newSnappyWriter(w block.Writer) *snappyWriter {
	return &snappyWriter{
		w: w,
	}
}

func (sw snappyWriter) WriteBlock(key, val []byte) (int, error) {
	encoded := snappy.Encode(nil, val)
	return sw.w.WriteBlock(key, encoded)
}

type snappyReader struct {
}

func newSnappyReader() *snappyReader {
	return &snappyReader{}
}

func (sr *snappyReader) ReadBlock(data []byte) ([]byte, error) {
	return snappy.Decode(nil, data)
}
