package models

import (
	"errors"

	"gopkg.in/validator.v2"
)

type ReservationRequest struct {
	Period int64 `json:"period" validate:"min=1,nonzero"`
	Size   int64 `json:"size" validate:"min=1,nonzero"`
}

func (s ReservationRequest) Validate() error {

	if err := validator.Validate(s); err != nil {
		return err
	}

	return nil
}

func (s ReservationRequest) ValidateFreeSpace(sizeAvailable float64) error {
	if float64(s.Size) > sizeAvailable {
		return errors.New("Requested reservation size exceeds allowed limits")
	}
	return nil
}
