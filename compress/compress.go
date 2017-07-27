package compress

import (
	"compress/gzip"
	"fmt"
	"io"

	"github.com/zero-os/0-stor-lib/allreader"
	"github.com/zero-os/0-stor-lib/fullreadwrite"
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

// NewWriter returns a new Writer. Writes to the returned writer are compressed and written to w.
func NewWriter(c Config, w fullreadwrite.FullWriter) (fullreadwrite.FullWriter, error) {
	switch c.Type {
	case TypeSnappy:
		return newSnappyWriter(w), nil

	/*case TypeGzip:
	if c.Level == 0 {
		c.Level = DefaultCompression
	}

	return newGzipWriter(w, c.Level), nil
	*/
	//case TypeLz4:
	//	return newLz4Writer(w), nil

	default:
		return nil, fmt.Errorf("unsupported compressor type:%v", c.Type)
	}
}

// NewReader returns a new Reader that decompresses from r
func NewReader(c Config, r io.Reader) (allreader.AllReader, error) {
	switch c.Type {
	case TypeSnappy:
		return newSnappyReader(r), nil

	case TypeGzip:
		return newGzipReader(r)
		/*
			case TypeLz4:
				return newLz4Reader(r), nil
		*/
	default:
		return nil, fmt.Errorf("unsupported decompressor type:%v", c.Type)
	}
}
