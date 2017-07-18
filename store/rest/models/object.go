package models

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strings"
	validator "gopkg.in/validator.v2"
	"hash/crc32"
)

const (
	FileSize = 1024 * 1024
	POLYNOMIAL = 0xD5828281
)

type File struct {
	Namespace string
	Id        string
	Reference byte
	CRC       string
	Payload   string
	Tags      []Tag
}

func (f File) Key() string {
	label := f.Namespace
	if strings.Index(label, NAMESPACE_PREFIX) != -1 {
		label = strings.Replace(label, NAMESPACE_PREFIX, "", 1)
	}
	return fmt.Sprintf("%s:%s", label, f.Id)
}

func(f File) ToObject() (*Object, error){
	return &Object{
		Id:   f.Id,
		Tags: f.Tags,
		Data: f.Payload,
	}, nil
}

func (f *File) Encode() ([]byte, error) {
	f.CRC = fmt.Sprint(crc32.Checksum([]byte(f.Payload), crc32.MakeTable(POLYNOMIAL)))

	id := []byte(f.Id)
	idSize := len(id)

	ns := []byte(f.Namespace)
	nsSize := len(ns)

	pl := []byte(f.Payload)
	plSize := len(pl)

	crc := []byte(f.CRC)
	crcSize:= len(crc)

	t := new(Tags)
	t.Tags = f.Tags
	tags, err := t.Encode()

	if err != nil{
		return []byte{}, nil
	}

	tagsSize := len(tags)

	size := 10 + 1 + idSize + nsSize + plSize + tagsSize + crcSize

	result := make([]byte, size)
	binary.LittleEndian.PutUint16(result[0:2], uint16(idSize))
	binary.LittleEndian.PutUint16(result[2:4], uint16(nsSize))
	binary.LittleEndian.PutUint16(result[4:6], uint16(crcSize))
	binary.LittleEndian.PutUint16(result[6:8], uint16(plSize))
	binary.LittleEndian.PutUint16(result[8:10], uint16(tagsSize))

	start := 10
	end := start + idSize
	copy(result[start:end], id)

	start = end
	end = start + nsSize
	copy(result[start:end], ns)

	result[end] = f.Reference

	start = end + 1
	end = start + crcSize
	copy(result[start:end], crc)

	start = end
	end = start + plSize
	copy(result[start:end], pl)

	start = end
	end = start + tagsSize
	copy(result[start:end], tags)
	return result, nil
}

func (f *File) Size() float64 {
	return math.Ceil((float64(len(f.Payload)) / (1024.0 * 1024.0)))
}

func (f *File) Decode(data []byte) error {
	if len(data) > FileSize {
		msg := fmt.Sprintf("Data size exceeds max limit (%s)", FileSize)
		return errors.New(msg)
	}

	idSize := int16(binary.LittleEndian.Uint16(data[0:2]))
	nsSize := int16(binary.LittleEndian.Uint16(data[2:4]))
	crcSize := int16(binary.LittleEndian.Uint16(data[4:6]))
	plSize := int16(binary.LittleEndian.Uint16(data[6:8]))
	tagsSize := int16(binary.LittleEndian.Uint16(data[8:10]))

	f.Tags = []Tag{}

	start := int16(10)
	end := start + idSize
	f.Id = string(data[start:end])

	start = end
	end = start + nsSize
	f.Namespace = string(data[start:end])

	f.Reference = data[end]

	start = end + 1
	end = start + int16(crcSize)
	f.CRC = string(data[start:end])

	start = end
	end = start + plSize

	f.Payload = string(data[start:end])


	start = end
	end = start + tagsSize
	t := new(Tags)
	err := t.Decode(data[start:end])

	if err != nil{
		return err
	}
	f.Tags = t.Tags
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
	t := crc32.MakeTable(POLYNOMIAL)
	crc := fmt.Sprint(crc32.Checksum([]byte(o.Data), t))
	file := &File{
		Namespace: nsid,
		Id:        o.Id,
		CRC: crc,
		Reference: 1,
		Payload: o.Data,
		Tags:     o.Tags,
	}
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
