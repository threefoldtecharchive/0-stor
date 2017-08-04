package itsyouonline

import (
	"gopkg.in/validator.v2"
)

type JoinOrganizationInvitationView struct {
	Created        DateTime                                 `json:"created,omitempty"`
	Emailaddress   string                                   `json:"emailaddress" validate:"nonzero"`
	Isorganization bool                                     `json:"isorganization" validate:"nonzero"`
	Method         EnumJoinOrganizationInvitationViewMethod `json:"method" validate:"nonzero"`
	Organization   string                                   `json:"organization" validate:"nonzero"`
	Phonenumber    string                                   `json:"phonenumber" validate:"nonzero"`
	Role           EnumJoinOrganizationInvitationViewRole   `json:"role" validate:"nonzero"`
	Status         EnumJoinOrganizationInvitationViewStatus `json:"status" validate:"nonzero"`
	User           MemberView                               `json:"user" validate:"nonzero"`
}

func (s JoinOrganizationInvitationView) Validate() error {

	return validator.Validate(s)
}
