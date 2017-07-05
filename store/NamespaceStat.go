package main

import (
	"gopkg.in/validator.v2"
	"encoding/binary"
	"time"
	"math"
	"fmt"
	"github.com/pkg/errors"
	"strings"
)

type NamespaceStat struct {
	NrObjects      int64 `json:"NrObjects" validate:"nonzero"`
	RequestPerHour int64 `json:"requestPerHour" validate:"nonzero"`
}

func (s NamespaceStat) Validate() error {
	return validator.Validate(s)
}


type NamespaceStats struct{
	NamespaceStat
	Namespace         string
	NrRequests        int64
	TotalSizeReserved float64
	Created           time.Time
}

func NewNamespaceStats(namespace string) *NamespaceStats{
	return &NamespaceStats{
		Namespace: namespace,
		Created: time.Now(),
		NrRequests: 0,
		NamespaceStat : NamespaceStat{
			NrObjects: 0,
		},
	}
}

func (s *NamespaceStats) ToBytes() []byte{
	/*
	------------------------------------
	NrObjects|NrRwquests|SizeUSed|Created
	   8     |   8      | 8bytes
	-------------------------------------

	*/

	created := []byte(time.Time(s.Created).Format(time.RFC3339))
	result := make([]byte, 24+len(created))
	binary.LittleEndian.PutUint64(result[0:8], uint64(s.NrObjects))
	binary.LittleEndian.PutUint64(result[8:16], uint64(s.NrRequests))

	copy(result[16:24], Float64bytes(s.TotalSizeReserved))

	copy(result[24:], created)
	return result
}

func (s *NamespaceStats) FromBytes(data []byte) error{
	s.NrObjects = int64(binary.LittleEndian.Uint64(data[0:8]))
	s.NrRequests = int64(binary.LittleEndian.Uint64(data[8:16]))
	s.TotalSizeReserved = Float64frombytes(data[16:24])
	cTime, err := time.Parse(time.RFC3339, string(data[24:]))

	if err != nil{
		return err
	}

	s.Created = cTime
	s.RequestPerHour = int64(float64(s.NrRequests) / math.Ceil(time.Since(cTime).Hours()))
	return nil
}

func (s NamespaceStats) GetKeyForNameSpace(config *settings) string{
	label := s.Namespace
	if strings.Index(s.Namespace, config.Namespace.prefix) != -1{
		label = strings.Replace(s.Namespace, config.Namespace.prefix, "", 1)
	}
	return fmt.Sprintf("%s%s", config.Stats.Namespaces.Prefix, label)
}

func (s NamespaceStats) Save(db *Badger, config *settings) error{
	key := s.GetKeyForNameSpace(config)
	return db.Set(key, s.ToBytes())
}

func (s NamespaceStats) Delete(db *Badger, config *settings) error{
	key := s.GetKeyForNameSpace(config)
	return db.Delete(key)
}

func (s *NamespaceStats) Get(db *Badger, config *settings) (*NamespaceStats, error){
	key := s.GetKeyForNameSpace(config)
	v, err := db.Get(key)
	if err != nil{
		return nil, err
	}

	if v == nil{
		return nil, errors.New("Namespace stats not found")
	}

	s.FromBytes(v)
	return s, nil
}
