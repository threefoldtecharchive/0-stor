package compress

import (
	"compress/gzip"
	"fmt"
	"io"

	"github.com/golang/snappy"
	"github.com/pierrec/lz4"
)

// Compressor/decompressor type
const (
	_ = iota
	TypeSnappy
	TypeGzip
	TypeLz4
)

// Compression level, only apply for gzip
const (
	BestSpeed          = gzip.BestSpeed
	BestCompression    = gzip.BestCompression
	DefaultCompression = gzip.DefaultCompression
	HuffmanOnly        = gzip.HuffmanOnly
)

// Config define compressor and decompressor configuration
type Config struct {
	// Compressor type : TypeSnappy, TypeGzip
	Type int `yaml:"type"`

	// Compression level : only supported for gzip
	// Leave it blank for default value
	Level int `yaml:"level"`
}

// Writer is the interface that wraps the basic compress method
type Writer interface {
	Close() error
	Flush() error
	Reset(w io.Writer)
	Write(p []byte) (int, error)
}

// NewWriter returns a new Writer. Writes to the returned writer are compressed and written to w.
// It is the caller's responsibility to call Close on the WriteCloser when done. Writes may be buffered and not flushed until Close.
func NewWriter(c Config, w io.Writer) (Writer, error) {
	switch c.Type {
	case TypeSnappy:
		return snappy.NewBufferedWriter(w), nil

	case TypeGzip:
		if c.Level == 0 {
			c.Level = DefaultCompression
		}
		return gzip.NewWriterLevel(w, c.Level)
	case TypeLz4:
		return lz4.NewWriter(w), nil

	default:
		return nil, fmt.Errorf("unsupported compressor type:%v", c.Type)
	}
}

// A Reader is an io.Reader that can be read to retrieve uncompressed data
type Reader interface {
	Close() error
	Read(p []byte) (int, error)
	Reset(r io.Reader) error
}

// NewReader returns a new Reader that decompresses from r
func NewReader(c Config, r io.Reader) (Reader, error) {
	switch c.Type {
	case TypeSnappy:
		return newSnappyReader(r), nil

	case TypeGzip:
		return gzip.NewReader(r)

	case TypeLz4:
		return newLz4Reader(r), nil

	default:
		return nil, fmt.Errorf("unsupported decompressor type:%v", c.Type)
	}
}
