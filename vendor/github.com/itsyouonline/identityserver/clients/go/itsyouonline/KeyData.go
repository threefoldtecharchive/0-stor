package itsyouonline

import (
	"gopkg.in/validator.v2"
)

type KeyData struct {
	Algorithm string   `json:"algorithm" validate:"nonzero"`
	Comment   string   `json:"comment,omitempty"`
	Timestamp DateTime `json:"timestamp,omitempty"`
}

func (s KeyData) Validate() error {

	return validator.Validate(s)
}
