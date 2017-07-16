package models

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	"strings"

	validator "gopkg.in/validator.v2"
)

const (
	FileSize = 1024 * 1024
	CRCSize  = 32
)

type File struct {
	Namespace string
	Id        string
	Reference byte
	CRC       [32]byte
	Payload   []byte
	Tags      []byte
}

func (f File) Key() string {
	label := f.Namespace
	if strings.Index(label, NAMESPACE_PREFIX) != -1 {
		label = strings.Replace(label, NAMESPACE_PREFIX, "", 1)
	}
	return fmt.Sprintf("%s:%s", label, f.Id)
}

func (f *File) Encode() ([]byte, error) {
	id := []byte(f.Id)
	idSize := len(id)

	ns := []byte(f.Namespace)
	nsSize := len(ns)

	payloadSize := len(f.Payload)

	tagSize := len(f.Tags)

	size := 8 + 1 + idSize + nsSize + payloadSize + tagSize + CRCSize

	result := make([]byte, size)
	binary.LittleEndian.PutUint16(result[0:2], uint16(idSize))
	binary.LittleEndian.PutUint16(result[2:4], uint16(nsSize))
	binary.LittleEndian.PutUint16(result[4:6], uint16(payloadSize))
	binary.LittleEndian.PutUint16(result[6:8], uint16(tagSize))

	start := 8
	end := start + idSize
	copy(result[start:end], id)

	start = end
	end = start + nsSize
	copy(result[start:end], ns)

	result[end] = f.Reference

	// Next 32 bytes CRC
	start = end + 1
	end = start + CRCSize
	copy(result[start:end], f.CRC[:])

	start = end
	end = start + payloadSize
	// Next 1Mbs (file content)
	copy(result[start:end], f.Payload)

	start = end
	end = start + tagSize
	copy(result[start:end], f.Tags)
	return result, nil
}

func (f *File) Size() float64 {
	return math.Ceil((float64(len(f.Payload)) / (1024.0 * 1024.0)))
}

func (f *File) Decode(data []byte) error {
	if len(data) > FileSize+CRCSize {
		return errors.New("Data size exceeds limits")
	} else if len(data) <= CRCSize {
		return errors.New(fmt.Sprintf("Invalid file size (%v) bytes", len(data)))
	}

	idSize := int16(binary.LittleEndian.Uint16(data[0:2]))
	nsSize := int16(binary.LittleEndian.Uint16(data[2:4]))
	payloadSize := int16(binary.LittleEndian.Uint16(data[4:6]))
	tagSize := int16(binary.LittleEndian.Uint16(data[6:8]))

	start := int16(8)
	end := start + idSize
	f.Id = string(data[start:end])

	start = end
	end = start + nsSize
	f.Namespace = string(data[start:end])

	f.Reference = data[end]

	start = end + 1
	end = start + int16(CRCSize)
	var crc [CRCSize]byte
	copy(crc[:], data[start:end])
	f.CRC = crc

	start = end
	end = start + payloadSize
	copy(f.Payload, data[start:end])

	start = end
	end = start + tagSize
	copy(f.Tags, data[start:end])
	return nil
}

type Object struct {
	Data string `json:"data" validate:"nonzero"`
	Id   string `json:"id" validate:"min=5,max=128,regexp=^[a-zA-Z0-9]+$,nonzero"`
	Tags []Tag  `json:"tags,omitempty"`
}

func (o Object) Validate() error {
	return validator.Validate(o)
}

func (o Object) Key() string {
	return o.Id
}

func (o *Object) ToFile(nsid string) (*File, error) {

	file := &File{
		Namespace: nsid,
		Id:        o.Id,
		Reference: 1,
		Tags:      []byte{},
	}

	bytes := []byte(o.Data)

	if len(bytes) <= CRCSize{
		return nil, errors.New("File contents must be greater than 32 bytes")
	}

	var crc [CRCSize]byte

	copy(crc[:], bytes[0:32])

	file.CRC = crc
	file.Payload = bytes[32:]

	return file, nil
}

type ObjectCreate struct{}

func (s ObjectCreate) Validate() error {
	return validator.Validate(s)
}

type ObjectUpdate struct {
	Data string `json:"data" validate:"nonzero"`
	Tags []Tag  `json:"tags,omitempty"`
}

func (s ObjectUpdate) Validate() error {
	return validator.Validate(s)
}

func (o *ObjectUpdate) ToFile(nsid string) (*File, error) {
	obj := &Object{
		Data: o.Data,
		Tags: o.Tags,
	}
	return obj.ToFile(nsid)

}
