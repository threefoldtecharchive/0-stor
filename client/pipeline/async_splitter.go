package pipeline

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	"golang.org/x/sync/errgroup"

	"github.com/zero-os/0-stor/client/metastor"
	"github.com/zero-os/0-stor/client/pipeline/storage"
)

// NewAsyncSplitterPipeline creates a parallel pipeline,
// which chunks all the content as it reads, and, processes
// and stores them as multiple objects. It is guaranteed that content
// that is written using this pipeline, can be read back
// in the same order as it is written.
//
// NewAsyncSplitterPipeline requires a non-nil ChunkStorage and will panic if it is missing.
// It also requires chunkSize to be positive, if not, it will panic as well.
//
// If no ProcessorConstructor is given, a default constructor will be created for you,
// which will construct a processing.NopProcessor, effectively keeping the data unprocessed at all times.
// While the ProcessorConstructor is optional, it is recommended to define a valid constructor,
// as the storage of unprocessed data is both insecure and inefficient.
//
// If no HasherConstructor is given, a default constructor will be created for you,
// which will construct a 256-bit crypto hasher for you, producing checksums as keys.
// While the HasherConstructor is optional and the default one performs well,
// it is still recommended to define a valid constructor, as it will allow you
// to give a HasherConstructor which creates a crypto Hasher that
// produces signatures as keys, rather than checksums.
//
// If no jobCount is given, meaning it is 0 or less, DefaultJobCount will be used.
func NewAsyncSplitterPipeline(cs storage.ChunkStorage, chunkSize int, pc ProcessorConstructor, hc HasherConstructor, jobCount int) *AsyncSplitterPipeline {
	if cs == nil {
		panic("no ChunkStorage given")
	}
	if chunkSize <= 0 {
		panic("chunk size has to be positive")
	}

	if pc == nil {
		pc = DefaultProcessorConstructor
	}
	if hc == nil {
		hc = DefaultHasherConstructor
	}

	if jobCount <= 0 {
		jobCount = DefaultJobCount
	}
	storageJobCount := jobCount * 2

	return &AsyncSplitterPipeline{
		hasher:            hc,
		processor:         pc,
		storage:           cs,
		storageJobCount:   storageJobCount,
		processorJobCount: jobCount,
		chunkSize:         chunkSize,
	}
}

// AsyncSplitterPipeline defines a parallel pipeline,
// which chunks all the content as it reads, and, processes
// and stores the read data as multiple objects.
// It is guaranteed that content that is written using this pipeline,
// can be read back in the same order as it is written.
type AsyncSplitterPipeline struct {
	hasher                             HasherConstructor
	processor                          ProcessorConstructor
	storage                            storage.ChunkStorage
	storageJobCount, processorJobCount int
	chunkSize                          int
}

// Write implements Pipeline.Write
//
// The following graph visualizes the logic of this pipeline's Write method:
//
//    +-------------------------------------------------------------------+
//    | +----------+                                                      |
//    | | Splitter +---> Processor.Write +   +-----------+                |
//    | |   +      |          ...        +--->  buf-chan +-------+        |
//    | | Hasher   +---> Processor.Write +   +-----------+       |        |
//    | +----------+                                             |        |
//    |                 +-----------------+----------------------+        |
//    |                 v                 v                      v        |
//    |           Storage.Write     Storage.Write    ...   Storage.Write  |
//    |                 +                 +                      +        |
//    |                 |                 |                      |        |
//    |                 |   (ChunkMeta)   |      (ChunkMeta)     |        |
//    |                 |                 |                      |        |
//    |         +-------v-----------------v----------------------v-----+  |
//    |         |              ordered   [] metastor.Chunk             |  |
//    |         +------------------------------------------------------+  |
//    +-------------------------------------------------------------------+
//
// All channels are buffered, as to keep the pipeline as smooth as possible.
//
// The chunks are stored and returned in an ordered slice.
// The order respected and defined by the order in which
// the data that created those chunks was read from the input io.Reader.
//
// As soon as an error happens within any stage, at any point,
// the entire pipeline will be cancelled and that error is returned to the callee of this method.
func (asp *AsyncSplitterPipeline) Write(r io.Reader) ([]metastor.Chunk, error) {
	if r == nil {
		return nil, errors.New("no reader given to read from")
	}

	group, ctx := errgroup.WithContext(context.Background())

	// start the data splitter
	inputCh, splitter := newAsyncDataSplitter(
		ctx, r, asp.chunkSize, asp.processorJobCount)
	group.Go(splitter)

	// start all the processors,
	// which will also create key, using the hasher
	type indexedData struct {
		Index int
		Hash  []byte
		Data  []byte
	}
	dataCh := make(chan indexedData, asp.storageJobCount)
	processorGroup, _ := errgroup.WithContext(ctx)
	for i := 0; i < asp.processorJobCount; i++ {
		hasher, err := asp.hasher()
		if err != nil {
			return nil, err
		}
		processor, err := asp.processor()
		if err != nil {
			return nil, err
		}
		processorGroup.Go(func() error {
			for input := range inputCh {
				// generate the data's hash
				hash := hasher.HashBytes(input.Data)

				// process the data
				data, err := processor.WriteProcess(input.Data)
				if err != nil {
					return err
				}

				// ensure to copy the data,
				// in case the used processor is sharing
				// the buffer between sequential write processes
				if processor.SharedWriteBuffer() {
					b := make([]byte, len(data))
					copy(b, data)
					data = b
				}

				select {
				case dataCh <- indexedData{input.Index, hash, data}:
				case <-ctx.Done():
					return nil
				}
			}
			return nil
		})
	}
	group.Go(func() error {
		err := processorGroup.Wait()
		close(dataCh)
		return err
	})

	// start all storage goroutines,
	// which will generate and send all the metadata as part of their job
	type indexedChunk struct {
		Index int
		Chunk metastor.Chunk
	}
	chunkCh := make(chan indexedChunk, asp.storageJobCount)
	storageGroup, _ := errgroup.WithContext(ctx)
	for i := 0; i < asp.storageJobCount; i++ {
		storageGroup.Go(func() error {
			for data := range dataCh {
				cfg, err := asp.storage.WriteChunk(data.Data)
				if err != nil {
					return err
				}
				chunk := metastor.Chunk{
					Size:    cfg.Size,
					Objects: cfg.Objects,
					Hash:    data.Hash,
				}
				select {
				case chunkCh <- indexedChunk{data.Index, chunk}:
				case <-ctx.Done():
					return nil
				}
			}
			return nil
		})
	}
	group.Go(func() error {
		err := storageGroup.Wait()
		close(chunkCh)
		return err
	})

	// collect all chunks, in the correct order
	var (
		chunks    []metastor.Chunk
		chunkSize int
	)
	group.Go(func() error {
		var (
			receivedChunkCount int
			bufferSize         = asp.storageJobCount
		)
		chunks = make([]metastor.Chunk, asp.storageJobCount)

		// receive all chunks that are send by our storage goroutines
		for chunk := range chunkCh {
			// grow the buffer if needed
			if chunk.Index >= bufferSize {
				bufferSize = chunk.Index + (bufferSize * 2)
				buf := make([]metastor.Chunk, bufferSize)
				copy(buf, chunks)
				chunks = buf
			}

			// store our chunk and increase our received count
			chunks[chunk.Index] = chunk.Chunk
			receivedChunkCount++

			// update our chunkSize if needed,
			// this to know the final chunk length in the end
			if chunk.Index >= chunkSize {
				chunkSize = chunk.Index + 1
			}
		}

		// ensure we have no gaps in our buffered chunk slice
		if receivedChunkCount != chunkSize {
			return errors.New("not all chunks were received")
		}
		return nil
	})

	// wait until all data has been
	// read, chunked, processed and stored
	err := group.Wait()
	if err != nil {
		return nil, err
	}

	// return all received chunks, and nothing more
	return chunks[:chunkSize], nil
}

// Read implements Pipeline.Read
//
// The following graph visualizes the logic of this pipeline's Read method:
//
//    +--------------------------------------------------------------------------------+
//    | +--------------+                                                               |
//    | |[]*ChunkMeta  +----> Storage.Read +-+                                         |
//    | |     to       |                     |                                         |
//    | |chan ChunkMeta+----> Storage.Read +-+     +----------+                        |
//    | +--------------+                     +-----> channels +--------------+         |
//    |            |   ...                   |     +----------+              |         |
//    |            |                         |          |         ...        |         |
//    |            +--------> Storage.Read +-+          |                    |         |
//    |                                         +-------v--------+  +--------v-------+ |
//    |                                         | Processor.Read |  | Processor.Read | |
//    |                                         |       +        |  |       +        | |
//    |                                         |   Hash/Data    |  |   Hash/Data    | |
//    |                                         |   Validation   |  |   Validation   | |
//    |                                         +----------------+  +----------------+ |
//    |                                                 |                    |         |
//    |                                                 |                    |         |
//    |                                             +---v--------------------v---+     |
//    |                                             |                            |     |
//    |                                             |     Data Composer          |     |
//    |                           io.Writer <-------+ (with internal buffer)     |     |
//    |                         (input param)       |                            |     |
//    |                                             +----------------------------+     |
//    +--------------------------------------------------------------------------------+
//
// The data composer (and its internal buffer) is used,
// to ensure we write the raw chunks in the correct order to the io.Writer.
//
// As soon as an error happens within any stage, at any point,
// the entire pipeline will be cancelled and that error is returned to the callee of this method.
//
// If however only one chunk is given, a temporary created SingleObjectPipeline will be used,
// to read the data using the Read method of that pipeline,
// as to now spawn an entire async pipeline, when only one chunk is to be read.
// See (*SingleObjectPipeline).Read for more information about the logic for this scenario.
func (asp *AsyncSplitterPipeline) Read(chunks []metastor.Chunk, w io.Writer) error {
	chunkLength := len(chunks)
	if chunkLength == 0 {
		return errors.New("no chunks given to read")
	}
	if chunkLength == 1 {
		// if only one chunk has to be read,
		// we can fall back on the simpler single-object-pipeline,
		// which executes everything on a single goroutine in a blocking fashion.
		sop := SingleObjectPipeline{
			hasher:    asp.hasher,
			processor: asp.processor,
			storage:   asp.storage,
		}
		return sop.Read(chunks, w)
	}
	if w == nil {
		return errors.New("no writer given to write to")
	}

	// limit our job count in case the chunk size is exceptionally low,
	// as to not spawn goroutines that will never be used
	storageJobCount, processorJobCount := asp.storageJobCount, asp.processorJobCount
	if storageJobCount > chunkLength {
		storageJobCount, processorJobCount = chunkLength, chunkLength
	} else if processorJobCount > chunkLength {
		processorJobCount = chunkLength
	}

	// the master group, which will spawn all non-grouped goroutines,
	// and the close-goroutines for the (processor and storage) sub groups.
	group, ctx := errgroup.WithContext(context.Background())

	// send all chunks one by one,
	// until all chunks have been send, or until the context is cancelled
	type indexedChunk struct {
		Index int
		Chunk metastor.Chunk
	}
	chunkCh := make(chan indexedChunk, storageJobCount)
	go func() {
		defer close(chunkCh)
		for index, chunk := range chunks {
			select {
			case chunkCh <- indexedChunk{index, chunk}:
			case <-ctx.Done():
				return
			}
		}
	}()

	// start all storage readers
	// which will read all data until all data has been read,
	// or until the context is cancelled
	type indexedInput struct {
		Index int
		Data  []byte
		Hash  []byte
	}
	storageGroup, _ := errgroup.WithContext(ctx)
	inputCh := make(chan indexedInput, processorJobCount)
	for i := 0; i < storageJobCount; i++ {
		storageGroup.Go(func() error {
			for ic := range chunkCh {
				data, err := asp.storage.ReadChunk(storage.ChunkConfig{
					Size:    ic.Chunk.Size,
					Objects: ic.Chunk.Objects,
				})
				if err != nil {
					return err
				}
				// send the object data and key, for further processing and validation
				select {
				case inputCh <- indexedInput{ic.Index, data, ic.Chunk.Hash}:
				case <-ctx.Done():
					return nil
				}
			}
			return nil
		})
	}
	group.Go(func() error {
		err := storageGroup.Wait()
		close(inputCh)
		return err
	})

	// start all processors
	// which will process all incoming objects,
	// until:
	//  - all objects have been processed;
	//  - or an object has an invalid hash;
	//  - the context has been cancelled due to an error somewhere else;
	type indexedData struct {
		Index     int
		DataChunk []byte
	}
	processorGroup, _ := errgroup.WithContext(ctx)
	outputCh := make(chan indexedData, processorJobCount)
	for i := 0; i < processorJobCount; i++ {
		hasher, err := asp.hasher()
		if err != nil {
			return err
		}
		processor, err := asp.processor()
		if err != nil {
			return err
		}
		processorGroup.Go(func() error {
			for input := range inputCh {
				data, err := processor.ReadProcess(input.Data)
				if err != nil {
					return err
				}
				if bytes.Compare(input.Hash, hasher.HashBytes(data)) != 0 {
					return fmt.Errorf("object chunk #%d's data and hash do not match", input.Index)
				}

				// ensure to copy the data,
				// in case the used processor is sharing
				// the buffer between sequential read processes
				if processor.SharedReadBuffer() {
					b := make([]byte, len(data))
					copy(b, data)
					data = b
				}

				result := indexedData{
					Index:     input.Index,
					DataChunk: data,
				}
				select {
				case outputCh <- result:
				case <-ctx.Done():
					return nil
				}
			}
			return nil
		})
	}
	group.Go(func() error {
		err := processorGroup.Wait()
		close(outputCh)
		return err
	})

	// start the output goroutine,
	// this one will write all output data,
	// to the given writer, respecting the original order
	// as defined by the input chunks.
	group.Go(func() error {
		var (
			err           error
			ok            bool
			data          []byte
			expectedIndex int
			buffer        = make(map[int][]byte, processorJobCount)
		)
		for output := range outputCh {
			if output.Index != expectedIndex {
				// if this data is not the one we're expecting,
				// buffer it for now
				buffer[output.Index] = output.DataChunk
				continue
			}

			// write the current data, as we expect it
			// also write all buffered data that can be written,
			// should it exist
			data, ok = output.DataChunk, true
			for ok {
				_, err = w.Write(data)
				if err != nil {
					return err
				}
				expectedIndex++
				data, ok = buffer[expectedIndex]
			}
		}

		// we're done, success!
		return nil
	})

	// wait for all goroutines to be finished
	return group.Wait()
}

// GetChunkStorage implements Pipeline.GetChunkStorage
func (asp *AsyncSplitterPipeline) GetChunkStorage() storage.ChunkStorage {
	return asp.storage
}

type indexedDataChunk struct {
	Index int
	Data  []byte
}

// newAsyncDataSplitter creates a functional data splitter,
// which can be used to split streaming input data into fixed-sized chunks,
// in an asynchronous fashion.
func newAsyncDataSplitter(ctx context.Context, r io.Reader, chunkSize, bufferSize int) (<-chan indexedDataChunk, func() error) {
	inputCh := make(chan indexedDataChunk, bufferSize)
	return inputCh, func() error {
		defer close(inputCh)
		var index int
		buf := make([]byte, chunkSize)
		for {
			n, err := r.Read(buf)
			if err != nil {
				if err == io.EOF {
					// we'll consider an EOF
					// as a signal to let us know the reader is exhausted
					return nil
				}
				return err
			}
			data := make([]byte, n)
			copy(data, buf)
			select {
			case inputCh <- indexedDataChunk{index, data}:
				index++
			case <-ctx.Done():
				return nil
			}
		}
	}
}
