package meta

import (
	"github.com/zero-os/0-stor/client/lib/block"
)

type Config struct {
	Shards []string `yaml:"shards"`
}

type Writer struct {
	client *Client
	w      block.Writer
}

func NewWriter(w block.Writer, conf Config) (*Writer, error) {
	cli, err := NewClient(conf.Shards)
	if err != nil {
		return nil, err
	}
	return &Writer{
		client: cli,
		w:      w,
	}, nil
}

// WriteBlock implements block.Writer interface.
// The value is encoded metadata
func (w *Writer) WriteBlock(key, val []byte) (int, error) {
	md, err := Decode(val)
	if err != nil {
		return 0, err
	}
	if err := w.client.Put(string(key), md); err != nil {
		return 0, err
	}
	return w.w.WriteBlock(key, val)
}

type Reader struct {
	client *Client
}

func NewReader(conf Config) (*Reader, error) {
	cli, err := NewClient(conf.Shards)
	if err != nil {
		return nil, err
	}
	return &Reader{
		client: cli,
	}, nil
}

func (r *Reader) ReadBlock(key []byte) ([]byte, error) {
	md, err := r.client.Get(string(key))
	if err != nil {
		return nil, err
	}
	return md.Bytes()
}
