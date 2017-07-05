package main

import (
	"gopkg.in/validator.v2"
	"encoding/binary"
	"fmt"
)

type NamespaceCreate struct {
	Acl   []ACL  `json:"acl"`
	Label string `json:"label" validate:"min=5,max=128,regexp=^[a-zA-Z0-9]+$,nonzero"`
}

func (s NamespaceCreate) Validate() error {
	return validator.Validate(s)
}

/*
 label size   | ACL[] length |Label   |ACL[0]  SizeAvailable | ACL[0] |
|-------------|--------------|--------|-------------|--------|
| 2 bytes     | 2 bytes      |        |    2 bytes |        |

 */

func (s NamespaceCreate) ToBytes() []byte{
	label := []byte(s.Label)
	labelSize := len(label)

	size := len(label)

	acls := [][]byte{}

	for _, acl := range s.Acl{
		bytes := acl.ToBytes()
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

	for _, a := range acls{
		aSize := len(a)
		start = end
		end = end + 2
		binary.LittleEndian.PutUint16(b[start:end], uint16(aSize))
		start = end
		end = end + aSize
		copy(b[start:end], a)
	}

	return b
}

func (s *NamespaceCreate) FromBytes(data []byte){
	lSize := int16(binary.LittleEndian.Uint16(data[0:2]))
	aSize := int16(binary.LittleEndian.Uint16(data[2:4]))

	start := int16(4)
	end := 4 + lSize
	s.Label = string(data[start: end])

	for i:= 0; i < int(aSize); i++{
		start = end
		end = end + 2
		size := int16(binary.LittleEndian.Uint16(data[start:end]))
		start = end
		end = end + size
		entry := ACL{}
		entry.FromBytes(data[start:end])
		s.Acl = append(s.Acl, entry)
	}
}

func (s NamespaceCreate) UpdateACL(db *Badger, config *settings, acl ACL) error{
	aclIndex := -1 // -1 means ACL for that user does not exist

	// Find if ACL for that user already exists
	for i, item := range s.Acl {
		if item.Id == acl.Id {
			aclIndex = i
			break
		}
	}

	// Update User ACL
	if aclIndex != -1 {
		s.Acl[aclIndex] = acl
	} else { // Insert new ACL
		s.Acl = append(s.Acl, acl)
	}

	return s.Save(db, config)

}

func (s NamespaceCreate) Exists(db *Badger, config *settings) (bool, error){
	return db.Exists(s.Label)
}

func (s NamespaceCreate) Save(db *Badger, config *settings) error{
	return db.Set(s.Label, s.ToBytes())
}

func (NamespaceCreate) GetKeyForNamespace(label string, config *settings) string{
	return fmt.Sprintf("%s%s", config.Namespace.prefix, label)
}


func (s *NamespaceCreate) Get(db *Badger, config *settings) (*NamespaceCreate, error){
	key := s.GetKeyForNamespace(s.Label, config)
	v, err := db.Get(key)
	if err != nil{
		return nil, err
	}

	if len(v) == 0{
		return nil, nil
	}
	s.FromBytes(v)
	return s, nil
}

func (s *NamespaceCreate) GetStats(db *Badger, config *settings) (*NamespaceStats, error){
	stats := NamespaceStats{
		Namespace:s.Label,
	}
	return stats.Get(db, config)
}

func (s *NamespaceCreate) GetKeyForReservations(config *settings) string{
	r := Reservation{
		Namespace: s.Label,
	}
	return r.GetKey(config)
}


