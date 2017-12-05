package encoding

import "io"

func newZeroAllocWriteBuffer(buf []byte) *zeroAllocWriteBuffer {
	length := len(buf)
	if length == 0 {
		panic("no buffer given to write to")
	}

	return &zeroAllocWriteBuffer{
		buf:    buf,
		length: length,
		offset: 0,
	}
}

type zeroAllocWriteBuffer struct {
	buf            []byte
	length, offset int
}

// Write implements io.Writer.Write
func (zab *zeroAllocWriteBuffer) Write(p []byte) (n int, err error) {
	length := len(p)
	if length == 0 {
		panic("nil slice given")
	}

	if length+zab.offset > zab.length {
		panic("out of bounds")
	}
	copy(zab.buf, p)
	zab.buf = zab.buf[length:]
	zab.offset += length
	return length, nil
}

func newZeroAllocReadBuffer(buf []byte) *zeroAllocReadBuffer {
	length := len(buf)
	if length == 0 {
		panic("no buffer given to read from")
	}

	return &zeroAllocReadBuffer{
		buf:    buf,
		length: length,
	}
}

type zeroAllocReadBuffer struct {
	buf    []byte
	length int
}

// Read implements io.Reader.Read
func (zab *zeroAllocReadBuffer) Read(p []byte) (n int, err error) {
	length := len(p)
	if length == 0 {
		panic("nil slice given")
	}

	// if we have no data left, return an EOF error
	if zab.length == 0 {
		return 0, io.EOF
	}

	// copy data from the buffer directly to the given slice
	copy(p, zab.buf)

	// cap the length to our available length
	if length > zab.length {
		length = zab.length
	}

	// update our internals
	zab.buf = zab.buf[length:]
	zab.length -= length

	// return the length read
	return length, nil
}

// Empty returns true if the buffer has nothing left to read,
// and thus the buffer can be considered empty.
func (zab *zeroAllocReadBuffer) Empty() bool {
	return zab.length == 0
}
