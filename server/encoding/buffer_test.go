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

package encoding

import (
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestZeroAllocWriteBuffer(t *testing.T) {
	require := require.New(t)

	require.Panics(func() {
		newZeroAllocWriteBuffer(nil)
	}, "should panic when a nil slice is given")

	// create a 3-element slice, and use it to create a zero-alloc write buffer
	slice := make([]byte, 3)
	buf := newZeroAllocWriteBuffer(slice)
	require.NotNil(buf)

	// write in sequential order
	n, err := buf.Write([]byte{1, 2})
	require.Equal(2, n, "2 bytes should have been written")
	require.NoError(err)
	n, err = buf.Write([]byte{3})
	require.Equal(1, n, "1 byte should have been written")
	require.NoError(err)

	require.Panics(func() {
		buf.Write([]byte{4})
	}, "out of bounds")
	require.Panics(func() {
		buf.Write(nil)
	}, "nil slice")

	// now ensure our underlying slice is filled as we expected
	require.Equal([]byte{1, 2, 3}, slice)

	// now let's try to write out-of order, using 2 different buffers, of the same slice
	bufA := newZeroAllocWriteBuffer(slice[:1])
	require.NotNil(bufA)
	bufB := newZeroAllocWriteBuffer(slice[1:])
	require.NotNil(bufB)

	// write to the buffers the data
	n, err = bufB.Write([]byte{12, 13})
	require.Equal(2, n, "2 bytes should have been written")
	require.NoError(err)
	n, err = bufA.Write([]byte{11})
	require.Equal(1, n, "1 byte should have been written")
	require.NoError(err)

	// both buffers are now full, so shouldn't be able to write to any of them
	require.Panics(func() {
		bufA.Write([]byte{4})
	}, "out of bounds")
	require.Panics(func() {
		bufB.Write([]byte{4})
	}, "out of bounds")
	// writing nothing however, should not return an error (even if the buffer is full)
	require.Panics(func() {
		bufA.Write(nil)
	}, "nil slice")
	require.Panics(func() {
		bufB.Write(nil)
	}, "nil slice")

	// now ensure our underlying slice is filled as we expected
	require.Equal([]byte{11, 12, 13}, slice)
}

func TestZeroAllocReadBuffer(t *testing.T) {
	require := require.New(t)

	require.Panics(func() {
		newZeroAllocReadBuffer(nil)
	}, "should panic when a nil slice is given")

	// create a buffer for testing purposes
	inputSlice := []byte{1, 2, 3, 4, 5}
	buf := newZeroAllocReadBuffer(inputSlice)
	require.NotNil(buf)

	// reading nothing should panic
	require.Panics(func() {
		buf.Read(nil)
	}, "nil slice")

	// create slice for testing purposes
	slice := make([]byte, 2)

	// reading nothing should panic
	require.Panics(func() {
		buf.Read(nil)
	}, "nil slice")

	// read one byte, and validate the result
	n, err := buf.Read(slice[:1])
	require.NoError(err)
	require.Equal(1, n)
	require.Equal([]byte{1, 0}, slice)

	// read another byte, and validate the result
	n, err = buf.Read(slice[1:])
	require.NoError(err)
	require.Equal(1, n)
	require.Equal([]byte{1, 2}, slice)

	// read 2 bytes
	n, err = buf.Read(slice)
	require.NoError(err)
	require.Equal(2, n)
	require.Equal([]byte{3, 4}, slice)

	require.False(buf.Empty())

	// read 1 byte (even though we provide space for 2 in our given slice)
	// this because only 1 byte is left (to be read) in the buffer's internal slice
	n, err = buf.Read(slice)
	require.NoError(err)
	require.Equal(1, n)
	require.Equal([]byte{5, 4}, slice)

	require.True(buf.Empty())

	// ensure our input slice is untouched
	require.Len(inputSlice, 5)
	require.Equal([]byte{1, 2, 3, 4, 5}, inputSlice)

	// reading nothing should still panic
	require.Panics(func() {
		buf.Read(nil)
	}, "nil slice")

	// trying to read more, shouldn't panic, and instead just return io.EOF
	n, err = buf.Read(slice)
	require.Equal(io.EOF, err)
	require.Equal(0, n)

	require.True(buf.Empty())
}
