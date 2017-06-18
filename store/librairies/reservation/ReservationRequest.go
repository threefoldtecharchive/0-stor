package reservation

import (
	"gopkg.in/validator.v2"
	"errors"
)

type ReservationRequest struct {
	Id     string `json:"id" validate:"regexp=^[a-zA-Z0-9]+$,nonzero"`
	Period int64 `json:"period" validate:"min=1,nonzero"`
	Size   int64 `json:"size" validate:"min=1,nonzero"`
}

func (s ReservationRequest) Validate(availableFreeSpace int64) error {

	if err := validator.Validate(s); err != nil {
		return err
	}

	if s.Size > availableFreeSpace{
		return errors.New("Data size exceeds limits")
	}

	return nil
}
