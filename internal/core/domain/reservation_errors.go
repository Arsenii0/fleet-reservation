package domain

import "errors"

var (
	ErrReservationCannotBeReleased = errors.New("reservation cannot be released")
	ErrReservationClosed           = errors.New("reservation is closed or failed")
	ErrReservationNotFound         = errors.New("reservation not found")
)
