package main

import (
	"gopkg.in/validator.v2"
)

type StoreStat struct {
	SizeAvailable float64 `json:"size_available" validate:"min=1,nonzero"`
}

func (s StoreStat) Validate() error {

	return validator.Validate(s)
}

func (s *StoreStat) ToBytes() []byte{
	return Float64bytes(s.SizeAvailable)
}

func (s *StoreStat) FromBytes(data []byte) error{
	s.SizeAvailable = Float64frombytes(data[0:8])
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