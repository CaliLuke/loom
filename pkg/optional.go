package loom

import (
	"bytes"
	"encoding/json/v2"
	"errors"
)

var (
	// ErrAbsentOptional is returned when an absent Optional is marshaled
	// without an enclosing omitzero field tag.
	ErrAbsentOptional = errors.New("cannot marshal absent Optional")
	// ErrNullOptional is returned when JSON null is decoded into Optional.
	ErrNullOptional = errors.New("Optional does not allow null")
)

// Optional represents a value that may be absent but cannot be null. Its zero
// value represents absence.
type Optional[T any] struct {
	value *T
}

// OptionalValue returns a present Optional containing value.
func OptionalValue[T any](value T) Optional[T] {
	return Optional[T]{value: &value}
}

// Present reports whether the Optional contains a concrete value.
func (o Optional[T]) Present() bool {
	return o.value != nil
}

// Value returns the concrete value and true when it is present.
func (o Optional[T]) Value() (T, bool) {
	if o.value == nil {
		var zero T
		return zero, false
	}
	return *o.value, true
}

// SetValue replaces the Optional state with a present concrete value.
func (o *Optional[T]) SetValue(value T) {
	o.value = &value
}

// IsZero reports whether the Optional is absent.
func (o Optional[T]) IsZero() bool {
	return o.value == nil
}

// MarshalJSON implements json.Marshaler.
func (o Optional[T]) MarshalJSON() ([]byte, error) {
	if o.value == nil {
		return nil, ErrAbsentOptional
	}
	data, err := json.Marshal(o.value, json.Deterministic(true))
	if err != nil {
		return nil, err
	}
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return nil, ErrNullOptional
	}
	return data, nil
}

// UnmarshalJSON implements json.Unmarshaler and rejects JSON null.
func (o *Optional[T]) UnmarshalJSON(data []byte) error {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		o.value = nil
		return ErrNullOptional
	}
	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	o.value = &value
	return nil
}
