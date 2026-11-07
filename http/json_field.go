package http

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"net/url"
	"reflect"

	loom "github.com/CaliLuke/loom/pkg"
)

// FieldApplication identifies the value encoded by a named content field.
type FieldApplication uint8

const (
	// FieldValue encodes one complete value in one field occurrence.
	FieldValue FieldApplication = iota
	// FieldItems encodes each top-level array item as a separate field occurrence.
	FieldItems
)

// EncodeJSONField writes deterministic JSON using the captured application.
// FieldValue writes one value; FieldItems requires an array and writes one JSON
// document per item. Empty arrays produce no occurrences in item mode. Nil
// pointers, interfaces and collections are absent. Nonnil empty collections
// are supplied JSON values, including values with custom marshalers. JSON tags and
// marshalers govern the values independently of form tags. Errors publish no
// field occurrences, including when a custom marshaler returns a non-array.
func EncodeJSONField(values url.Values, name string, value any, application FieldApplication) (bool, error) {
	if err := validateFieldApplication(application); err != nil {
		return false, err
	}
	v := reflect.ValueOf(value)
	for v.IsValid() && (v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface) {
		if v.IsNil() {
			return false, nil
		}
		v = v.Elem()
	}
	if !v.IsValid() || (v.Kind() == reflect.Map || v.Kind() == reflect.Slice) && v.IsNil() {
		return false, nil
	}
	data, err := json.Marshal(value, loom.JSONOptions(), json.Deterministic(true))
	if err != nil {
		return false, fmt.Errorf("encode field %q: %w", name, err)
	}
	if application == FieldValue {
		values.Add(name, string(data))
		return true, nil
	}
	if jsontext.Value(data).Kind() != '[' {
		return false, fmt.Errorf("encode field %q: item application requires an array", name)
	}
	var items []jsontext.Value
	if err := json.Unmarshal(data, &items); err != nil {
		return false, fmt.Errorf("encode field %q: %w", name, err)
	}
	for _, item := range items {
		values.Add(name, string(item))
	}
	return len(items) > 0, nil
}

// DecodeJSONField decodes named content using the framework JSON codec.
// FieldValue uses the first occurrence. FieldItems validates every occurrence
// as JSON and assembles one array before typed decoding; nested arrays remain
// single items. Absence leaves target unchanged. allowNull governs the complete
// value, while item null admission belongs to the target's element schema.
func DecodeJSONField(values url.Values, name string, target any, allowNull bool, application FieldApplication) (bool, error) {
	if err := validateFieldApplication(application); err != nil {
		return false, err
	}
	raw, exists := values[name]
	if !exists {
		return false, nil
	}
	data, err := jsonFieldValue(raw, application)
	if err != nil {
		return true, fmt.Errorf("decode field %q: %w", name, err)
	}
	decode := decodeNonNullableJSON
	if allowNull {
		decode = decodeJSON
	}
	if err := decode(data, target); err != nil {
		return true, fmt.Errorf("decode field %q: %w", name, err)
	}
	return true, nil
}

func validateFieldApplication(application FieldApplication) error {
	if application != FieldValue && application != FieldItems {
		return fmt.Errorf("invalid field application %d", application)
	}
	return nil
}

func jsonFieldValue(values []string, application FieldApplication) (jsontext.Value, error) {
	if application == FieldValue {
		raw := jsontext.Value(nil)
		if len(values) > 0 {
			raw = jsontext.Value(values[0])
		}
		return raw, nil
	}
	data := []byte{'['}
	for index, value := range values {
		raw := jsontext.Value(value)
		var checked jsontext.Value
		if err := json.Unmarshal(raw, &checked); err != nil {
			return nil, fmt.Errorf("field item %d: %w", index, err)
		}
		if index > 0 {
			data = append(data, ',')
		}
		data = append(data, raw...)
	}
	return append(data, ']'), nil
}
