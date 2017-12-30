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

// Processor defines the interface which can process data.
// It processes data in two directions, read and write.
//
// The processor implementation has to guarantee,
// that data that was written, can also be read by it.
// And thus the ReadProcess can be seen as the reverse WritePRocess.
// In other words, the following has to be true for all processors:
//
//    require.NotNil(inputData)
//    require.NotNil(processor)
//    processedData, err := processor.WriteProcess(inputData)
//    if err == nil {
//        outputData, err := processor.ReadProcess(processedData)
//        if err == nil {
//            require.Equal(inputData, outputData)
//        }
//    }
//
// A Processor is /NEVER/ thread-safe,
// and should only ever be used on /ONE/ goroutine at a time.
// If you wish to hash on multiple goroutines,
// you'll have to create one Processor (instance) per goroutine.
type Processor interface {
	// WriteProcess processes data in the write direction.
	//
	// The input byte slice is used as read-only data.
	//
	// Should you want to want to take ownership of the the output slice,
	// you'll have to copy the output data slice,
	// if and only if the `SharedWriteBuffer` of this Processor returns true,
	// as this means the allocated memory for that slice
	// is shared between sequential WriteProcess calls.
	// See SharedWriteBuffer for more information.
	WriteProcess(input []byte) (output []byte, err error)

	// ReadProcess processes data in the read direction,
	// and thus takes in processed data as input,
	// in other words data that was previously processed
	// by the WriteProcess method of an instance of this Processor type.
	//
	// The input byte slice is used as read-only data.
	//
	// Should you want to want to take ownership of the the output slice,
	// you'll have to copy the output data slice,
	// if and only if the `SharedReadBuffer` of this Processor returns true,
	// as this means the allocated memory for that slice
	// is shared between sequential ReadProcess calls.
	// See SharedReadBuffer for more information.
	ReadProcess(input []byte) (output []byte, err error)

	// SharedWriteBuffer returns true in case
	// the internal buffer is reused for sequential SharedWriteBuffer calls.
	// Meaning that the output data returned by WriteProcess
	// will remain owned by this Processor,
	// and should be either consumed as read-only and instantly,
	// or copied to a different slice in memory right after the data has been received.
	SharedWriteBuffer() bool

	// SharedReadBuffer returns true in case
	// the internal buffer is reused for sequential ReadProcess calls.
	// Meaning that the output data returned by ReadProcess
	// will remain owned by this Processor,
	// and should be either consumed as read-only and instantly,
	// or copied to a different slice in memory right after the data has been received.
	SharedReadBuffer() bool
}

// NopProcessor implements the Processor interface,
// but does not do any processing whatsoever.
// Instead it returns the data it receives.
//
// This implementation can be used in case you are required to give a Processor,
// in some location, but have no desire to do any processing yourself.
type NopProcessor struct{}

// WriteProcess implements Processor.WriteProcess
func (nop NopProcessor) WriteProcess(data []byte) ([]byte, error) { return data, nil }

// ReadProcess implements Processor.WriteProcess
func (nop NopProcessor) ReadProcess(data []byte) ([]byte, error) { return data, nil }

// SharedWriteBuffer implements Processor.SharedWriteBuffer
func (nop NopProcessor) SharedWriteBuffer() bool { return false }

// SharedReadBuffer implements Processor.SharedReadBuffer
func (nop NopProcessor) SharedReadBuffer() bool { return false }

// NewProcessorChain creates a new processor chain.
// At least 2 processors have to be given, or else NewProcessorChain panics.
// See ProcessorChain for more about information about the type.
func NewProcessorChain(processors []Processor) *ProcessorChain {
	processorLength := len(processors)
	if processorLength < 2 {
		panic("ProcessorChain requires at least two underlying processors")
	}
	return &ProcessorChain{
		processors:        processors,
		processorMaxIndex: processorLength - 1,
	}
}

// ProcessorChain can be used to chain multiple processor together.
// It will process data using all given processors in sequence,
// in the order that they are given, when writing.
// When reading the order of processors will be reverse as the one given,
// this to ensure that what has been written with this chain can also be read again.
type ProcessorChain struct {
	processors        []Processor
	processorMaxIndex int
}

// WriteProcess implements Processor.WriteProcess
//
// Processes the given data in the write direction,
// using all the given processors in the order they're given,
// starting with the first and ending with the last.
//
// If an error happens at any point,
// this method will short circuit and return that error immediately.
func (chain *ProcessorChain) WriteProcess(data []byte) ([]byte, error) {
	var err error
	for _, processor := range chain.processors {
		data, err = processor.WriteProcess(data)
		if err != nil {
			return nil, err
		}
	}
	return data, nil
}

// ReadProcess implements Processor.ReadProcess
//
// Processes the given data in the read direction,
// using all the given processors in the reverse order they're given,
// starting with the last and ending with the first.
//
// If an error happens at any point,
// this method will short circuit and return that error immediately.
func (chain *ProcessorChain) ReadProcess(data []byte) ([]byte, error) {
	var err error
	for i := chain.processorMaxIndex; i >= 0; i-- {
		data, err = chain.processors[i].ReadProcess(data)
		if err != nil {
			return nil, err
		}
	}
	return data, nil
}

// SharedWriteBuffer implements Processor.SharedWriteBuffer
func (chain *ProcessorChain) SharedWriteBuffer() bool {
	// the last processor in
	return chain.processors[chain.processorMaxIndex].SharedWriteBuffer()
}

// SharedReadBuffer implements Processor.SharedReadBuffer
func (chain *ProcessorChain) SharedReadBuffer() bool {
	// the last processor in
	return chain.processors[0].SharedReadBuffer()
}

var (
	_ Processor = NopProcessor{}
	_ Processor = (*ProcessorChain)(nil)
)
