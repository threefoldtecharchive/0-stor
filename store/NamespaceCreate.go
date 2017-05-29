package main

import (
	"gopkg.in/validator.v2"
)

type NamespaceCreate struct {
	Acl   []ACL  `json:"acl" validate:"nonzero"`
	Label string `json:"label" validate:"min=5,max=128,regexp=^\w+$,nonzero"`
}

func (s NamespaceCreate) Validate() error {

	return validator.Validate(s)
}
