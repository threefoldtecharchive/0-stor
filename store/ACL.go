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
