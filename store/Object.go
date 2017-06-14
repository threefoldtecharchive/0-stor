package main

import (
	"errors"
	"fmt"

	"gopkg.in/validator.v2"
)

const (
	FileSize = 1024 * 1024
	CRCSize  = 32
)

type Object struct {
	Data string `json:"data" validate:"nonzero"`
	Id   string `json:"id" validate:"min=5,max=128,regexp=^[a-zA-Z0-9]+$,nonzero"`
	Tags []Tag  `json:"tags,omitempty"`
}

type File struct {
	Reference byte
	CRC       [32]byte
	Payload   []byte
	Tags      []byte
}

func (s Object) Validate() error {

	return validator.Validate(s)
}

func (f *File) ToBytes() []byte {
	size := len(f.Payload) + CRCSize + 1
	result := make([]byte, size)
	// First byte is reference
	result[0] = f.Reference
	// Next 32 bytes CRC

	copy(result[1:CRCSize+1], f.CRC[:])
	// Next 1Mbs (file content)
	copy(result[CRCSize+1:], f.Payload)

	return result
}

func (f *File) FromBytes(data []byte) error {
	if len(data) > FileSize+CRCSize {
		return errors.New("Data size exceeds limits")
	} else if len(data) <= CRCSize {
		return errors.New(fmt.Sprintf("Invalid file size (%v) bytes", len(data)))
	}

	var crc [CRCSize]byte

	copy(crc[:], data[1:CRCSize+1])

	var maxIdx int

	if len(data) > FileSize+CRCSize+1 {
		maxIdx = FileSize + CRCSize
	} else {
		maxIdx = len(data) - 1
	}

	var payload = make([]byte, maxIdx-CRCSize)

	copy(payload, data[CRCSize+1:])

	f.Reference = data[0]
	f.CRC = crc
	f.Payload = payload

	return nil
}

func (f *File) ToObject(data []byte, Id string) *Object {
	return &Object{
		Id:   Id,
		Data: string(data[1:]),
	}
}

func (o *Object) ToFile(addReferenceByte bool) (*File, error) {
	file := &File{}
	var data []byte
	bytes := []byte(o.Data)

	// add reference
	if addReferenceByte {
		data = make([]byte, len(bytes)+1)
		data[0] = byte(1)
		copy(data[1:], bytes)
	} else {
		data = bytes
	}

	err := file.FromBytes(data)
	return file, err
}
