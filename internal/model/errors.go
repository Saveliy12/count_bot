package model

import "errors"

var (
	ErrNotFound   = errors.New("not found")
	ErrValidation = errors.New("validation error")
	ErrConflict   = errors.New("conflict")
)

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

func (e *ValidationError) Unwrap() error {
	return ErrValidation
}

func NewValidationError(field, message string) error {
	return &ValidationError{Field: field, Message: message}
}
