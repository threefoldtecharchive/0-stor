package models

import (
	"github.com/zero-os/0-stor/store/core/librairies/reservation"
	"gopkg.in/validator.v2"
)

type NamespacesNsidReservationPostRespBody struct {
	DataAccessToken  string                  `json:"dataAccessToken" validate:"nonzero"`
	Reservation      reservation.Reservation `json:"reservation" validate:"nonzero"`
	ReservationToken string                  `json:"reservationToken" validate:"nonzero"`
}

func (s NamespacesNsidReservationPostRespBody) Validate() error {
	return validator.Validate(s)
}

type Tag struct {
	Key   string `json:"key" validate:"regexp=^\w+$,nonzero"`
	Value string `json:"value" validate:"nonzero"`
}

func (s Tag) Validate() error {
	return validator.Validate(s)
}
