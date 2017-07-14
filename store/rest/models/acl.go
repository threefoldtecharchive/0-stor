package models

import (
	"github.com/zero-os/0-stor/store/db"
	validator "gopkg.in/validator.v2"
)

var _ (db.Model) = (*ACL)(nil)

/*
 ACL of a reservation
 | ACLEntry  | Id      |
 |-----------|---------|
 | 4 bytes   | variable|
*/

type ACL struct {
	Acl ACLEntry `json:"acl" validate:"nonzero"`
	Id  string   `json:"id" validate:"regexp=^\w+$,nonzero"`
}

func (s ACL) Validate() error {
	return validator.Validate(s)
}

func (s ACL) Encode() ([]byte, error) {
	Id := []byte(s.Id)
	b := make([]byte, len(Id)+4)
	aclEncoded, err := s.Acl.Encode()
	if err != nil {
		return nil, err
	}
	copy(b[0:4], aclEncoded)
	copy(b[4:], Id)
	return b, nil
}

func (s *ACL) Decode(data []byte) error {
	s.Acl = ACLEntry{}
	s.Acl.Decode(data[0:4])
	s.Id = string(data[4:])
	return nil
}

func (s *ACL) Key() string {
	return ""
}

/*
 ACLEntry for a reservation
 | Admin  | Read  | Write  | Delete  |
 |--------|-------|--------|---------|
 | 1 byte | 1 byte| 1 byte | 1 byte  |
*/
type ACLEntry struct {
	Admin  bool `json:"admin"`
	Delete bool `json:"delete"`
	Read   bool `json:"read"`
	Write  bool `json:"write"`
}

func (s ACLEntry) Validate() error {
	return validator.Validate(s)
}

func (s ACLEntry) Encode() ([]byte, error) {
	r := []byte{0, 0, 0, 0}

	if s.Admin {
		r[0] = 1
	}

	if s.Read {
		r[1] = 1

	}

	if s.Write {
		r[2] = 1
	}

	if s.Delete {
		r[3] = 1
	}
	return r, nil
}

func (s *ACLEntry) Decode(data []byte) error {
	if data[0] == 1 {
		s.Admin = true
	} else {
		s.Admin = false
	}

	if data[1] == 1 {
		s.Read = true
	} else {
		s.Read = false
	}

	if data[2] == 1 {
		s.Write = true
	} else {
		s.Write = false
	}

	if data[3] == 1 {
		s.Delete = true
	} else {
		s.Delete = false
	}
	return nil
}

func (s ACLEntry) Key() string {
	return ""
}
