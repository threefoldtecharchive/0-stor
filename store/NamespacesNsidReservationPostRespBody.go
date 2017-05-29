package main

import (
	"gopkg.in/validator.v2"
)

type NamespacesNsidReservationPostRespBody struct {
	DataAccessToken  string      `json:"dataAccessToken" validate:"nonzero"`
	Reservation      Reservation `json:"reservation" validate:"nonzero"`
	ReservationToken string      `json:"reservationToken" validate:"nonzero"`
}

func (s NamespacesNsidReservationPostRespBody) Validate() error {

	return validator.Validate(s)
}
