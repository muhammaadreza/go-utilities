package env

import (
	"errors"
	"math"
	"strconv"
	"strings"
	"time"
)

func zero[T any]() T {
	var value T
	return value
}

// "parse" converts raw to T without reading the environment.
//
// Strings are returned unchanged, including empty strings. Other values are
// trimmed before parsing. Integers use base 10, booleans follow strconv.ParseBool,
// timestamps use time.RFC3339, and durations follow time.ParseDuration.
// Floating-point results must be finite.
//
// On failure, parse returns the zero value of T. Empty normalized values produce
// ErrEmptyValue, integer or floating-point overflow produces ErrInvalidRange,
// and other parsing failures produce ErrInvalidValue. ErrInvalidType is reserved
// for unsupported types.
func parse[T types](raw string) (T, error) {
	var value T
	if _, ok := any(value).(string); ok {
		return any(raw).(T), nil
	}

	raw = strings.TrimSpace(raw)
	if raw == "" {
		return value, ErrEmptyValue
	}

	var err error
	switch any(value).(type) {
	case int:
		var number int
		number, err = strconv.Atoi(raw)
		value = any(number).(T)
	case int8:
		var number int64
		number, err = strconv.ParseInt(raw, 10, 8)
		value = any(int8(number)).(T)
	case int16:
		var number int64
		number, err = strconv.ParseInt(raw, 10, 16)
		value = any(int16(number)).(T)
	case int32:
		var number int64
		number, err = strconv.ParseInt(raw, 10, 32)
		value = any(int32(number)).(T)
	case int64:
		var number int64
		number, err = strconv.ParseInt(raw, 10, 64)
		value = any(number).(T)
	case uint:
		var number uint64
		number, err = strconv.ParseUint(raw, 10, strconv.IntSize)
		value = any(uint(number)).(T)
	case uint8:
		var number uint64
		number, err = strconv.ParseUint(raw, 10, 8)
		value = any(uint8(number)).(T)
	case uint16:
		var number uint64
		number, err = strconv.ParseUint(raw, 10, 16)
		value = any(uint16(number)).(T)
	case uint32:
		var number uint64
		number, err = strconv.ParseUint(raw, 10, 32)
		value = any(uint32(number)).(T)
	case uint64:
		var number uint64
		number, err = strconv.ParseUint(raw, 10, 64)
		value = any(number).(T)
	case float32:
		var number float64
		number, err = parseFloat(raw, 32)
		value = any(float32(number)).(T)
	case float64:
		var number float64
		number, err = parseFloat(raw, 64)
		value = any(number).(T)
	case bool:
		var boolean bool
		boolean, err = strconv.ParseBool(raw)
		value = any(boolean).(T)
	case time.Time:
		var timestamp time.Time
		timestamp, err = time.Parse(time.RFC3339, raw)
		value = any(timestamp).(T)
	case time.Duration:
		var duration time.Duration
		duration, err = time.ParseDuration(raw)
		value = any(duration).(T)
	default:
		return value, ErrInvalidType
	}

	if err != nil {
		if errors.Is(err, strconv.ErrRange) {
			return zero[T](), ErrInvalidRange
		}
		return zero[T](), ErrInvalidValue
	}

	return value, nil
}

// "parseFloat" parses raw with the requested 32- or 64-bit precision and rejects
// NaN and infinities with ErrInvalidValue.
//
// Parsing errors from strconv.ParseFloat are returned unchanged so the caller
// can distinguish syntax errors from overflow. On any failure, the value is zero.
// Whitespace normalization is handled by parse before calling this helper.
func parseFloat(raw string, bits int) (float64, error) {
	value, err := strconv.ParseFloat(raw, bits)
	if err != nil {
		return 0, err
	}

	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, ErrInvalidValue
	}

	return value, nil
}

// "parseSlice" splits raw using a non-empty literal separator and parses each
// trimmed element as T. The caller must validate separator before calling it.
//
// Empty elements and parsing failures return nil and an *Error containing key
// and the expected element type. No partial result is returned.
// The result is preallocated, and splitting avoids an intermediate string slice.
func parseSlice[T types](key, raw, separator string) ([]T, error) {
	result := make([]T, strings.Count(raw, separator)+1)
	for i := range result {
		var part string
		part, raw, _ = strings.Cut(raw, separator)
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, newError[T](key, ErrEmptyValue)
		}

		value, err := parse[T](part)
		if err != nil {
			return nil, newError[T](key, err)
		}

		result[i] = value
	}

	return result, nil
}
