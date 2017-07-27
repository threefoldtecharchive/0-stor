package compress

import (
	"io"

	"github.com/pierrec/lz4"
)

type lz4Writer struct {
	w io.Writer
}

func newLz4Writer(w io.Writer) *lz4Writer {
	return &lz4Writer{
		w: w,
	}
}

func (lw lz4Writer) Write(p []byte) (int, error) {
	var dst []byte

	n, err := lz4.CompressBlock(p, dst, 0)
	if err != nil {
		return n, err
	}
	return lw.w.Write(dst)
}

// lz4Reader wraps lz4.Reader to conform to Decompressor interface
type lz4Reader struct {
	*lz4.Reader
}

func newLz4Reader(r io.Reader) *lz4Reader {
	return &lz4Reader{
		Reader: lz4.NewReader(r),
	}
}

func (lr *lz4Reader) Close() error {
	return nil
}

func (lr *lz4Reader) Reset(r io.Reader) error {
	lr.Reader.Reset(r)
	return nil
}
