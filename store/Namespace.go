package main

import (
	"gopkg.in/validator.v2"
)

type Namespace struct {
	NamespaceCreate
	SpaceAvailable float64 `json:"spaceAvailable,omitempty"`
	SpaceUsed      float64 `json:"spaceUsed,omitempty"`
}

func (s Namespace) Validate() error {

	return validator.Validate(s)
}
