package models

import (
	"encoding/binary"
	"fmt"
	"math"
	"time"

	"github.com/zero-os/0-stor/store/db"
	"github.com/zero-os/0-stor/store/utils"
	validator "gopkg.in/validator.v2"
)

const (
	NAMESPACE_PREFIX       = "2@_"
	NAMESPACE_STATS_PREFIX = "0@stats_"
)

var _ (db.Model) = (*Namespace)(nil)

type Namespace struct {
	NamespaceCreate
	SpaceAvailable float64 `json:"spaceAvailable,omitempty"`
	SpaceUsed      float64 `json:"spaceUsed,omitempty"`
}

func (s Namespace) Validate() error {
	return validator.Validate(s)
}

func (s Namespace) Encode() ([]byte, error) {
	nsc, err := s.NamespaceCreate.Encode()
	if err != nil {
		return nil, err
	}
	b := make([]byte, len(nsc)+16)

	copy(b[0:8], utils.Float64bytes(s.SpaceAvailable))
	copy(b[8:16], utils.Float64bytes(s.SpaceUsed))
	copy(b[16:], nsc)

	return b, nil
}

func (s *Namespace) Decode(data []byte) error {
	s.SpaceAvailable = utils.Float64frombytes(data[0:8])
	s.SpaceUsed = utils.Float64frombytes(data[8:16])

	nsc := NamespaceCreate{}
	nsc.Decode(data[16:])

	s.NamespaceCreate = nsc
	return nil
}

func (s *Namespace) Key() string {
	return fmt.Sprintf("%s%s", NAMESPACE_PREFIX, s.Label)
}

var _ (db.Model) = (*NamespaceCreate)(nil)

// NamespaceCreate is the object sent from the user to create a namespace
// label size   | ACL[] length |Label   |ACL[0]  SizeAvailable | ACL[0] |
// |-------------|--------------|--------|-------------|--------|
// | 2 bytes     | 2 bytes      |        |    2 bytes |        |
type NamespaceCreate struct {
	Acl   []ACL  `json:"acl"`
	Label string `json:"label" validate:"min=5,max=128,regexp=^[a-zA-Z0-9]+$,nonzero"`
}

func (s NamespaceCreate) Validate() error {
	return validator.Validate(s)
}

func (s NamespaceCreate) Encode() ([]byte, error) {
	label := []byte(s.Label)
	labelSize := len(label)

	size := len(label)

	acls := [][]byte{}

	for _, acl := range s.Acl {
		bytes, err := acl.Encode()
		if err != nil {
			return nil, err
		}
		acls = append(acls, bytes)
		size += len(bytes)
		size += 2 // 2 bytes to old size of each element
	}

	b := make([]byte, size+4)

	binary.LittleEndian.PutUint16(b[0:2], uint16(labelSize))
	binary.LittleEndian.PutUint16(b[2:4], uint16(len(acls)))

	start := 4
	end := 4 + labelSize
	copy(b[start:end], label)

	for _, a := range acls {
		aSize := len(a)
		start = end
		end = end + 2
		binary.LittleEndian.PutUint16(b[start:end], uint16(aSize))
		start = end
		end = end + aSize
		copy(b[start:end], a)
	}

	return b, nil
}

func (s *NamespaceCreate) Decode(data []byte) error {
	lSize := int16(binary.LittleEndian.Uint16(data[0:2]))
	aSize := int16(binary.LittleEndian.Uint16(data[2:4]))

	start := int16(4)
	end := 4 + lSize
	s.Label = string(data[start:end])

	for i := 0; i < int(aSize); i++ {
		start = end
		end = end + 2
		size := int16(binary.LittleEndian.Uint16(data[start:end]))
		start = end
		end = end + size
		entry := ACL{}
		entry.Decode(data[start:end])
		s.Acl = append(s.Acl, entry)
	}
	return nil
}

// FIXME
// func (s NamespaceCreate) UpdateACL(db DB, config *Settings, acl ACL) error {
// 	aclIndex := -1 // -1 means ACL for that user does not exist
//
// 	// Find if ACL for that user already exists
// 	for i, item := range s.Acl {
// 		if item.Id == acl.Id {
// 			aclIndex = i
// 			break
// 		}
// 	}
//
// 	// Update User ACL
// 	if aclIndex != -1 {
// 		s.Acl[aclIndex] = acl
// 	} else { // Insert new ACL
// 		s.Acl = append(s.Acl, acl)
// 	}
//
// 	return s.Save(db, config)
//
// }

//
// func (s NamespaceCreate) Exists(db DB, config *Settings) (bool, error) {
// 	exists, err := db.Exists(s.Label)
// 	if err != nil {
// 		log.Errorln(err.Error())
// 		return exists, err
// 	}
// 	return exists, nil
// }
//
// func (s NamespaceCreate) Save(db DB, config *Settings) error {
// 	return db.Set(s.Label, s.Encode())
// }

func (s *NamespaceCreate) Key() string {
	return fmt.Sprintf("%s%s", NAMESPACE_PREFIX, s.Label)
}

// func (s *NamespaceCreate) Get(db DB, config *Settings) (*NamespaceCreate, error) {
// 	key := s.GetKeyForNamespace(s.Label, config)
// 	v, err := db.Get(key)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	if len(v) == 0 {
// 		return nil, nil
// 	}
// 	s.Decode(v)
// 	return s, nil
// // }
//
// func (s *NamespaceCreate) GetStats(db DB, config *Settings) (*NamespaceStats, error) {
// 	stats := NamespaceStats{
// 		Namespace: s.Label,
// 	}
// 	return stats.Get(db, config)
// }
//
// func (s *NamespaceCreate) GetKeyForReservations(config *Settings) string {
// 	r := Reservation{
// 		Namespace: s.Label,
// 	}
// 	return r.GetKey(config)
// }

type NamespaceStat struct {
	NrObjects      int64 `json:"NrObjects" validate:"nonzero"`
	RequestPerHour int64 `json:"requestPerHour" validate:"nonzero"`
}

func (s NamespaceStat) Validate() error {
	return validator.Validate(s)
}

type NamespaceStats struct {
	NamespaceStat
	Namespace         string
	NrRequests        int64
	TotalSizeReserved float64
	Created           time.Time
}

func NewNamespaceStats(namespace string) *NamespaceStats {
	return &NamespaceStats{
		Namespace:  namespace,
		Created:    time.Now(),
		NrRequests: 0,
		NamespaceStat: NamespaceStat{
			NrObjects: 0,
		},
	}
}

func (s *NamespaceStats) Encode() ([]byte, error) {
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

	copy(result[16:24], utils.Float64bytes(s.TotalSizeReserved))

	copy(result[24:], created)
	return result, nil
}

func (s *NamespaceStats) Decode(data []byte) error {
	s.NrObjects = int64(binary.LittleEndian.Uint64(data[0:8]))
	s.NrRequests = int64(binary.LittleEndian.Uint64(data[8:16]))
	s.TotalSizeReserved = utils.Float64frombytes(data[16:24])
	cTime, err := time.Parse(time.RFC3339, string(data[24:]))

	if err != nil {
		return err
	}

	s.Created = cTime
	s.RequestPerHour = int64(float64(s.NrRequests) / math.Ceil(time.Since(cTime).Hours()))
	return nil
}

func (s NamespaceStats) Key() string {
	return fmt.Sprintf("%s%s", NAMESPACE_STATS_PREFIX, s.Namespace)
}

// func (s NamespaceStats) Save(db DB, config *Settings) error {
// 	key := s.GetKeyForNameSpace(config)
// 	return db.Set(key, s.Encode())
// }
//
// func (s NamespaceStats) Delete(db DB, config *Settings) error {
// 	key := s.GetKeyForNameSpace(config)
// 	return db.Delete(key)
// }
//
// func (s *NamespaceStats) Get(db DB, config *Settings) (*NamespaceStats, error) {
// 	key := s.GetKeyForNameSpace(config)
// 	v, err := db.Get(key)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	if v == nil {
// 		return nil, errors.New("Namespace stats not found")
// 	}
//
// 	s.Decode(v)
// 	return s, nil
// }
