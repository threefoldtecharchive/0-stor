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

func (s *StoreStat) Encode() []byte {
	bytes := make([]byte, 16)
	copy(bytes[0:8], utils.Float64bytes(s.SizeAvailable))
	copy(bytes[8:16], utils.Float64bytes(s.SizeUsed))
	return bytes
}

func (s *StoreStat) Decode(data []byte) error {
	s.SizeAvailable = utils.Float64frombytes(data[0:8])
	s.SizeUsed = utils.Float64frombytes(data[8:16])
	return nil
}

//
// func (s StoreStat) Save(db DB, config *Settings) error {
// 	key := config.Store.Stats.Collection
// 	return db.Set(key, s.Encode())
// }
//
// func (s StoreStat) Exists(db DB, config *Settings) (bool, error) {
// 	return db.Exists(config.Store.Stats.Collection)
// }
//
// func (s *StoreStat) Get(db DB, config *Settings) error {
// 	key := config.Store.Stats.Collection
// 	v, err := db.Get(key)
// 	if err != nil {
// 		return err
// 	}
// 	s.Decode(v)
// 	return nil
// }
