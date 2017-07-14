package models

import (
	"github.com/zero-os/0-stor/store/utils"
	validator "gopkg.in/validator.v2"
)

type StoreStatRequest struct {
	SizeAvailable float64 `json:"size_available" validate:"min=1,nonzero"`
}

type StoreStat struct {
	StoreStatRequest
	SizeUsed float64 `json:"size_used" validate:"min=1,nonzero"`
}

func (s StoreStatRequest) Validate() error {
	return validator.Validate(s)
}

func (s *StoreStat) Encode() ([]byte, error) {
	bytes := make([]byte, 16)
	copy(bytes[0:8], utils.Float64bytes(s.SizeAvailable))
	copy(bytes[8:16], utils.Float64bytes(s.SizeUsed))
	return bytes, nil
}

func (s *StoreStat) Decode(data []byte) error {
	s.SizeAvailable = utils.Float64frombytes(data[0:8])
	s.SizeUsed = utils.Float64frombytes(data[8:16])
	return nil
}

func (s StoreStat) Key() string {
	return STORE_STATS_PREFIX
}
