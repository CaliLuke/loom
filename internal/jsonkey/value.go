package jsonkey

import (
	"fmt"
	"reflect"
)

// Value canonicalizes builtin map keys in an acyclic, captured value. Scalars
// retain precision and byte slices retain their JSON byte encoding. Custom
// encoders and maps with custom key encoders keep ownership of their wire form.
// Unsupported or colliding builtin keys are errors.
func Value(value any) (any, error) {
	actual := reflect.ValueOf(value)
	if !actual.IsValid() || HasCustomEncoding(actual.Type()) {
		return value, nil
	}
	switch actual.Kind() {
	case reflect.Map:
		for _, key := range actual.MapKeys() {
			if HasCustomEncoding(reflect.TypeOf(key.Interface())) {
				return value, nil
			}
		}
		result := make(map[string]any, actual.Len())
		iterator := actual.MapRange()
		for iterator.Next() {
			name, ok := Name(iterator.Key())
			if !ok {
				return nil, fmt.Errorf("unsupported JSON map key %T", iterator.Key().Interface())
			}
			if _, exists := result[name]; exists {
				return nil, fmt.Errorf("colliding JSON map key %q", name)
			}
			child, err := Value(iterator.Value().Interface())
			if err != nil {
				return nil, err
			}
			result[name] = child
		}
		return result, nil
	case reflect.Slice, reflect.Array:
		if actual.Kind() == reflect.Slice && actual.Type().Elem().Kind() == reflect.Uint8 {
			return value, nil
		}
		result := make([]any, actual.Len())
		for index := range result {
			child, err := Value(actual.Index(index).Interface())
			if err != nil {
				return nil, err
			}
			result[index] = child
		}
		return result, nil
	default:
		return value, nil
	}
}
