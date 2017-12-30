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

// Package encoding provides encoding and decoding logic
// for the data structures available in the root `server` package.
// In a functional way it allows you to encode/decode objects, namespaces and stats.
//
// This packages tries to allocate as little as possible.
// For encoding that means that for a single Encode call,
// only once an allocation happens, and that is to encode the returned data slice.
// For Decoding that means that the given data slice,
// is to be considered owned by the returned object, in case of a successful decode.
// In general this last statement only counts for objects which have byte slices as properties,
// but you can check the relevant Decode methods to be sure.
//
// All encoding formats, used by the Encoding/Decoding formats of this package,
// prefix the encoded data with a CRC32 checksum.
// This is only meant to protect against (accidental) data corruption.
// The checksum is validated at decoding time only,
// and is not returned ever to the user of this package.
//
// Please read the documentation of the individual Encoding methods,
// in order to learn more about the specific (binary) Encoding format,
// of each type supported by this package.
package encoding

import (
	"encoding/binary"
	"errors"
	"hash/crc32"

	"github.com/zero-os/0-stor/server"
)

var (
	// ErrInvalidChecksum is returned in case the decoded data,
	// has a different checksum, than the one stored
	// as part of the data blob.
	ErrInvalidChecksum = errors.New("decoded data had wrong checksum")

	// ErrInvalidData is an error returned in case a nil data slice,
	// or a given data slice which is smaller than expected,
	// is given to a Decode function of any kind.
	ErrInvalidData = errors.New("invalid data slice cannot be decoded")
)

// EncodeObject encodes an object to a raw binary format:
//
//    +---------------+-------------------+
//    |     CRC32     |    Data (Blob)    |
//    |     uint32    |      []byte       |
//    +---------------+-------------------+
//    | 0 | 1 | 2 | 3 |      ...      | n | # bytes
//    +-----------------------------------+
//
// Where everything is arranged in Little Endian Byte Order.
// Except for binary data ([]byte), which is written as it is.
//
// An error is only to be expected if the object's data is nil.
func EncodeObject(obj server.Object) ([]byte, error) {
	objDataLength := len(obj.Data)
	if objDataLength == 0 {
		return nil, errors.New("no data given to encode")
	}

	// create the final data slice (CRC+DatLength)
	data := make([]byte, checksumSize+objDataLength)

	// copy our data directly to this slice
	copy(data[checksumSize:], obj.Data)

	// package data (checksum + add that checksum)
	// and return the (encoded) data to the user
	packageData(data)
	return data, nil
}

// DecodeObject decodes an object, encoded in a raw binary format.
// See `EncodeObject` for more information about the encoding format.
//
// `ErrInvalidData` is returned in case the given data slice,
// is not big enough to hold a valid encoded Object,
// and thus we can consider it invalid before we put any extra work in.
//
// `ErrInvalidChecksum` is returned in case the given data package,
// contained a checksum which could not be matched with
// the freshly made data blob's checksum.
//
// NOTE: that the data slice given is owned by the returned Object,
// in case of a successful decoding call,
// and should no longer be used by the callee.
func DecodeObject(data []byte) (server.Object, error) {
	const minDecodeSize = checksumSize + 1 // CRC+DataProp
	if len(data) < minDecodeSize {
		return server.Object{}, ErrInvalidData
	}

	blob, err := unpackageData(data)
	if err != nil {
		// invalid crc
		return server.Object{}, err
	}
	// return the blob directly as the data,
	// as that's the only thing the blob contains
	return server.Object{Data: blob}, nil
}

// EncodeNamespace encodes a namespace to a raw binary format:
//
//    +---------------+-------------------+-------------------+
//    |     CRC32     |   Reserved Size   |       Label       |
//    |     uint32    |       uint64      |      []byte       |
//    +---+---+---+---+---+---+------+----+---------------+---+
//    | 0 | 1 | 2 | 3 | 4 | 5 |  ... | 11 |      ...      | n | # bytes
//    +---+---+---+---+---+---+------+----+---------------+---+
//
// Where everything is arranged in Little Endian Byte Order.
// Except for binary data ([]byte), which is written as it is.
//
// An error is only to be expected if the namespace's label is nil.
func EncodeNamespace(ns server.Namespace) ([]byte, error) {
	labelLength := len(ns.Label)
	if labelLength == 0 {
		return nil, errors.New("no label given to encode")
	}

	const (
		reservedLength = 8                             // ReservedSizeProp
		staticSize     = checksumSize + reservedLength // CRC+ReservedSizeProp
	)

	// create the final data slice (CRC+ReservedSizeProp+LabelProp)
	data := make([]byte, staticSize+labelLength)

	// copy our label directly
	copy(data[staticSize:], ns.Label)

	// write our reserved size prop (if needed)
	if ns.Reserved != 0 {
		// allocate write buffers, which does not take ownership of the input buffer,
		// nor does it ever allocate a new data slice.
		buf := newZeroAllocWriteBuffer(data[checksumSize:staticSize])
		// no need to check for errors as we 100% control
		// the write buffer and input value (and its type)
		binary.Write(buf, binary.LittleEndian, ns.Reserved)
	}

	// package data (checksum + add that checksum)
	// and return the (encoded) data to the user
	packageData(data)
	return data, nil
}

// DecodeNamespace decodes a namespace, encoded in a raw binary format.
// See `EncodeNamespace` for more information about the encoding format.
//
// `ErrInvalidData` is returned in case the given data slice,
// is not big enough to hold a valid encoded Namespace,
// and thus we can consider it invalid before we put any extra work in.
//
// `ErrInvalidChecksum` is returned in case the given data package,
// contained a checksum which could not be matched with
// the freshly made data blob's checksum.
//
// NOTE: that the data slice given is owned by the returned Namespace,
// in case of a successful decoding call,
// and should no longer be used by the callee.
func DecodeNamespace(data []byte) (server.Namespace, error) {
	const (
		reservedLength = 8                                 // ReservedSizeProp
		minDecodeSize  = checksumSize + reservedLength + 1 // CRC+ReserveSizeProp+LabelProp
	)

	if len(data) < minDecodeSize {
		return server.Namespace{}, ErrInvalidData
	}

	blob, err := unpackageData(data)
	if err != nil {
		// invalid crc
		return server.Namespace{}, err
	}

	var ns server.Namespace

	// read the reserved size prop
	buf := newZeroAllocReadBuffer(blob[:reservedLength])

	// we do not control the input data slice,
	// we do however know that the blob is of a correct size,
	// so no need to check for errors here
	binary.Read(buf, binary.LittleEndian, &ns.Reserved)

	// directly use the label as it is,
	// without copying or allocating anything new
	ns.Label = blob[reservedLength:]

	// return the decoded namespace
	return ns, nil
}

// EncodeStoreStat encodes store statistics to a raw binary format:
//
//    +---------------+-------------------+-------------------+
//    |     CRC32     |  Size Available   |     Size Used     |
//    |     uint32    |      uint64       |      uint64       |
//    +---+---+---+---+---+---+------+----+----+---------+----+
//    | 0 | 1 | 2 | 3 | 4 | 5 |  ... | 11 | 12 |   ...   | 19 | # bytes
//    +---+---+---+---+---+---+------+----+----+---------+----+
//
// Where everything is arranged in Little Endian Byte Order.
//
// No error is ever returned, as all properties are valid in their nil state.
func EncodeStoreStat(stats server.StoreStat) []byte {
	const propertySize = 16 // uint64 x 2
	// allocate the final output data
	data := make([]byte, checksumSize+propertySize)

	// write both properties to the already allocated data
	buf := newZeroAllocWriteBuffer(data[checksumSize:])
	// no need to check for errors, as everything is fully controlled by us
	binary.Write(buf, binary.LittleEndian, stats.SizeAvailable)
	binary.Write(buf, binary.LittleEndian, stats.SizeUsed)

	// package data and return it to the user
	packageData(data)
	return data
}

// DecodeStoreStat decodes store statistics, encoded in a raw binary format.
// See `EncodeStoreStat` for more information about the encoding format.
//
// `ErrInvalidData` is returned in case the given data slice,
// is not big enough to hold a valid encoded StoreStat,
// and thus we can consider it invalid before we put any extra work in.
//
// `ErrInvalidChecksum` is returned in case the given data package,
// contained a checksum which could not be matched with
// the freshly made data blob's checksum.
func DecodeStoreStat(data []byte) (server.StoreStat, error) {
	const staticDecodeSize = checksumSize + 16 // CRC (uint32) + Props (2 times uint64)
	if len(data) != staticDecodeSize {
		return server.StoreStat{}, ErrInvalidData
	}

	blob, err := unpackageData(data)
	if err != nil {
		// invalid crc
		return server.StoreStat{}, err
	}

	var stats server.StoreStat

	// read the properties,
	// we do not control the properties,
	// we know however that the length of the blob will be big enough
	buf := newZeroAllocReadBuffer(blob)
	binary.Read(buf, binary.LittleEndian, &stats.SizeAvailable)
	binary.Read(buf, binary.LittleEndian, &stats.SizeUsed)

	// return the decoded store statistics
	return stats, nil
}

// ValidateData can be used to validate a data slice, encoded by this package.
// The resulting error is nil in case the given data is valid.
//
// `ErrInvalidData` is returned in case the given data slice,
// is not big enough to hold any valid encoded value,
// and thus we can consider it invalid before we put any extra work in.
// That same error is also returned in case the data was invalid for any other reason.
//
// `ErrInvalidChecksum` is returned in case the given data package,
// contained a checksum which could not be matched with
// the freshly made data blob's checksum.
func ValidateData(data []byte) error {
	const minDataSize = checksumSize + 1
	if len(data) < minDataSize {
		return ErrInvalidData
	}

	// create validation checksum
	validCRC := crc32.Checksum(data[checksumSize:], crc32TablePolynomial)

	// read packaged checksum
	buf := newZeroAllocReadBuffer(data[:checksumSize])
	var packagedCRC uint32

	// we do not control the encoded data,
	// however an uint32 cannot be /not/ encoded,
	// as any combination of 4 bytes form a valid uint32,
	// whether that is the desired number is a different question
	binary.Read(buf, binary.LittleEndian, &packagedCRC)

	// validate checksum
	if validCRC != packagedCRC {
		return ErrInvalidChecksum
	}
	return nil
}

// packageData creates a CRC32 checksum of the data-blob-part of this data slice,
// and writes it in the raw binary format as the first 4 bytes of the given data slice.
func packageData(data []byte) {
	// create zero-alloc write buffer
	buf := newZeroAllocWriteBuffer(data[:checksumSize])

	// create the checksum
	crc := crc32.Checksum(data[checksumSize:], crc32TablePolynomial)

	// write the checksum (without allocating extra bytes),
	// an error should never happen, as we 100% control
	// the reader and the input value (and its type)
	binary.Write(buf, binary.LittleEndian, crc)
}

// unpackageData creates a CRC32 checksum of the data-blob-part of this data slice,
// and compares it with the stored CRC32 checksum (the first 4 bytes of this data slice).
// If those 2 checksum aren't equal, the ErrInvalidChecksum error will be returned.
// Otherwise no error is returned, and the returned slice will be equal to the data-blob-part of this slice.
func unpackageData(data []byte) ([]byte, error) {
	// create validation checksum
	blob := data[checksumSize:]
	validCRC := crc32.Checksum(blob, crc32TablePolynomial)

	// read packaged checksum
	buf := newZeroAllocReadBuffer(data[:checksumSize])
	var packagedCRC uint32

	// we do not control the encoded data,
	// however an uint32 cannot be /not/ encoded,
	// as any combination of 4 bytes form a valid uint32,
	// whether that is the desired number is a different question
	binary.Read(buf, binary.LittleEndian, &packagedCRC)

	// validate checksum
	if validCRC != packagedCRC {
		return nil, ErrInvalidChecksum
	}

	// return unpackaged data blob, still without allocating
	return blob, nil
}

const (
	// the size of all checksums created as part of this encoding package.
	checksumSize = 4
)

var (
	// CRC32Polynomial is the Polynomial used
	// to create the CRC32 checksums,
	// as part of all encoding/decoding functions of this package.
	CRC32Polynomial uint32 = 0xD5828281

	// table used to create a 32bit checksum for data,
	// prior to storage of a data (blob),
	// as well as after the decoding, as to ensure that the data is correct.
	crc32TablePolynomial = crc32.MakeTable(CRC32Polynomial)
)
