package chunker

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestChunkerEven test chunker with data length == multiply of chunkSize
func TestChunkerEven(t *testing.T) {
	dataLen := 100
	chunkSize := 10

	data := make([]byte, dataLen)
	c := NewChunker(data, chunkSize)
	testSplitterEven(t, c, dataLen, chunkSize)
}

func TestReaderEven(t *testing.T) {
	dataLen := 100
	chunkSize := 10

	data := make([]byte, dataLen)
	c := NewReader(bytes.NewReader(data), chunkSize)
	testSplitterEven(t, c, dataLen, chunkSize)
}

func testSplitterEven(t *testing.T, s splitter, dataLen, chunkSize int) {
	chunkNum := 0

	for s.Next() {
		chunkNum++
		val := s.Value()
		assert.Equal(t, chunkSize, len(val))
	}
	assert.Equal(t, dataLen/chunkSize, chunkNum)
}

func TestChunkerNotEven(t *testing.T) {
	dataLen := 99
	chunkSize := 10

	data := make([]byte, dataLen)

	c := NewChunker(data, chunkSize)
	testSplitterNotEven(t, c, dataLen, chunkSize)
}

func TestReaderNotEven(t *testing.T) {
	dataLen := 99
	chunkSize := 10

	data := make([]byte, dataLen)

	c := NewReader(bytes.NewReader(data), chunkSize)
	testSplitterNotEven(t, c, dataLen, chunkSize)
}

// testSplitterNotEven test splitter with data length !=  multiply of chunkSize
func testSplitterNotEven(t *testing.T, s splitter, dataLen, chunkSize int) {
	expectedChunkNum := (dataLen / chunkSize) + 1
	chunkNum := 0

	for s.Next() {
		chunkNum++
		val := s.Value()
		if chunkNum != expectedChunkNum {
			assert.Equal(t, chunkSize, len(val))
		} else {
			assert.Equal(t, dataLen%chunkSize, len(val))
		}
	}
	assert.Equal(t, (dataLen/chunkSize)+1, chunkNum)
}
