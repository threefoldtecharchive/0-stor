package main

import (
	"gopkg.in/validator.v2"
	"encoding/binary"
)

type StoreStat struct {
	Size int64 `json:"size" validate:"min=1,nonzero"`
}

func (s StoreStat) Validate() error {

	return validator.Validate(s)
}

func (s *StoreStat) toBytes() []byte{
	result := make([]byte, 8)
	binary.LittleEndian.PutUint64(result[0:8], uint64(s.Size))
	return result
}

func (s *StoreStat) fromBytes(data []byte) error{
	s.Size = int64(binary.LittleEndian.Uint64(data[0:8]))
	return nil
}
