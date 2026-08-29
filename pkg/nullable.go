package loom

import (
	"bytes"
	"encoding/json/v2"
	"errors"
)

var (
	// ErrAbsentNullable is returned when an absent Nullable is marshaled
	// without an enclosing omitzero field tag.
	ErrAbsentNullable = errors.New("cannot marshal absent Nullable")
	// ErrConcreteNullNullable is returned when a concrete Nullable value would
	// encode as JSON null. Use NullValue to represent an intentional null.
	ErrConcreteNullNullable = errors.New("concrete Nullable value encodes as null; use NullValue")
)

// Nullable represents a JSON value that can be present with either a concrete
// value or null. Its zero value represents an absent property.
type Nullable[T any] struct {
	value   *T
	present bool
	null    bool
}

// NullableValue returns a present Nullable containing value.
func NullableValue[T any](value T) Nullable[T] {
	return Nullable[T]{value: &value, present: true}
}

// NullValue returns a present Nullable containing JSON null.
func NullValue[T any]() Nullable[T] {
	return Nullable[T]{present: true, null: true}
}

// Present reports whether the value was present in decoded JSON or was
// explicitly constructed with NullableValue or NullValue.
func (n Nullable[T]) Present() bool {
	return n.present
}

// IsNull reports whether the value is present and explicitly null.
func (n Nullable[T]) IsNull() bool {
	return n.present && n.null
}

// Value returns the concrete value and true when this Nullable is present and
// non-null. It returns the zero value and false otherwise.
func (n Nullable[T]) Value() (T, bool) {
	if n.value == nil || !n.present || n.null {
		var zero T
		return zero, false
	}
	return *n.value, true
}

// SetValue replaces the Nullable state with a present concrete value.
func (n *Nullable[T]) SetValue(value T) {
	n.value = &value
	n.present = true
	n.null = false
}

// SetNull replaces the Nullable state with a present JSON null.
func (n *Nullable[T]) SetNull() {
	n.value = nil
	n.present = true
	n.null = true
}

// IsZero reports whether the value is absent. It supports the encoding/json/v2
// omitzero field option used by imported optional nullable properties.
func (n Nullable[T]) IsZero() bool {
	return !n.present
}

// MarshalJSON implements json.Marshaler.
func (n Nullable[T]) MarshalJSON() ([]byte, error) {
	if !n.present {
		return nil, ErrAbsentNullable
	}
	if n.null {
		return []byte("null"), nil
	}
	data, err := json.Marshal(n.value)
	if err != nil {
		return nil, err
	}
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return nil, ErrConcreteNullNullable
	}
	return data, nil
}

// UnmarshalJSON implements json.Unmarshaler and records the distinction
// between an absent property and an explicitly null property.
func (n *Nullable[T]) UnmarshalJSON(data []byte) error {
	n.present = true
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		n.value = nil
		n.null = true
		return nil
	}
	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	n.value = &value
	n.null = false
	return nil
}
