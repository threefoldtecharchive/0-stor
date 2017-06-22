package main

import (
	"gopkg.in/validator.v2"
)

// ACL entry for a reservation
type ACLEntry struct {
	Admin  bool `json:"admin"`
	Delete bool `json:"delete"`
	Read   bool `json:"read"`
	Write  bool `json:"write"`
}

func (s ACLEntry) Validate() error {

	return validator.Validate(s)
}

/*

| Admin  | Read  | Write  | Delete  |
|--------|-------|--------|---------|
| 1 byte | 1 byte| 1 byte | 1 byte  |

 */
func(s ACLEntry) ToBytes() []byte{
	r := []byte{0, 0, 0, 0}

	if s.Admin{
		r[0] = 1
	}

	if s.Read{
		r[1] = 1

	}

	if s.Write{
		r[2] = 1
	}

	if s.Delete{
		r[3] = 1
	}
	return r
}

func (s *ACLEntry) FromBytes(data []byte){
	if data[0] == 1{
		s.Admin = true
	}else{
		s.Admin = false
	}

	if data[1] == 1{
		s.Read = true
	}else{
		s.Read = false
	}

	if data[2] == 1{
		s.Write = true
	}else{
		s.Write = false
	}

	if data[3] == 1{
		s.Delete = true
	}else{
		s.Delete = false
	}
}


func (s *ACLEntry) SetAdminPrivileges() *ACLEntry{
	s.Admin = true
	s.Delete = true
	s.Read = true
	s.Write = true

	return s
}