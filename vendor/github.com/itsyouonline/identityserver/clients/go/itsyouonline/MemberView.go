package itsyouonline

import (
	"gopkg.in/validator.v2"
)

type MemberView struct {
	Useridentifier string `json:"useridentifier" validate:"nonzero"`
	Username       string `json:"username" validate:"nonzero"`
}

func (s MemberView) Validate() error {

	return validator.Validate(s)
}
