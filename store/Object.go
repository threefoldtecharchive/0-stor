package main

import (
	"gopkg.in/validator.v2"
)

type Object struct {
	Data string `json:"data" validate:"nonzero"`
	Id   string `json:"id" validate:"min=5,max=128,regexp=^\w+$,nonzero"`
	Tags []Tag  `json:"tags,omitempty"`
}

func (s Object) Validate() error {

	return validator.Validate(s)
}
