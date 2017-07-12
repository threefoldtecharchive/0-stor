package compress

import (
	"io"

	"github.com/golang/snappy"
)

// snappyReader wraps the snappy.Reader object
// to conform to Decompressor interface
type snappyReader struct {
	*snappy.Reader
}

func newSnappyReader(r io.Reader) *snappyReader {
	return &snappyReader{
		Reader: snappy.NewReader(r),
	}
}

func (sr *snappyReader) Read(p []byte) (int, error) {
	return sr.Reader.Read(p)
}

func (sr *snappyReader) Reset(rd io.Reader) error {
	sr.Reader.Reset(rd)
	return nil
}

func (sr *snappyReader) Close() error {
	return nil
}
