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

//Compressor is the interface that wraps the basic compress method
type Compressor interface {
	Close() error
	Flush() error
	Reset(w io.Writer)
	Write(p []byte) (int, error)
}

// Writer is compress writer, implements the Compressor interface
// Writes to a Writer are compressed and written to w.
type Writer struct {
	Compressor
	typ int
}

// NewWriter returns a new Writer. Writes to the returned writer are compressed and written to w.
// It is the caller's responsibility to call Close on the WriteCloser when done. Writes may be buffered and not flushed until Close.
func NewWriter(c Config, w io.Writer) (*Writer, error) {
	var comp Compressor
	var err error

	switch c.Type {
	case TypeSnappy:
		comp = snappy.NewBufferedWriter(w)

	case TypeGzip:
		if c.Level == 0 {
			c.Level = DefaultCompression
		}
		comp, err = gzip.NewWriterLevel(w, c.Level)
		if err != nil {
			return nil, err
		}

	case TypeLz4:
		comp = lz4.NewWriter(w)

	default:
		return nil, fmt.Errorf("unsupported compressor type:%v", c.Type)
	}

	return &Writer{
		Compressor: comp,
		typ:        c.Type,
	}, nil
}

// Decompressor is the interface that wraps the basic decompress method
type Decompressor interface {
	Close() error
	Read(p []byte) (int, error)
	Reset(r io.Reader) error
}

// A Reader is an io.Reader that can be read to retrieve uncompressed data
// from the compressed one.
// It implements the Decompressor interface
type Reader struct {
	Decompressor
	typ int
}

// NewReader returns a new Reader that decompresses from r
func NewReader(c Config, r io.Reader) (*Reader, error) {
	var d Decompressor
	var err error

	switch c.Type {
	case TypeSnappy:
		d = newSnappyReader(r)

	case TypeGzip:
		d, err = gzip.NewReader(r)
		if err != nil {
			return nil, err
		}

	case TypeLz4:
		d = newLz4Reader(r)

	default:
		return nil, fmt.Errorf("unsupported decompressor type:%v", c.Type)
	}

	return &Reader{
		Decompressor: d,
		typ:          c.Type,
	}, nil

}
