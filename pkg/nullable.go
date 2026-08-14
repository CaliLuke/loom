package loom

import (
	"bytes"
	"encoding/json"
)

// Nullable represents a JSON value that can be present with either a concrete
// value or null. Its zero value represents an absent property.
type Nullable[T any] struct {
	value   T
	present bool
	null    bool
}

// NullableValue returns a present Nullable containing value.
func NullableValue[T any](value T) Nullable[T] {
	return Nullable[T]{value: value, present: true}
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
	return n.value, n.present && !n.null
}

// IsZero reports whether the value is absent. It supports the encoding/json
// omitzero field option used by imported optional nullable properties.
func (n Nullable[T]) IsZero() bool {
	return !n.present
}

// MarshalJSON implements json.Marshaler.
func (n Nullable[T]) MarshalJSON() ([]byte, error) {
	if !n.present || n.null {
		return []byte("null"), nil
	}
	return json.Marshal(n.value)
}

// UnmarshalJSON implements json.Unmarshaler and records the distinction
// between an absent property and an explicitly null property.
func (n *Nullable[T]) UnmarshalJSON(data []byte) error {
	n.present = true
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		var zero T
		n.value = zero
		n.null = true
		return nil
	}
	if err := json.Unmarshal(data, &n.value); err != nil {
		return err
	}
	n.null = false
	return nil
}
