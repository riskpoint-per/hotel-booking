package model

import (
	"errors"
	"fmt"
)

// Sentinel domain errors. The HTTP layer maps these to status codes, so
// lower layers never need to know about HTTP
var (
	ErrNotFound           = errors.New("resource not found")
	ErrRoomUnavailable    = errors.New("room is not available for the requested dates")
	ErrCapacityExceeded   = errors.New("number of guests exceeds room capacity")
	ErrDuplicateReference = errors.New("booking reference already exists")
)

// ValidationError describes invalid input, field by field
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed: %v", e.Fields)
}

// Add records a problem with a field.
func (e *ValidationError) Add(field, msg string) {
	if e.Fields == nil {
		e.Fields = map[string]string{}
	}
	e.Fields[field] = msg
}

// ErrorOrNil returns the error only if at least one field failed, which lets
// callers write `return v.ErrorOrNil()` without a typed-nil interface pitfall
func (e *ValidationError) ErrorOrNil() error {
	if len(e.Fields) == 0 {
		return nil
	}
	return e
}
