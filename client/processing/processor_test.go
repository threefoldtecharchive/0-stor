/*
 * Copyright (C) 2017-2018 GIG Technology NV and Contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package processing

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	mathRand "math/rand"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

func TestNopProcessor_ReadWrite(t *testing.T) {
	testProcessorReadWrite(t, NopProcessor{})
}

func TestNopProcessor_ReadWrite_MultiLayer(t *testing.T) {
	testProcessorReadWriteMultiLayer(t, NopProcessor{})
}

func TestNopProcessor_ReadWrite_Async(t *testing.T) {
	testProcessorReadWriteAsync(t, func() Processor {
		return NopProcessor{}
	})
}

// fooPrefixProcessor is a simple processor,
// which simply prefixes input data with 'foo_' when writing,
// and it removes that prefix again when reading
// (after validating that the read input data has that prefix indeed).
//
// The purpose of this processor is to validate our
// processor tests' logic, especially for the more complex
// tests this is useful, as to not waste valuable debugging time,
// while in fact our processor test itself is buggy.
type fooPrefixProcessor struct{}

func (fpp fooPrefixProcessor) WriteProcess(data []byte) ([]byte, error) {
	return append([]byte("foo_"), data...), nil
}

func (fpp fooPrefixProcessor) ReadProcess(data []byte) ([]byte, error) {
	if !bytes.HasPrefix(data, []byte("foo_")) {
		return nil, fmt.Errorf("'%s' does not have prefix 'foo_'", data)
	}
	return bytes.TrimPrefix(data, []byte("foo_")), nil
}

func (fpp fooPrefixProcessor) SharedWriteBuffer() bool { return true }

func (fpp fooPrefixProcessor) SharedReadBuffer() bool { return true }

func TestFooPrefixProcessor_ReadWrite(t *testing.T) {
	testProcessorReadWrite(t, fooPrefixProcessor{})
}

func TestFooPrefixProcessor_ReadWrite_MultiLayer(t *testing.T) {
	testProcessorReadWriteMultiLayer(t, fooPrefixProcessor{})
}

func TestFooPrefixProcessor_ReadWrite_Async(t *testing.T) {
	testProcessorReadWriteAsync(t, func() Processor {
		return fooPrefixProcessor{}
	})
}

func TestNewProcessorChain(t *testing.T) {
	require := require.New(t)

	require.Panics(func() {
		NewProcessorChain(nil)
	}, "no processors given")
	require.Panics(func() {
		NewProcessorChain([]Processor{})
	}, "no processors given")
	require.Panics(func() {
		NewProcessorChain([]Processor{NopProcessor{}})
	}, "< 2 processors given")

	chain := NewProcessorChain([]Processor{NopProcessor{}, NopProcessor{}})
	require.NotNil(chain)

	require.False(chain.SharedReadBuffer(), "nop processor has no shared read buffer")
	require.False(chain.SharedWriteBuffer(), "nop processor has no shared write buffer")

	data, err := chain.WriteProcess([]byte("data"))
	require.NoError(err)
	require.Equal([]byte("data"), data)
	data, err = chain.ReadProcess([]byte("data"))
	require.NoError(err)
	require.Equal([]byte("data"), data)

	data, err = chain.WriteProcess(nil)
	require.NoError(err)
	require.Empty(data)
	data, err = chain.ReadProcess(nil)
	require.NoError(err)
	require.Empty(data)
}

type pcc func(t *testing.T) Processor

var processorChainTestcases = []struct {
	Name        string
	Constructor pcc
}{
	// some initial test cases,
	// using just dev/test processors,
	// should already tell us if the chain works
	{"nop<->nop", func(t *testing.T) Processor {
		return NewProcessorChain([]Processor{
			NopProcessor{},
			NopProcessor{},
		})
	}},
	{"nop<->fooPrefix", func(t *testing.T) Processor {
		return NewProcessorChain([]Processor{
			NopProcessor{},
			fooPrefixProcessor{},
		})
	}},
	{"fooPrefix<->nop", func(t *testing.T) Processor {
		return NewProcessorChain([]Processor{
			fooPrefixProcessor{},
			NopProcessor{},
		})
	}},
	{"fooPrefix<->fooPrefix", func(t *testing.T) Processor {
		return NewProcessorChain([]Processor{
			fooPrefixProcessor{},
			fooPrefixProcessor{},
		})
	}},
	{"nop<->nop<->nop", func(t *testing.T) Processor {
		return NewProcessorChain([]Processor{
			NopProcessor{},
			NopProcessor{},
			NopProcessor{},
		})
	}},
	{"fooPrefix<->fooPrefix<->fooPrefix", func(t *testing.T) Processor {
		return NewProcessorChain([]Processor{
			fooPrefixProcessor{},
			fooPrefixProcessor{},
			fooPrefixProcessor{},
		})
	}},
	{"fooPrefix<->nop<->fooPrefix", func(t *testing.T) Processor {
		return NewProcessorChain([]Processor{
			fooPrefixProcessor{},
			NopProcessor{},
			fooPrefixProcessor{},
		})
	}},
	{"nop<->fooPrefix<->nop", func(t *testing.T) Processor {
		return NewProcessorChain([]Processor{
			NopProcessor{},
			fooPrefixProcessor{},
			NopProcessor{},
		})
	}},

	// some more interesting processor chains
	{"fooPrefix<->compression:snappy", func(t *testing.T) Processor {
		cd, err := NewSnappyCompressorDecompressor(CompressionModeDefault)
		require.NoError(t, err)
		require.NotNil(t, cd)

		return NewProcessorChain([]Processor{
			fooPrefixProcessor{},
			cd,
		})
	}},
	{"compression:snappy<->fooPrefix", func(t *testing.T) Processor {
		cd, err := NewSnappyCompressorDecompressor(CompressionModeDefault)
		require.NoError(t, err)
		require.NotNil(t, cd)

		return NewProcessorChain([]Processor{
			cd,
			fooPrefixProcessor{},
		})
	}},
	{"fooPrefix<->compression:snappy<->fooPrefix", func(t *testing.T) Processor {
		cd, err := NewSnappyCompressorDecompressor(CompressionModeDefault)
		require.NoError(t, err)
		require.NotNil(t, cd)

		return NewProcessorChain([]Processor{
			fooPrefixProcessor{},
			cd,
			fooPrefixProcessor{},
		})
	}},
	{"compression:snappy<->fooPrefix<->compression:snappy", func(t *testing.T) Processor {
		cd1, err := NewSnappyCompressorDecompressor(CompressionModeDefault)
		require.NoError(t, err)
		require.NotNil(t, cd1)
		cd2, err := NewSnappyCompressorDecompressor(CompressionModeDefault)
		require.NoError(t, err)
		require.NotNil(t, cd2)

		return NewProcessorChain([]Processor{
			cd1,
			fooPrefixProcessor{},
			cd2,
		})
	}},

	// let's do some weird combinations, with multiple compressors
	{"compression:snappy<->compression:snappy", func(t *testing.T) Processor {
		cd1, err := NewSnappyCompressorDecompressor(CompressionModeDefault)
		require.NoError(t, err)
		require.NotNil(t, cd1)
		cd2, err := NewSnappyCompressorDecompressor(CompressionModeDefault)
		require.NoError(t, err)
		require.NotNil(t, cd2)

		return NewProcessorChain([]Processor{
			cd1,
			cd2,
		})
	}},
	{"compression:snappy<->compression:LZ4", func(t *testing.T) Processor {
		cd1, err := NewSnappyCompressorDecompressor(CompressionModeDefault)
		require.NoError(t, err)
		require.NotNil(t, cd1)
		cd2, err := NewLZ4CompressorDecompressor(CompressionModeDefault)
		require.NoError(t, err)
		require.NotNil(t, cd2)

		return NewProcessorChain([]Processor{
			cd1,
			cd2,
		})
	}},
	{"compression:LZ4<->compression:Snappy", func(t *testing.T) Processor {
		cd1, err := NewLZ4CompressorDecompressor(CompressionModeDefault)
		require.NoError(t, err)
		require.NotNil(t, cd1)
		cd2, err := NewSnappyCompressorDecompressor(CompressionModeDefault)
		require.NoError(t, err)
		require.NotNil(t, cd2)

		return NewProcessorChain([]Processor{
			cd1,
			cd2,
		})
	}},
	{"compression:Snappy<->compression:LZ4<->compression:GZip", func(t *testing.T) Processor {
		cd1, err := NewSnappyCompressorDecompressor(CompressionModeDefault)
		require.NoError(t, err)
		require.NotNil(t, cd1)
		cd2, err := NewLZ4CompressorDecompressor(CompressionModeDefault)
		require.NoError(t, err)
		require.NotNil(t, cd2)
		cd3, err := NewGZipCompressorDecompressor(CompressionModeDefault)
		require.NoError(t, err)
		require.NotNil(t, cd3)

		return NewProcessorChain([]Processor{
			cd1,
			cd2,
			cd3,
		})
	}},
	{"compression:GZip<->compression:LZ4<->compression:Snappy", func(t *testing.T) Processor {
		cd1, err := NewGZipCompressorDecompressor(CompressionModeDefault)
		require.NoError(t, err)
		require.NotNil(t, cd1)
		cd2, err := NewLZ4CompressorDecompressor(CompressionModeDefault)
		require.NoError(t, err)
		require.NotNil(t, cd2)
		cd3, err := NewSnappyCompressorDecompressor(CompressionModeDefault)
		require.NoError(t, err)
		require.NotNil(t, cd3)

		return NewProcessorChain([]Processor{
			cd1,
			cd2,
			cd3,
		})
	}},
	{"fooPrefix<->encryption:aes_256", func() pcc {
		key := []byte(randomString(32))
		return func(t *testing.T) Processor {
			ed, err := NewAESEncrypterDecrypter(key)
			require.NoError(t, err)
			require.NotNil(t, ed)

			return NewProcessorChain([]Processor{
				fooPrefixProcessor{},
				ed,
			})
		}
	}()},
	{"encryption:aes_256<->fooPrefix", func() pcc {
		key := []byte(randomString(32))
		return func(t *testing.T) Processor {
			ed, err := NewAESEncrypterDecrypter(key)
			require.NoError(t, err)
			require.NotNil(t, ed)

			return NewProcessorChain([]Processor{
				ed,
				fooPrefixProcessor{},
			})
		}
	}()},
	{"fooPrefix<->encryption:aes_256<->fooPrefix", func() pcc {
		key := []byte(randomString(32))
		return func(t *testing.T) Processor {
			ed, err := NewAESEncrypterDecrypter(key)
			require.NoError(t, err)
			require.NotNil(t, ed)

			return NewProcessorChain([]Processor{
				fooPrefixProcessor{},
				ed,
				fooPrefixProcessor{},
			})
		}
	}()},

	// let's do some weird combinations, simply chaining AES encryption
	{"encryption:aes_256<->encryption:aes_256", func() pcc {
		key1 := []byte(randomString(32))
		key2 := []byte(randomString(32))
		return func(t *testing.T) Processor {
			ed1, err := NewAESEncrypterDecrypter(key1)
			require.NoError(t, err)
			require.NotNil(t, ed1)
			ed2, err := NewAESEncrypterDecrypter(key2)
			require.NoError(t, err)
			require.NotNil(t, ed2)

			return NewProcessorChain([]Processor{
				ed1,
				ed2,
			})
		}
	}()},
	{"encryption:aes_128<->encryption:aes_196<->encryption:aes_256", func() pcc {
		key1 := []byte(randomString(16))
		key2 := []byte(randomString(24))
		key3 := []byte(randomString(32))
		return func(t *testing.T) Processor {
			ed1, err := NewAESEncrypterDecrypter(key1)
			require.NoError(t, err)
			require.NotNil(t, ed1)
			ed2, err := NewAESEncrypterDecrypter(key2)
			require.NoError(t, err)
			require.NotNil(t, ed2)
			ed3, err := NewAESEncrypterDecrypter(key3)
			require.NoError(t, err)
			require.NotNil(t, ed3)

			return NewProcessorChain([]Processor{
				ed1,
				ed2,
				ed3,
			})
		}
	}()},

	// Production-like processor chains (as used in the pipeline code)
	{"compression:Snappy<->encryption:aes_256", func() pcc {
		key := []byte(randomString(32))
		return func(t *testing.T) Processor {
			cd, err := NewSnappyCompressorDecompressor(CompressionModeDefault)
			require.NoError(t, err)
			require.NotNil(t, cd)
			ed, err := NewAESEncrypterDecrypter(key)
			require.NoError(t, err)
			require.NotNil(t, ed)

			return NewProcessorChain([]Processor{
				cd,
				ed,
			})
		}
	}()},
	{"compression:LZ4<->encryption:aes_256", func() pcc {
		key := []byte(randomString(32))
		return func(t *testing.T) Processor {
			cd, err := NewLZ4CompressorDecompressor(CompressionModeDefault)
			require.NoError(t, err)
			require.NotNil(t, cd)
			ed, err := NewAESEncrypterDecrypter(key)
			require.NoError(t, err)
			require.NotNil(t, ed)

			return NewProcessorChain([]Processor{
				cd,
				ed,
			})
		}
	}()},
	{"compression:GZip<->encryption:aes_256", func() pcc {
		key := []byte(randomString(32))
		return func(t *testing.T) Processor {
			cd, err := NewGZipCompressorDecompressor(CompressionModeDefault)
			require.NoError(t, err)
			require.NotNil(t, cd)
			ed, err := NewAESEncrypterDecrypter(key)
			require.NoError(t, err)
			require.NotNil(t, ed)

			return NewProcessorChain([]Processor{
				cd,
				ed,
			})
		}
	}()}, {"compression:Snappy<->encryption:aes_128", func() pcc {
		key := []byte(randomString(16))
		return func(t *testing.T) Processor {
			cd, err := NewSnappyCompressorDecompressor(CompressionModeDefault)
			require.NoError(t, err)
			require.NotNil(t, cd)
			ed, err := NewAESEncrypterDecrypter(key)
			require.NoError(t, err)
			require.NotNil(t, ed)

			return NewProcessorChain([]Processor{
				cd,
				ed,
			})
		}
	}()},
	{"compression:LZ4<->encryption:aes_128", func() pcc {
		key := []byte(randomString(16))
		return func(t *testing.T) Processor {
			cd, err := NewLZ4CompressorDecompressor(CompressionModeDefault)
			require.NoError(t, err)
			require.NotNil(t, cd)
			ed, err := NewAESEncrypterDecrypter(key)
			require.NoError(t, err)
			require.NotNil(t, ed)

			return NewProcessorChain([]Processor{
				cd,
				ed,
			})
		}
	}()},
	{"compression:GZip<->encryption:aes_128", func() pcc {
		key := []byte(randomString(16))
		return func(t *testing.T) Processor {
			cd, err := NewGZipCompressorDecompressor(CompressionModeDefault)
			require.NoError(t, err)
			require.NotNil(t, cd)
			ed, err := NewAESEncrypterDecrypter(key)
			require.NoError(t, err)
			require.NotNil(t, ed)

			return NewProcessorChain([]Processor{
				cd,
				ed,
			})
		}
	}()},
}

func TestProcessorChain_ReadWrite(t *testing.T) {
	for _, testCase := range processorChainTestcases {
		t.Run(testCase.Name, func(t *testing.T) {
			chain := testCase.Constructor(t)
			testProcessorReadWrite(t, chain)
		})
	}
}

func TestProcessorChain_ReadWrite_MultiLayer(t *testing.T) {
	for _, testCase := range processorChainTestcases {
		t.Run(testCase.Name, func(t *testing.T) {
			chain := testCase.Constructor(t)
			testProcessorReadWriteMultiLayer(t, chain)
		})
	}
}

func TestProcessorChain_ReadWrite_Async(t *testing.T) {
	for _, testCase := range processorChainTestcases {
		t.Run(testCase.Name, func(t *testing.T) {
			testProcessorReadWriteAsync(t, func() Processor {
				return testCase.Constructor(t)
			})
		})
	}
}

func testProcessorReadWrite(t *testing.T, processor Processor) {
	t.Run("fixed-test-cases", func(t *testing.T) {
		testCases := []string{
			"a",
			"foo",
			"Hello, World!",
			"大家好",
			"This... is my finger :)",
		}
		for _, testCase := range testCases {
			testProcessorReadWriteCycle(t, processor, []byte(testCase))
		}
	})

	t.Run("random-test-cases", func(t *testing.T) {
		for i := 0; i < 4; i++ {
			inputData := make([]byte, mathRand.Int31n(256)+1)
			rand.Read(inputData)
			testProcessorReadWriteCycle(t, processor, inputData)
		}
	})
}

func testProcessorReadWriteCycle(t *testing.T, processor Processor, inputData []byte) {
	require := require.New(t)

	data, err := processor.WriteProcess(inputData)
	require.NoError(err)
	require.NotEmpty(data)

	outputData, err := processor.ReadProcess(data)
	require.NoError(err)
	require.Equal(inputData, outputData)
}

func testProcessorReadWriteMultiLayer(t *testing.T, processor Processor) {
	t.Run("fixed-test-cases", func(t *testing.T) {
		testCases := []string{
			"a",
			"foo",
			"Hello, World!",
			"大家好",
			"This... is my finger :)",
		}
		for _, testCase := range testCases {
			testProcessorReadWriteMultiLayerCycle(t, processor, []byte(testCase))
		}
	})

	t.Run("random-test-cases", func(t *testing.T) {
		for i := 0; i < 4; i++ {
			inputData := make([]byte, mathRand.Int31n(256)+1)
			rand.Read(inputData)
			testProcessorReadWriteMultiLayerCycle(t, processor, inputData)
		}
	})
}

func testProcessorReadWriteMultiLayerCycle(t *testing.T, processor Processor, inputData []byte) {
	for n := 2; n <= 4; n++ {
		t.Run(fmt.Sprintf("%d_times", n), func(t *testing.T) {
			require := require.New(t)

			var (
				err  error
				data = inputData
			)

			// write `n` times
			for i := 0; i < n; i++ {
				data, err = processor.WriteProcess(data)
				require.NoError(err)
				require.NotEmpty(data)

				// ensure to copy our data
				// in case the buffer is shared,
				// otherwise we're going to get weird results
				if processor.SharedWriteBuffer() {
					d := make([]byte, len(data))
					copy(d, data)
					data = d
				}
			}

			// read `n` times
			for i := 0; i < n; i++ {
				data, err = processor.ReadProcess(data)
				require.NoError(err)
				require.NotEmpty(data)

				// ensure to copy our data
				// in case the buffer is shared,
				// otherwise we're going to get weird results
				if processor.SharedReadBuffer() {
					d := make([]byte, len(data))
					copy(d, data)
					data = d
				}
			}

			// ensure the inputData equals the last-read data
			require.Equal(inputData, data)
		})
	}
}

func testProcessorReadWriteAsync(t *testing.T, pc func() Processor) {
	t.Run("fixed-test-cases", func(t *testing.T) {
		testProcessorReadWriteAsyncCycle(t, pc, [][]byte{
			[]byte("a"),
			[]byte("Hello, World!"),
			[]byte("大家好"),
			[]byte("This... is my finger :)"),
		})
	})

	t.Run("random-test-cases", func(t *testing.T) {
		var testCases [][]byte
		for i := 0; i < 4; i++ {
			testCase := make([]byte, mathRand.Int31n(256)+1)
			rand.Read(testCase)
			testCases = append(testCases, testCase)
		}
		testProcessorReadWriteAsyncCycle(t, pc, testCases)
	})
}

func testProcessorReadWriteAsyncCycle(t *testing.T, pc func() Processor, inputDataSlice [][]byte) {
	group, ctx := errgroup.WithContext(context.Background())

	inputLength := len(inputDataSlice)

	type (
		inputPackage struct {
			inputIndex int
		}

		processedPackage struct {
			inputIndex    int
			dataSliceCopy []byte
			dataSlice     []byte
		}

		outputPackage struct {
			inputIndex          int
			dataSliceCopy       []byte
			dataSlice           []byte
			outputDataSliceCopy []byte
			outputDataSlice     []byte
		}
	)

	// start the input goroutine, to give all the indices
	inputCh := make(chan inputPackage, inputLength)
	group.Go(func() error {
		defer close(inputCh)
		for index := 0; index < inputLength; index++ {
			select {
			case inputCh <- inputPackage{index}:
			case <-ctx.Done():
				return nil
			}
		}
		return nil
	})

	// start a write goroutine, using a write processor
	writeProcessor := pc()
	processedCh := make(chan processedPackage, inputLength)
	group.Go(func() error {
		defer close(processedCh)
		for input := range inputCh {
			inputData := inputDataSlice[input.inputIndex]

			data, err := writeProcessor.WriteProcess(inputData)
			if err != nil {
				return err
			}

			pkg := processedPackage{
				inputIndex: input.inputIndex,
			}
			pkg.dataSliceCopy = make([]byte, len(data))
			copy(pkg.dataSliceCopy, data)

			if writeProcessor.SharedWriteBuffer() {
				pkg.dataSlice = make([]byte, len(data))
				copy(pkg.dataSlice, data)
			} else {
				pkg.dataSlice = data
			}

			select {
			case processedCh <- pkg:
			case <-ctx.Done():
				return nil
			}
		}
		return nil
	})

	// start a read goroutine, using a read processor
	readProcessor := pc()
	outputCh := make(chan outputPackage, inputLength)
	group.Go(func() error {
		defer close(outputCh)
		for processed := range processedCh {
			if bytes.Compare(processed.dataSliceCopy, processed.dataSlice) != 0 {
				return fmt.Errorf("index %d: dataSliceCopy (%s) and dataSlice (%s) not equal any longer)",
					processed.inputIndex, processed.dataSliceCopy, processed.dataSlice)
			}

			outputData, err := readProcessor.ReadProcess(processed.dataSliceCopy)
			if err != nil {
				return err
			}

			inputData := inputDataSlice[processed.inputIndex]
			if bytes.Compare(inputData, outputData) != 0 {
				return fmt.Errorf("index %d: inputData (%s) outputData (%s) not equal)",
					processed.inputIndex, inputData, outputData)
			}

			pkg := outputPackage{
				inputIndex:    processed.inputIndex,
				dataSliceCopy: processed.dataSliceCopy,
				dataSlice:     processed.dataSlice,
			}
			pkg.outputDataSliceCopy = make([]byte, len(outputData))
			copy(pkg.outputDataSliceCopy, outputData)

			if readProcessor.SharedWriteBuffer() {
				pkg.outputDataSlice = make([]byte, len(outputData))
				copy(pkg.outputDataSlice, outputData)
			} else {
				pkg.outputDataSlice = outputData
			}

			select {
			case outputCh <- pkg:
			case <-ctx.Done():
				return nil
			}
		}
		return nil
	})

	// start a goroutine, simply to validate the final output of all received input
	group.Go(func() error {
		for output := range outputCh {
			if bytes.Compare(output.dataSliceCopy, output.dataSlice) != 0 {
				return fmt.Errorf("index %d: dataSliceCopy (%s) and dataSlice (%s) not equal any longer)",
					output.inputIndex, output.dataSliceCopy, output.dataSlice)
			}

			if bytes.Compare(output.outputDataSliceCopy, output.outputDataSlice) != 0 {
				return fmt.Errorf("index %d: outputDataSliceCopy (%s) and outputDataSlice (%s) not equal any longer)",
					output.inputIndex, output.outputDataSliceCopy, output.outputDataSlice)
			}

			inputData := inputDataSlice[output.inputIndex]
			if bytes.Compare(output.outputDataSlice, inputData) != 0 {
				return fmt.Errorf("index %d: output (%s) and input (%s) not equal any longer)",
					output.inputIndex, output.outputDataSlice, inputData)
			}
		}
		return nil
	})

	err := group.Wait()
	require.NoError(t, err)
}
