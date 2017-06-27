package main

import (
	"gopkg.in/validator.v2"
)

type StoreStatRequest struct{
	SizeAvailable float64 `json:"size_available" validate:"min=1,nonzero"`
}

type StoreStat struct {
	StoreStatRequest
	SizeUsed float64 `json:"size_used" validate:"min=1,nonzero"`
}

func (s StoreStatRequest) Validate() error {
	return validator.Validate(s)
}

func (s *StoreStat) ToBytes() []byte{
	bytes := make([]byte, 16)
	copy(bytes[0:8], Float64bytes(s.SizeAvailable))
	copy(bytes[8:16], Float64bytes(s.SizeUsed))
	return bytes
}

func (s *StoreStat) FromBytes(data []byte) error{
	s.SizeAvailable = Float64frombytes(data[0:8])
	s.SizeUsed = Float64frombytes(data[8:16])
	return nil
}

func (s StoreStat) Save(db *Badger, config *settings) error{
	key := config.Stats.Store.Collection
	return db.Set(key, s.ToBytes())
}

func (s StoreStat) Exists(db *Badger, config *settings) (bool, error){
	return db.Exists(config.Stats.Store.Collection)
}

func (s *StoreStat) Get(db *Badger, config *settings) error{
	key := config.Stats.Store.Collection
	v, err :=  db.Get(key)
	if err != nil{
		return err
	}
	s.FromBytes(v)
	return nil
}