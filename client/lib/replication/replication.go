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
func (w *Writer) WriteBlock(data []byte) block.WriteResponse {
	if w.async {
		return w.writeAsync(data)
	}
	return w.writeSync(data)
}

func (w *Writer) writeAsync(data []byte) (resp block.WriteResponse) {
	var wg sync.WaitGroup
	var mux sync.Mutex

	wg.Add(len(w.writers))

	for _, writer := range w.writers {
		go func(writer block.Writer) {
			defer wg.Done()

			curResp := writer.WriteBlock(data)

			mux.Lock()
			defer mux.Unlock()

			resp.Written += curResp.Written

			if curResp.Err != nil {
				resp.Err = curResp.Err
				return
			}
		}(writer)
	}

	wg.Wait()

	return
}

func (w *Writer) writeSync(data []byte) (resp block.WriteResponse) {
	for _, writer := range w.writers {
		curResp := writer.WriteBlock(data)
		resp.Written += curResp.Written
		if curResp.Err != nil {
			resp.Err = curResp.Err
			return
		}
	}
	return
}
