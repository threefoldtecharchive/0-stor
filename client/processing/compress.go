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
	"compress/gzip"
	"fmt"
	"io"

	"github.com/golang/snappy"
	"github.com/pierrec/lz4"
	log "github.com/sirupsen/logrus"
)

// NewCompressorDecompressor returns a new instance for the given compression type.
// If the compression type is not registered as a valid compression type, an error is returned.
//
// The given compression mode is required, but the type's default value can be used.
func NewCompressorDecompressor(ct CompressionType, mode CompressionMode) (Processor, error) {
	cdc, ok := _CompressionTypeValueToConstructorMapping[ct]
	if !ok {
		return nil, fmt.Errorf("'%s' is not a valid/registered CompressionType value", ct)
	}
	return cdc(mode)
}

// NewSnappyCompressorDecompressor creates a new compressor-decompressor processor,
// using google's Snappy compression implementation in Golang.
//
// See SnappyCompressorDecompressor for more information.
func NewSnappyCompressorDecompressor(cm CompressionMode) (*SnappyCompressorDecompressor, error) {
	if cm != CompressionModeDefault {
		log.Warningf("Snappy does not support compression mode, ignoring desired mode: '%s'", cm)
	}
	return &SnappyCompressorDecompressor{
		readBuffer:  bytes.NewBuffer(nil),
		writeBuffer: bytes.NewBuffer(nil),
	}, nil
}

// SnappyCompressorDecompressor defines a processor, which compresses and decompresses,
// using google's Snappy compression implementation in Golang.
//
// It will compress text to compressed text while writing,
// and it will decompress compress text to (uncompressed) text while reading.
//
// See github.com/golang/snappy for more information about the
// technical details beyind this compressor-decompressor type.
type SnappyCompressorDecompressor struct {
	readBuffer, writeBuffer *bytes.Buffer
}

// WriteProcess implements Processor.WriteProcess
//
// input data gets compressed, and returned as compressed output data
func (cd *SnappyCompressorDecompressor) WriteProcess(data []byte) ([]byte, error) {
	cd.writeBuffer.Reset()
	w := snappy.NewWriter(cd.writeBuffer)
	_, err := io.Copy(w, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	return cd.writeBuffer.Bytes(), nil
}

// ReadProcess implements Processor.ReadProcess
//
// input data gets decompressed, and returned
// as the decompressed (and uncompressed?) output data
func (cd *SnappyCompressorDecompressor) ReadProcess(data []byte) ([]byte, error) {
	r := snappy.NewReader(bytes.NewReader(data))
	cd.readBuffer.Reset()
	_, err := io.Copy(cd.readBuffer, r)
	if err != nil {
		return nil, err
	}
	return cd.readBuffer.Bytes(), nil
}

// SharedWriteBuffer implements Processor.SharedWriteBuffer
func (cd *SnappyCompressorDecompressor) SharedWriteBuffer() bool { return true }

// SharedReadBuffer implements Processor.SharedReadBuffer
func (cd *SnappyCompressorDecompressor) SharedReadBuffer() bool { return true }

// NewLZ4CompressorDecompressor creates a new compressor-decompressor processor,
// using the LZ4 compression algorithm, implemented by Pierre Curto.
//
// See LZ4CompressorDecompressor for more information.
func NewLZ4CompressorDecompressor(cm CompressionMode) (*LZ4CompressorDecompressor, error) {
	if cm != CompressionModeDefault {
		log.Warningf("LZ4 does not support compression mode, ignoring desired mode: '%s'", cm)
	}
	return &LZ4CompressorDecompressor{
		readBuffer:  bytes.NewBuffer(nil),
		writeBuffer: bytes.NewBuffer(nil),
	}, nil
}

// LZ4CompressorDecompressor defines a processor, which compresses and decompresses,
// using the LZ4 compression algorithm, implemented by Pierre Curto.
//
// It will compress text to compressed text while writing,
// and it will decompress compress text to (uncompressed) text while reading.
//
// See github.com/pierrec/lz4 for more information about the
// technical details beyind this compressor-decompressor type.
type LZ4CompressorDecompressor struct {
	readBuffer, writeBuffer *bytes.Buffer
}

// WriteProcess implements Processor.WriteProcess
//
// input data gets compressed, and returned as compressed output data
func (cd *LZ4CompressorDecompressor) WriteProcess(data []byte) ([]byte, error) {
	cd.writeBuffer.Reset()
	w := lz4.NewWriter(cd.writeBuffer)
	w.Header.BlockDependency = true
	_, err := w.Write(data)
	if err != nil {
		return nil, err
	}
	err = w.Close()
	if err != nil {
		return nil, err
	}
	return cd.writeBuffer.Bytes(), nil
}

// ReadProcess implements Processor.ReadProcess
//
// input data gets decompressed, and returned
// as the decompressed (and uncompressed?) output data
func (cd *LZ4CompressorDecompressor) ReadProcess(data []byte) ([]byte, error) {
	r := lz4.NewReader(bytes.NewReader(data))
	cd.readBuffer.Reset()
	_, err := io.Copy(cd.readBuffer, r)
	if err != nil {
		return nil, err
	}
	return cd.readBuffer.Bytes(), nil
}

// SharedWriteBuffer implements Processor.SharedWriteBuffer
func (cd *LZ4CompressorDecompressor) SharedWriteBuffer() bool { return true }

// SharedReadBuffer implements Processor.SharedReadBuffer
func (cd *LZ4CompressorDecompressor) SharedReadBuffer() bool { return true }

// NewGZipCompressorDecompressor creates a new compressor-decompressor processor,
// using Golang's standard gzip implementation.
//
// See GZipCompressorDecompressor for more information.
func NewGZipCompressorDecompressor(cm CompressionMode) (*GZipCompressorDecompressor, error) {
	level, ok := _GZipCompressionModeMapping[cm]
	if !ok {
		log.Warningf("GZip does not support compression mode '%s', "+
			"defaulting to '%s'", cm, CompressionModeDefault)
		level = _GZipCompressionModeMapping[CompressionModeDefault]
	}

	return &GZipCompressorDecompressor{
		level:       level,
		readBuffer:  bytes.NewBuffer(nil),
		writeBuffer: bytes.NewBuffer(nil),
	}, nil
}

var _GZipCompressionModeMapping = map[CompressionMode]int{
	CompressionModeBestSpeed:       gzip.BestSpeed,
	CompressionModeBestCompression: gzip.BestCompression,
	CompressionModeDefault:         gzip.DefaultCompression,
}

// GZipCompressorDecompressor defines a processor, which compresses and decompresses,
// using Golang's standard gzip implementation.
//
// It will compress text to compressed text while writing,
// and it will decompress compress text to (uncompressed) text while reading.
//
// The CompressionMode is mapped directly to the gzip mode, with the same name.
// When the given CompressionMode couldn't be recognized,
// the gzip.BestSpeed mode will be used.
//
// See compress/gzip (Golang STD package) for more information about the
// technical details beyind this compression type.
type GZipCompressorDecompressor struct {
	level                   int
	readBuffer, writeBuffer *bytes.Buffer
}

// WriteProcess implements Processor.WriteProcess
//
// input data gets compressed, and returned as compressed output data
func (cd *GZipCompressorDecompressor) WriteProcess(data []byte) ([]byte, error) {
	cd.writeBuffer.Reset()
	w, err := gzip.NewWriterLevel(cd.writeBuffer, cd.level)
	if err != nil {
		return nil, err
	}
	_, err = w.Write(data)
	if err != nil {
		return nil, err
	}
	err = w.Close()
	if err != nil {
		return nil, err
	}
	return cd.writeBuffer.Bytes(), nil
}

// ReadProcess implements Processor.ReadProcess
//
// input data gets decompressed, and returned
// as the decompressed (and uncompressed?) output data
func (cd *GZipCompressorDecompressor) ReadProcess(data []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	cd.readBuffer.Reset()
	_, err = io.Copy(cd.readBuffer, r)
	if err != nil {
		return nil, err
	}
	return cd.readBuffer.Bytes(), nil
}

// SharedWriteBuffer implements Processor.SharedWriteBuffer
func (cd *GZipCompressorDecompressor) SharedWriteBuffer() bool { return true }

// SharedReadBuffer implements Processor.SharedReadBuffer
func (cd *GZipCompressorDecompressor) SharedReadBuffer() bool { return true }

var (
	_ Processor = (*SnappyCompressorDecompressor)(nil)
	_ Processor = (*LZ4CompressorDecompressor)(nil)
	_ Processor = (*GZipCompressorDecompressor)(nil)
)

func init() {
	// register all our standard compression types
	RegisterCompressorDecompressor(CompressionTypeSnappy, "Snappy",
		func(mode CompressionMode) (Processor, error) {
			return NewSnappyCompressorDecompressor(mode)
		})
	RegisterCompressorDecompressor(CompressionTypeLZ4, "LZ4",
		func(mode CompressionMode) (Processor, error) {
			return NewLZ4CompressorDecompressor(mode)
		})
	RegisterCompressorDecompressor(CompressionTypeGZip, "GZip",
		func(mode CompressionMode) (Processor, error) {
			return NewGZipCompressorDecompressor(mode)
		})
}
