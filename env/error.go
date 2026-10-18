package env

import (
	"errors"
	"fmt"
)

var (
	// Environment errors
	ErrInvalidKey     = errors.New("environment variable key is invalid")
	ErrEmptySeparator = errors.New("slice separator cannot be empty")

	ErrMissingValue   = errors.New("environment variable is not set")
	ErrEmptyValue     = errors.New("environment variable is empty")

	ErrInvalidType    = errors.New("unsupported environment variable type")
	ErrInvalidValue   = errors.New("environment variable has an invalid value")
	ErrInvalidRange   = errors.New("environment variable is out of range")
)

type Error struct {
	key      string
	expected string
	err      error
}

func newError[T any](key string, err error) *Error {
	return &Error{
		err:      err,
		key:      key,
		expected: fmt.Sprintf("%T", zero[T]()),
	}
}

func (e *Error) Error() string {
	return fmt.Sprintf(
		"environment variable '%q': %v (expected %s)",
		e.key, e.err, e.expected,
	)
}

func (e *Error) Unwrap() error {
	return e.err
}

func (e *Error) Key() string {
	return e.key
}

func (e *Error) Expected() string {
	return e.expected
}
