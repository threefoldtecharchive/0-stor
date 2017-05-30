package main

import (
	"gopkg.in/validator.v2"
)

type ObjectCreate struct {
}

func (s ObjectCreate) Validate() error {

	return validator.Validate(s)
}
