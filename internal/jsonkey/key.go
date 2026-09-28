// Package jsonkey defines canonical JSON object member names for DSL map keys.
package jsonkey

import (
	"encoding/json/v2"
	"reflect"
	"strconv"
)

// Name returns the JSON object member name of a string, boolean, or finite
// numeric map key, including named scalar types and interface wrappers. It
// reports false for nil, non-finite numbers, and unsupported key kinds.
func Name(key reflect.Value) (string, bool) {
	for key.IsValid() && key.Kind() == reflect.Interface {
		if key.IsNil() {
			return "", false
		}
		key = key.Elem()
	}
	if !key.IsValid() {
		return "", false
	}
	switch key.Kind() {
	case reflect.String:
		return key.String(), true
	case reflect.Bool:
		return strconv.FormatBool(key.Bool()), true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(key.Int(), 10), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return strconv.FormatUint(key.Uint(), 10), true
	case reflect.Float32:
		return jsonNumberText(float32(key.Float()))
	case reflect.Float64:
		return jsonNumberText(key.Float())
	default:
		return "", false
	}
}

// jsonNumberText returns the JSON text of the float number, or false when
// JSON cannot represent it.
func jsonNumberText[T float32 | float64](number T) (string, bool) {
	text, err := json.Marshal(number)
	if err != nil {
		return "", false
	}
	return string(text), true
}
