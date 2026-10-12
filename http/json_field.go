package http

import (
	"encoding/json/v2"
	"fmt"
	"net/url"
	"reflect"

	loom "github.com/CaliLuke/loom/pkg"
)

// EncodeJSONField writes value as one deterministically encoded JSON field.
// Nil values and empty outer maps retain the existing URL omission behavior.
// JSON tags and JSON marshalers govern the contents, independently of form tags.
func EncodeJSONField(values url.Values, name string, value any) (bool, error) {
	v := reflect.ValueOf(value)
	for v.IsValid() && (v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface) {
		if v.IsNil() {
			return false, nil
		}
		v = v.Elem()
	}
	if !v.IsValid() || v.Kind() == reflect.Map && v.Len() == 0 {
		return false, nil
	}
	data, err := json.Marshal(value, loom.JSONOptions(), json.Deterministic(true))
	if err != nil {
		return false, fmt.Errorf("encode field %q: %w", name, err)
	}
	values.Add(name, string(data))
	return true, nil
}

// DecodeJSONField decodes the first value of the named field into target using
// the framework JSON codec. Absent fields leave target unchanged. Malformed or
// trailing JSON or top-level null returns an error. URL fields do not support
// nullable attributes; bracketed keys are not an alternate codec.
func DecodeJSONField(values url.Values, name string, target any) (bool, error) {
	raw, exists := values[name]
	if !exists {
		return false, nil
	}
	data := ""
	if len(raw) > 0 {
		data = raw[0]
	}
	if err := decodeNonNullableJSON([]byte(data), target); err != nil {
		return true, fmt.Errorf("decode field %q: %w", name, err)
	}
	return true, nil
}
