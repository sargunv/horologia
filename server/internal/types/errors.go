package types

import "errors"

type validationError struct {
	message string
}

func (e *validationError) Error() string {
	return e.message
}

// ValidationError returns an error indicating invalid input.
func ValidationError(msg string) error {
	return &validationError{message: msg}
}

// IsValidationError reports whether err (or any error in its chain) is a validation error.
func IsValidationError(err error) bool {
	var ve *validationError
	return errors.As(err, &ve)
}

type forbiddenError struct {
	message string
}

func (e *forbiddenError) Error() string {
	return e.message
}

// ForbiddenError returns an error indicating the action is not allowed.
func ForbiddenError(msg string) error {
	return &forbiddenError{message: msg}
}

// IsForbiddenError reports whether err (or any error in its chain) is a forbidden error.
func IsForbiddenError(err error) bool {
	var fe *forbiddenError
	return errors.As(err, &fe)
}
