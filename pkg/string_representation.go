package loom

import (
	"encoding"
	"fmt"
	"reflect"
)

// StringValue returns a custom String field's semantic text for validation or
// conversion to native String storage.
// An authored fmt.Stringer on the value or its address takes precedence over
// native or defined string storage. Opaque values without that capability and
// nil pointers return an error containing path. JSON codecs are not consulted.
func StringValue[T any](value T, path string) (string, error) {
	physical := reflect.ValueOf(&value).Elem()
	seen := make(map[reflect.Type]struct{})
	for {
		if !physical.IsValid() || (physical.Kind() == reflect.Pointer || physical.Kind() == reflect.Interface) && physical.IsNil() {
			return "", fmt.Errorf("%s: nil custom String representation", path)
		}
		if physical.Kind() == reflect.Interface {
			// Interface values may hide a non-addressable concrete value. Copy
			// that value so its pointer method set is available without writes
			// to the caller's storage.
			concrete := physical.Elem()
			physical = reflect.New(concrete.Type()).Elem()
			physical.Set(concrete)
			continue
		}
		if text, ok := physical.Interface().(fmt.Stringer); ok {
			return text.String(), nil
		}
		if physical.CanAddr() {
			if text, ok := physical.Addr().Interface().(fmt.Stringer); ok {
				return text.String(), nil
			}
		}

		if physical.Kind() == reflect.String {
			return physical.String(), nil
		}
		if physical.Kind() != reflect.Pointer {
			break
		}
		if _, repeated := seen[physical.Type()]; repeated {
			return "", fmt.Errorf("%s: recursive pointer custom String representation %T", path, value)
		}
		seen[physical.Type()] = struct{}{}
		physical = physical.Elem()
	}
	return "", fmt.Errorf("%s: custom String representation %T has no fmt.Stringer", path, value)
}

// ParseStringValue materializes authored text into a custom String field.
// A fresh target's encoding.TextUnmarshaler takes precedence over assignment to
// native or defined string storage. Concrete pointer receivers are allocated as
// needed; recursive pointer types and unsupported opaque representations return
// errors. Failed parsing returns the zero T and wraps the parser error with path.
// JSON codecs are not consulted and callers decide whether absence permits a
// default to be materialized. Supplied wire text uses the same parser.
func ParseStringValue[T any](text, path string) (T, error) {
	var value T
	physical := reflect.ValueOf(&value).Elem()
	seen := make(map[reflect.Type]struct{})
	for {
		if physical.CanAddr() {
			if parser, ok := physical.Addr().Interface().(encoding.TextUnmarshaler); ok {
				if err := parser.UnmarshalText([]byte(text)); err != nil {
					var zero T
					return zero, fmt.Errorf("%s: custom String value: %w", path, err)
				}
				return value, nil
			}
		}
		if physical.Kind() == reflect.String {
			physical.SetString(text)
			return value, nil
		}
		if physical.Kind() != reflect.Pointer {
			var zero T
			return zero, fmt.Errorf("%s: custom String representation %T has no encoding.TextUnmarshaler", path, value)
		}
		if _, repeated := seen[physical.Type()]; repeated {
			var zero T
			return zero, fmt.Errorf("%s: recursive pointer custom String representation %T", path, value)
		}
		seen[physical.Type()] = struct{}{}
		physical.Set(reflect.New(physical.Type().Elem()))
		physical = physical.Elem()
	}
}
