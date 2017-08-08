package replication

import (
	"sync"

	"github.com/zero-os/0-stor/client/lib/block"
	"github.com/zero-os/0-stor/client/meta"
)

// A Writer is created by taking one input and specifying multiple outputs.
// All the data that comes in are replicated on all the configured outputs.
type Writer struct {
	async   bool
	writers []block.Writer
}

// Config defines replication's configuration
type Config struct {
	Async bool `yaml:"async"`
}

// NewWriter creates new writer.
// The replication will be done in async way if async = true.
func NewWriter(writers []block.Writer, conf Config) *Writer {
	return &Writer{
		async:   conf.Async,
		writers: writers,
	}
}

// Write writes data to underlying writer
func (w *Writer) WriteBlock(key, data []byte, md *meta.Meta) (*meta.Meta, error) {
	if w.async {
		return w.writeAsync(key, data, md)
	}
	return w.writeSync(key, data, md)
}

func (w *Writer) writeAsync(key, data []byte, md *meta.Meta) (*meta.Meta, error) {
	var wg sync.WaitGroup
	var mux sync.Mutex
	var errs []error
	var written int

	wg.Add(len(w.writers))

	for _, writer := range w.writers {
		go func(writer block.Writer) {
			defer wg.Done()

			md, err := writer.WriteBlock(key, data, md)

			mux.Lock()
			defer mux.Unlock()

			written += int(md.Size())

			if err != nil {
				errs = append(errs, err)
				return
			}
		}(writer)
	}

	wg.Wait()

	md.SetSize(uint64(written))
	if len(errs) > 0 {
		return md, Error{errs: errs}
	}

	return md, nil
}

func (w *Writer) writeSync(key, data []byte, md *meta.Meta) (*meta.Meta, error) {
	var written int
	for _, writer := range w.writers {
		md, err := writer.WriteBlock(key, data, md)
		written += int(md.Size())
		if err != nil {
			md.SetSize(uint64(written))
			return md, err
		}
	}
	md.SetSize(uint64(written))
	return md, nil
}
