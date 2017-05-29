package main

import (
	"gopkg.in/validator.v2"
)

type ObjectUpdate struct {
	Data string `json:"data" validate:"nonzero"`
	Tags []Tag  `json:"tags,omitempty"`
}

func (s ObjectUpdate) Validate() error {

	return validator.Validate(s)
}
