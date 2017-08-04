package replication

import (
	"sync"

	"github.com/zero-os/0-stor/client/lib/block"
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
func (w *Writer) WriteBlock(key, data []byte) (int, error) {
	if w.async {
		return w.writeAsync(key, data)
	}
	return w.writeSync(key, data)
}

func (w *Writer) writeAsync(key, data []byte) (int, error) {
	var wg sync.WaitGroup
	var mux sync.Mutex
	var errs []error
	var written int

	wg.Add(len(w.writers))

	for _, writer := range w.writers {
		go func(writer block.Writer) {
			defer wg.Done()

			n, err := writer.WriteBlock(key, data)

			mux.Lock()
			defer mux.Unlock()

			written += n

			if err != nil {
				errs = append(errs, err)
				return
			}
		}(writer)
	}

	wg.Wait()

	if len(errs) > 0 {
		return written, Error{errs: errs}
	}

	return written, nil
}

func (w *Writer) writeSync(key, data []byte) (int, error) {
	var written int
	for _, writer := range w.writers {
		n, err := writer.WriteBlock(key, data)
		written += n
		if err != nil {
			return written, err
		}
	}
	return written, nil
}
