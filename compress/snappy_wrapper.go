package compress

import (
	"io"

	"github.com/golang/snappy"

	"github.com/zero-os/0-stor-lib/fullreadwrite"
)

type snappyWriter struct {
	w fullreadwrite.Writer
}

func newSnappyWriter(w fullreadwrite.Writer) *snappyWriter {
	return &snappyWriter{
		w: w,
	}
}

func (sw snappyWriter) Write(p []byte) (int, error) {
	encoded := snappy.Encode(nil, p)
	return sw.w.Write(encoded)
}

func (sw snappyWriter) WriteFull(p []byte) fullreadwrite.WriteResponse {
	encoded := snappy.Encode(nil, p)
	return sw.w.WriteFull(encoded)
}

// snappyReader wraps the snappy.Reader object
// to conform to Decompressor interface
type snappyReader struct {
	*snappy.Reader
	rd io.Reader
}

func newSnappyReader(r io.Reader) *snappyReader {
	return &snappyReader{
		Reader: snappy.NewReader(r),
		rd:     r,
	}
}

func (sr *snappyReader) ReadFull(data []byte) ([]byte, error) {
	return snappy.Decode(nil, data)
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
