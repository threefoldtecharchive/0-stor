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
