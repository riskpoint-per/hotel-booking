package repository

import (
	"errors"

	"github.com/lib/pq"

	"github.com/riskpoint-per/hotel-booking/model"
)

// Postgres SQLSTATE codes into domain errors
const (
	codeUniqueViolation    = "23505"
	codeCheckViolation     = "23514"
	codeExclusionViolation = "23P01"
)

// translateBookingError converts constraint violations raised by the
// bookings table into domain errors
func translateBookingError(err error) error {
	var pqErr *pq.Error
	if !errors.As(err, &pqErr) {
		return err
	}
	switch string(pqErr.Code) {
	case codeExclusionViolation:
		return model.ErrRoomUnavailable
	case codeUniqueViolation:
		if pqErr.Constraint == "bookings_reference_key" {
			return model.ErrDuplicateReference
		}
	case codeCheckViolation:
		if pqErr.Constraint == "" {
			return model.ErrCapacityExceeded
		}
	}
	return err
}
