package main

import (
	"gopkg.in/validator.v2"
)

// Mapping between a user ID or group ID and an ACLEntry
type ACL struct {
	Acl ACLEntry `json:"acl" validate:"nonzero"`
	Id  string   `json:"id" validate:"regexp=^\w+$,nonzero"`
}

func (s ACL) Validate() error {

	return validator.Validate(s)
}

/*

| ACLEntry  | Id      |
|-----------|---------|
| 4 bytes   | variable|

 */
func (s ACL) ToBytes() []byte{
	Id := []byte(s.Id)
	b := make([]byte, len(Id) + 4)
	copy(b[0:4], s.Acl.ToBytes())
	copy(b[4:], Id)
	return b
}

func (s *ACL) FromBytes(data []byte){
	s.Acl = ACLEntry{}
	s.Acl.FromBytes(data[0:4])
	s.Id = string(data[4:])
}
