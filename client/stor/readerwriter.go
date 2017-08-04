package stor

import (
	"github.com/zero-os/0-stor/client/lib/block"
	"github.com/zero-os/0-stor/client/meta"
)

// Writer is 0-stor client that implements block.Writer interface
type Writer struct {
	shards []string
	client Client
	w      block.Writer
}

// NewWriter creates new writer
func NewWriter(w block.Writer, conf Config, org, namespace string) (*Writer, error) {
	cli, err := NewClient(&conf, org, namespace)
	if err != nil {
		return nil, err
	}
	return &Writer{
		shards: []string{conf.Shard},
		client: cli,
		w:      w,
	}, nil
}

// WriteBlock implements block.Writer.WriteBlock interface
// It creates object in 0-stor server with given key as id and value as data.
// It write the key-metadata to the underlying writer
func (w *Writer) WriteBlock(key, val []byte) (int, error) {
	_, err := w.client.ObjectCreate(key, val, nil)
	if err != nil {
		return 0, err
	}

	// write metadata to the underlying writer
	md, err := meta.New(key, uint64(len(val)), w.shards)
	if err != nil {
		return 0, err
	}

	mdBytes, err := md.Bytes()
	if err != nil {
		return 0, err
	}

	return w.w.WriteBlock(key, mdBytes)
}

// Reader is 0-stor client that implements block.Reader interface
type Reader struct {
	client Client
	w      block.Writer
}

// NewReader creates new Reader
func NewReader(conf Config, org, namespace string) (*Reader, error) {
	cli, err := NewClient(&conf, org, namespace)
	if err != nil {
		return nil, err
	}
	return &Reader{
		client: cli,
	}, nil
}

// ReadBlock implements block.Reader
func (r *Reader) ReadBlock(key []byte) ([]byte, error) {
	obj, err := r.client.ObjectGet(key)
	if err != nil {
		return nil, err
	}
	return obj.Value, nil
}
