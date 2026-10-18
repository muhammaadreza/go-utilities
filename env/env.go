package env

import (
	"errors"
	"os"
	"slices"
	"strings"
	"time"
)

// Defines the exact scalar types supported by the parsers.
// Aliases of these types are supported; newly defined types are not.
type types interface {
	string | int | int8 | int16 | int32 | int64 | uint | uint8 | uint16 |
	uint32 | uint64 | float32 | float64 | bool | time.Time | time.Duration
}

// "Required" reads the environment variable identified by key and parses it as T.
//
// String values are preserved. Other scalar values are trimmed before parsing.
// An unset variable returns ErrMissingValue; an empty value, including a
// whitespace-only non-string value, returns ErrEmptyValue.
// On any failure, Required returns the zero value of T and an *Error.
func Required[T types](key string) (T, error) {
	raw, err := lookup(key)
	if err != nil {
		return zero[T](), newError[T](key, err)
	}

	value, err := parse[T](raw)
	if err != nil {
		return zero[T](), newError[T](key, err)
	}

	return value, nil
}

// "Optional" reads an environment variable as T, using fallback when the variable
// is unset or its raw value is exactly empty. The fallback is returned as supplied
// and is not validated.
//
// String values are preserved; other scalar values are trimmed before parsing.
// Invalid keys, parsing failures, and whitespace-only non-string values return
// the zero value of T and an *Error instead of using fallback.
func Optional[T types](key string, fallback T) (T, error) {
	raw, err := lookup(key)
	if err != nil {
		if errors.Is(err, ErrMissingValue) || errors.Is(err, ErrEmptyValue) {
			return fallback, nil
		}

		return zero[T](), newError[T](key, err)
	}

	value, err := parse[T](raw)
	if err != nil {
		return zero[T](), newError[T](key, err)
	}

	return value, nil
}

// "RequiredSlice" reads an environment variable as a slice of T using separator.
//
// The separator must be non-empty and is checked before the environment lookup.
// Splitting is literal, without CSV quoting or escaping. Every element is trimmed,
// including strings, and empty elements are rejected.
// On any failure, RequiredSlice returns nil and an *Error, never a partial slice.
func RequiredSlice[T types](key, separator string) ([]T, error) {
	if separator == "" {
		return nil, newError[[]T](key, ErrEmptySeparator)
	}

	raw, err := lookup(key)
	if err != nil {
		return nil, newError[[]T](key, err)
	}

	return parseSlice[T](key, raw, separator)
}

// "OptionalSlice" reads an environment variable as a slice of T using the same
// separator and element rules as RequiredSlice.
//
// If the variable is unset or its raw value is exactly empty, it returns a shallow
// copy of fallback without validating its elements. A nil fallback remains nil.
// The separator is checked first. Invalid keys, invalid separators, and element
// parsing failures return nil and an *Error instead of using fallback.
func OptionalSlice[T types](key, separator string, fallback []T) ([]T, error) {
	if separator == "" {
		return nil, newError[[]T](key, ErrEmptySeparator)
	}

	raw, err := lookup(key)
	if err != nil {
		if errors.Is(err, ErrMissingValue) || errors.Is(err, ErrEmptyValue) {
			return slices.Clone(fallback), nil
		}

		return nil, newError[[]T](key, err)
	}

	return parseSlice[T](key, raw, separator)
}

// "lookup" validates key and retrieves its raw environment value without trimming.
//
// It returns ErrInvalidKey for an empty key or a key containing '=' or NUL,
// ErrMissingValue for an unset variable, and ErrEmptyValue for an exactly empty
// value. On failure, the returned string is empty.
// The errors are returned directly, without wrapping.
func lookup(key string) (string, error) {
	if key == "" || strings.ContainsAny(key, "=\x00") {
		return "", ErrInvalidKey
	}

	raw, exists := os.LookupEnv(key)
	if !exists {
		return "", ErrMissingValue
	}
	if raw == "" {
		return "", ErrEmptyValue
	}

	return raw, nil
}
