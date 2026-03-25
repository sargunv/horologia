package apierrors

import "errors"

type badRequestError struct {
	message string
}

func (e *badRequestError) Error() string {
	return e.message
}

// BadRequest returns an error that signals a 400 Bad Request response.
func BadRequest(msg string) error {
	return &badRequestError{message: msg}
}

// IsBadRequest reports whether err (or any error in its chain) is a bad request error.
func IsBadRequest(err error) bool {
	var bre *badRequestError
	return errors.As(err, &bre)
}

type forbiddenError struct {
	message string
}

func (e *forbiddenError) Error() string {
	return e.message
}

// Forbidden returns an error that signals a 403 Forbidden response.
func Forbidden(msg string) error {
	return &forbiddenError{message: msg}
}

// IsForbidden reports whether err (or any error in its chain) is a forbidden error.
func IsForbidden(err error) bool {
	var fe *forbiddenError
	return errors.As(err, &fe)
}
