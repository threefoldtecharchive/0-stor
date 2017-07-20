package db

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io/ioutil"
)

const POLYNOMIAL = 0xD5828281

// object is the data structure used to encode, decode object on the disk
type Object struct {
	ReferenceList [160][16]byte
	CRC           uint32
	Data          []byte
}

func NewObjet() *Object {
	return &Object{
		Data: make([]byte, 0, 1024),
	}
}

func (o *Object) Encode() ([]byte, error) {
	o.CRC = crc32.Checksum([]byte(o.Data), crc32.MakeTable(POLYNOMIAL))

	var err error
	buf := &bytes.Buffer{}

	for i := range o.ReferenceList {
		err = binary.Write(buf, binary.LittleEndian, o.ReferenceList[i][:])
		if err != nil {
			return nil, err
		}
	}

	err = binary.Write(buf, binary.LittleEndian, o.CRC)
	if err != nil {
		return nil, err
	}

	err = binary.Write(buf, binary.LittleEndian, o.Data)

	return buf.Bytes(), err
}

func (o *Object) Decode(b []byte) error {
	if o == nil {
		o = NewObjet()
	}

	var err error
	r := bytes.NewReader(b)

	refBuf := make([]byte, 16)
	for i := range o.ReferenceList {
		err = binary.Read(r, binary.LittleEndian, refBuf)
		if err != nil {
			return err
		}
		n := copy(o.ReferenceList[i][:], refBuf)
		if n != 16 {
			return fmt.Errorf("error decoding reference list")
		}
	}

	err = binary.Read(r, binary.LittleEndian, &o.CRC)
	if err != nil {
		return err
	}

	// read the rest of the data from the read
	o.Data, err = ioutil.ReadAll(r)
	return err
}
