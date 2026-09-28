// Package enumvalue projects authored enum values onto their declared JSON shapes.
package enumvalue

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/examplevalue"
)

// Normalize projects a DSL value onto its declared JSON shape without
// mutating the attribute or value. It applies the primitive coercions used by
// generated Go comparisons: floats round to their declared precision and Bytes
// accepts text. Non-null nil collections become empty collections, as
// encoding/json/v2 does. Any values retain their original Go representation.
// Pointers are dereferenced for declared types; incompatible example shapes are
// left unchanged for the caller to validate or omit.
func Normalize(attribute *expr.AttributeExpr, value any) any {
	if value == nil || attribute == nil || attribute.Type == nil {
		return value
	}
	datatype := baseType(attribute.Type)
	if datatype == expr.Any {
		return value
	}
	input := reflect.ValueOf(value)
	for input.Kind() == reflect.Pointer || input.Kind() == reflect.Interface {
		if input.IsNil() {
			return nil
		}
		input = input.Elem()
	}
	value = input.Interface()
	switch actual := datatype.(type) {
	case *expr.Array:
		if input.Kind() != reflect.Array && input.Kind() != reflect.Slice {
			return value
		}
		values := make([]any, input.Len())
		for index := range values {
			values[index] = Normalize(actual.ElemType, input.Index(index).Interface())
		}
		return values
	case *expr.Map:
		if input.Kind() != reflect.Map {
			return value
		}
		values := make(map[any]any, input.Len())
		iterator := input.MapRange()
		for iterator.Next() {
			key := iterator.Key().Interface()
			if baseType(actual.KeyType.Type).Kind() != expr.AnyKind {
				key = Normalize(actual.KeyType, key)
			}
			values[key] = Normalize(actual.ElemType, iterator.Value().Interface())
		}
		// Canonicalize keys without traversing the normalized values again:
		// Any must retain its exact Go/JSON value, including typed nil bytes.
		keysOnly := &expr.AttributeExpr{Type: &expr.Map{KeyType: actual.KeyType, ElemType: &expr.AttributeExpr{}}}
		return expr.CanonicalizeExample(keysOnly, values)
	case *expr.Object:
		return collectionEnumObjectValue(attribute, actual, value)
	case *expr.Union:
		return collectionEnumUnionValue(attribute, actual, value)
	case expr.Primitive:
		return collectionEnumPrimitiveValue(actual, value)
	default:
		return expr.CanonicalizeExample(attribute, value)
	}
}

func collectionEnumPrimitiveValue(primitive expr.Primitive, value any) any {
	switch primitive {
	case expr.Float32, expr.Float64:
		bits := 64
		if primitive == expr.Float32 {
			bits = 32
		}
		// Format the authored number as the Go literal emitter does, then
		// round that decimal constant to the destination precision. Direct
		// float32-to-float64 conversion would retain unwanted binary digits.
		number, err := strconv.ParseFloat(fmt.Sprint(value), bits)
		if err != nil {
			return value
		}
		if primitive == expr.Float32 {
			return float32(number)
		}
		return number
	case expr.Bytes:
		if text, ok := value.(string); ok {
			return []byte(text)
		}
		bytes, ok := value.([]byte)
		if !ok {
			return value
		}
		copied := make([]byte, len(bytes))
		copy(copied, bytes)
		return copied
	default:
		return value
	}
}

func collectionEnumObjectValue(attribute *expr.AttributeExpr, object *expr.Object, value any) any {
	input := reflect.ValueOf(value)
	if input.Kind() == reflect.Struct {
		return collectionEnumObjectValue(attribute, object, collectionEnumStructFields(input))
	}
	if input.Kind() != reflect.Map {
		return expr.CanonicalizeExample(attribute, value)
	}
	if input.IsNil() {
		return nil
	}
	authored := make(map[string]any, input.Len())
	iterator := input.MapRange()
	for iterator.Next() {
		key := iterator.Key()
		for key.Kind() == reflect.Interface {
			key = key.Elem()
		}
		authored[key.String()] = iterator.Value().Interface()
	}
	values := make(map[string]any, input.Len())
	for _, field := range *object {
		wireName := expr.JSONFieldName(expr.ElementName(field.Name), field.Attribute)
		value, present := authored[wireName]
		if !present {
			value, present = authored[field.Name]
		}
		delete(authored, wireName)
		delete(authored, field.Name)
		if present && wireName != "-" {
			values[wireName] = Normalize(field.Attribute, value)
		}
	}
	for name, value := range authored {
		values[name] = value
	}
	return values
}

// collectionEnumStructFields preserves authored struct field values so nested
// arrays, bytes and numbers can be coerced before JSON serialization.
func collectionEnumStructFields(value reflect.Value) map[string]any {
	fields := make(map[string]any)
	for index := range value.NumField() {
		field := value.Type().Field(index)
		if !field.IsExported() {
			continue
		}
		name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		if name == "-" {
			continue
		}
		item := value.Field(index)
		if field.Anonymous && name == "" && item.Kind() == reflect.Struct {
			for key, nested := range collectionEnumStructFields(item) {
				fields[key] = nested
			}
			continue
		}
		if name == "" {
			name = field.Name
		}
		fields[name] = item.Interface()
	}
	return fields
}

func collectionEnumUnionValue(attribute *expr.AttributeExpr, union *expr.Union, value any) any {
	// Use the shared example branch selection, temporarily retaining a tag for
	// untagged unions so the selected branch can receive its declared coercions.
	tagged := *union
	tagged.Untagged = false
	selection := *attribute
	selection.Type = &tagged
	projected := expr.CanonicalizeExample(&selection, value)
	if selected, ok := value.(examplevalue.Union); ok {
		value = selected.Value
	}
	envelope, ok := projected.(map[string]any)
	if !ok {
		return projected
	}
	for _, branch := range union.Values {
		if envelope[union.GetTypeKey()] != expr.UnionVariantTag(branch) {
			continue
		}
		normalized := Normalize(branch.Attribute, value)
		if union.Untagged {
			return normalized
		}
		return map[string]any{union.GetTypeKey(): expr.UnionVariantTag(branch), union.GetValueKey(): normalized}
	}
	return projected
}

func baseType(datatype expr.DataType) expr.DataType {
	for {
		userType, ok := datatype.(expr.UserType)
		if !ok {
			return datatype
		}
		datatype = userType.Attribute().Type
	}
}
