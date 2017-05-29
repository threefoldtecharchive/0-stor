package main

import (
	"gopkg.in/validator.v2"
)

type Tag struct {
	Key   string `json:"key" validate:"regexp=^\w+$,nonzero"`
	Value string `json:"value" validate:"nonzero"`
}

func (s Tag) Validate() error {

	return validator.Validate(s)
}
