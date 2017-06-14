package main

import (
	"gopkg.in/validator.v2"
	"encoding/binary"
	"time"
)

type NamespaceStat struct {
	NrObjects      int64 `json:"NrObjects" validate:"nonzero"`
	RequestPerHour int64 `json:"requestPerHour" validate:"nonzero"`
}

func (s NamespaceStat) Validate() error {

	return validator.Validate(s)
}

type Stat struct{
	NamespaceStat
	NrRequests int64
	creationDate time.Time
}


func (s *Stat) toBytes() []byte{
	t := []byte(s.creationDate.Format(time.RFC3339))
	result := make([]byte, 16 + len(t))
	binary.LittleEndian.PutUint64(result[0:8], uint64(s.NrObjects))
	binary.LittleEndian.PutUint64(result[8:16], uint64(s.NrRequests))
	copy(result[16:], t)
	return result
}

func (s *Stat) fromBytes(data []byte) error{
	s.NrObjects = int64(binary.LittleEndian.Uint64(data[0:8]))
	s.NrRequests = int64(binary.LittleEndian.Uint64(data[8:16]))
	d := string(data[16:])

	t, err := time.Parse(time.RFC3339, d)

	if err != nil{
		return err
	}

	s.creationDate = t
	s.RequestPerHour = int64(float64(s.NrRequests) / time.Since(t).Hours())
	return nil
}
