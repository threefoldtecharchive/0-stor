// Package encoding provides encoding and decoding logic
// for the data structures available in the root `server` package.
// In a functional way it allows you to encode/decode objects, namespaces, reference lists and stats.
// On top of that it will allow you to manipulate reference lists,
// without having to decode them first.
// When using these functions the result is encoded automatically for you.
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
// Please consultate the documentation of the individual Encoding methods,
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

	// ErrReferenceIDTooLarge is an error returned,
	// in case a reference is too large to be encoded,
	// and thus doesn't fit in the reference list's binary format.
	ErrReferenceIDTooLarge = errors.New("reference identifier too large")
)

const (
	// MaxReferenceIDLength defines the maximum length
	// a reference (identifier) can be.
	MaxReferenceIDLength = 255
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
// and thus we can concider it invalid before we put any extra work in.
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
// and thus we can concider it invalid before we put any extra work in.
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
// and thus we can concider it invalid before we put any extra work in.
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

// EncodeReferenceList encodes a reference list to a raw binary format:
//
//    +---------------+-------------------+
//    |     CRC32     |  1+ reference(s)  |
//    |     uint32    |   custom format   |
//    +---+---+---+---+---+-----------+---+
//    | 0 | 1 | 2 | 3 | 4 |    ...    | n |  # bytes
//    +---+---+---+---+---+-----------+---+
//
// Where the CRC32 is arranged in Little Endian Byte Order,
// and where each reference is encoded as:
//
//    +-------+-------------+
//    | size  | identifier  |
//    | uint8 |   []byte    |
//    +-------+---+-----+---+
//    |   0   | 1 | ... | n | # bytes
//    +---+---+---+-----+---+
//
// Where the size is arranged in Little Endian Byte Order,
// and where the identifier has a maximum length of 255,
// and thus the maximum number of bytes for a reference is 256.
//
// An error is returned in case
// a nil reference list was given, and thus nothing could be encoded.
//
// ErrReferenceIDTooLarge is returned as an error,
// in case a given reference is larger than ErrReferenceIDTooLarge.
func EncodeReferenceList(list server.ReferenceList) ([]byte, error) {
	data, err := encodeRefList(list, checksumSize)
	if err != nil {
		return nil, err
	}
	packageData(data)
	return data, nil
}

// AppendToEncodedReferenceList allows you to append a reference list,
// an already encoded reference list. This makes it a very cheap operation,
// as we do not have to first decode the already existing reference list.
// See `EncodeReferenceList` for more information about the encoding format of a ReferenceList.
//
// `ErrInvalidData` is returned in case the given data slice,
// is not big enough to hold a valid encoded ReferenceList,
// and thus we can concider it invalid before we put any extra work in.
//
// `ErrInvalidChecksum` is returned in case the given data package,
// contained a checksum which could not be matched with
// the freshly made data blob's checksum.
//
// ErrReferenceIDTooLarge is returned as an error,
// in case a given reference is larger than ErrReferenceIDTooLarge.
//
// An error is returned in case
// a nil reference list was given, and thus nothing could be encoded.
func AppendToEncodedReferenceList(data []byte, list server.ReferenceList) ([]byte, error) {
	const minDataSize = checksumSize + 1 // CRC32 + nilString
	length := len(data)
	if length < minDataSize {
		return nil, ErrInvalidData
	}

	// first unpackage the data, so we get just the data blob,
	// without the checksum
	blob, err := unpackageData(data)
	if err != nil {
		// invalid crc
		return nil, err
	}

	if len(list) == 0 {
		// no references to append,
		// and thus we can exit early
		return data, nil
	}

	// allocate enough space for the checksum, and the combination of the 2 reference lists.
	data, err = encodeRefList(list, length)
	if err != nil {
		// a given reference is too large
		return nil, err
	}

	// copy the old data into the new data
	copy(data[checksumSize:], blob)

	// package new data and return it
	packageData(data)
	return data, nil
}

// DecodeReferenceList decodes a reference list, encoded in a raw binary format.
// See `EncodeReferenceList` for more information about the encoding format.
//
// `ErrInvalidData` is returned in case the given data slice,
// is not big enough to hold a valid encoded ReferenceList,
// and thus we can concider it invalid before we put any extra work in.
// That same error is also returned in case the data was invalid for any other reason.
//
// `ErrInvalidChecksum` is returned in case the given data package,
// contained a checksum which could not be matched with
// the freshly made data blob's checksum.
func DecodeReferenceList(data []byte) (server.ReferenceList, error) {
	const minDataSize = checksumSize + 1 // CRC32 + NilString
	if len(data) < minDataSize {
		return nil, ErrInvalidData
	}

	// first unpackage the data, so we get just the data blob,
	// without the checksum
	blob, err := unpackageData(data)
	if err != nil {
		// invalid crc
		return nil, err
	}

	var (
		buf                             = newZeroAllocReadBuffer(blob)
		list                            server.ReferenceList
		ul                              uint8
		readLength, idBufLength, length int
		idBuf                           []byte
	)
	// read all strings one by one
	for !buf.Empty() {
		// no need to check for errors,
		// as no error can happen at this point with the binary reading of an uint8
		binary.Read(buf, binary.LittleEndian, &ul)
		// stop early in case the read length is 0
		if ul == 0 {
			list = append(list, "")
			continue
		}
		// grow identifier buffer if needed
		if length = int(ul); idBufLength < length {
			idBuf = make([]byte, (idBufLength+length)*2)
		}
		// read the actual string, and ensure that we read as much as we want
		readLength, err = buf.Read(idBuf[:length])
		if err != nil {
			// EOF
			return nil, ErrInvalidData
		}
		if readLength < length {
			return nil, ErrInvalidData
		}
		// identifier read and is valid
		list = append(list, string(idBuf[:length]))
	}

	return list, nil
}

// RemoveFromEncodedReferenceList decodes a reference list, encoded in a raw binary format.
// It removes the given reference list from the decoded list.
// If no elements were removed from the decoded list, the original data will be returned,
// otherwise the new smaller decoded reference list will be encoded and returned.
// See `EncodeReferenceList` for more information about the encoding format.
//
// `ErrInvalidData` is returned in case the given data slice,
// is not big enough to hold a valid encoded ReferenceList,
// and thus we can concider it invalid before we put any extra work in.
// That same error is also returned in case the data was invalid for any other reason.
//
// `ErrInvalidChecksum` is returned in case the given data package,
// contained a checksum which could not be matched with
// the freshly made data blob's checksum.
//
// A nil slice can be returned in a non-error case,
// if and only if the resulting list is empty, after removal of the other list.
func RemoveFromEncodedReferenceList(data []byte, other server.ReferenceList) ([]byte, int, error) {
	list, err := DecodeReferenceList(data)
	if err != nil {
		return nil, 0, err
	}

	if len(other) == 0 {
		// nothing to do, no references to remove
		return data, len(list), nil
	}

	// Remove the other list from the decoded list
	remaining := list.RemoveReferences(other)
	count := len(list)
	if len(remaining) == len(other) {
		// nothing to do, no references were removed
		return data, count, nil
	}

	if count == 0 {
		return nil, count, nil // nothing to do, and nothing to encode
	}

	// encode list and return the newly encoded data
	data, err = EncodeReferenceList(list)
	if err != nil {
		return nil, 0, err
	}
	return data, count, nil
}

func encodeRefList(list server.ReferenceList, offset int) ([]byte, error) {
	length := len(list)
	if length == 0 {
		return nil, errors.New("no references given to encode")
	}

	// add offset to the length
	length += offset

	// add the individual lengths of each string also to the total length
	var strLength int
	for _, str := range list {
		strLength = len(str)
		if strLength > MaxReferenceIDLength {
			return nil, ErrReferenceIDTooLarge
		}
		// add string length to the length
		length += strLength
	}

	var (
		// allocate total data slice
		data = make([]byte, length)
		buf  = newZeroAllocWriteBuffer(data[offset:])
	)
	// write each reference to the slice, starting at the offset,
	// in our bencode-modified string format
	for _, str := range list {
		// we control the entire buffer and input value (and its type),
		// so no need to check for errors here
		strLength = len(str)
		binary.Write(buf, binary.LittleEndian, uint8(strLength))

		// skip any further logic, in case the string length is 0
		if strLength == 0 {
			continue // nothing to write
		}

		// simply write the slice directly to the buffer,
		// again no need to check for errors here,
		// especially as our custom buffer returns panics if we screw up.
		buf.Write([]byte(str))
	}

	// return the entire data slice (which includes the encoded reference list)
	return data, nil
}

// ValidateData can be used to validate a data slice, encoded by this package.
// The resulting error is nil in case the given data is valid.
//
// `ErrInvalidData` is returned in case the given data slice,
// is not big enough to hold any valid encoded value,
// and thus we can concider it invalid before we put any extra work in.
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
