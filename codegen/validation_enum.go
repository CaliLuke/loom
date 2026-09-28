package codegen

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/CaliLuke/loom/expr"
)

// collectionEnumValue applies the same primitive coercions as generated Go
// comparisons before projecting the value onto its JSON representation. In
// particular, floats round to their declared precision and Bytes accepts text.
// Non-null nil collections become empty collections, as encoding/json/v2 does.
func collectionEnumValue(attribute *expr.AttributeExpr, value any) any {
	if value == nil {
		return nil
	}
	switch actual := unalias(attribute.Type).(type) {
	case *expr.Array:
		input := reflect.ValueOf(value)
		values := make([]any, input.Len())
		for index := range values {
			values[index] = collectionEnumValue(actual.ElemType, input.Index(index).Interface())
		}
		return values
	case *expr.Map:
		input := reflect.ValueOf(value)
		values := make(map[any]any, input.Len())
		iterator := input.MapRange()
		for iterator.Next() {
			key := iterator.Key().Interface()
			if unalias(actual.KeyType.Type).Kind() != expr.AnyKind {
				key = collectionEnumValue(actual.KeyType, key)
			}
			values[key] = collectionEnumValue(actual.ElemType, iterator.Value().Interface())
		}
		return expr.CanonicalizeExample(attribute, values)
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
		bytes := value.([]byte)
		copied := make([]byte, len(bytes))
		copy(copied, bytes)
		return copied
	case expr.Any:
		// Keep Any's own JSON semantics when enclosing object/map projection
		// traverses this value, including typed nil values and raw JSON.
		encoded, err := json.Marshal(value, json.Deterministic(true))
		if err != nil {
			panic(fmt.Sprintf("encode arbitrary JSON enum value: %v", err))
		}
		return jsontext.Value(encoded)
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
	values := make(map[string]any, input.Len())
	iterator := input.MapRange()
	for iterator.Next() {
		key := iterator.Key()
		for key.Kind() == reflect.Interface {
			key = key.Elem()
		}
		name := key.String()
		field := &expr.AttributeExpr{Type: expr.Any}
		for _, candidate := range *object {
			wireName := expr.JSONFieldName(expr.ElementName(candidate.Name), candidate.Attribute)
			if name == candidate.Name || name == wireName {
				field = candidate.Attribute
				break
			}
		}
		values[name] = collectionEnumValue(field, iterator.Value().Interface())
	}
	return expr.CanonicalizeExample(attribute, values)
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
	envelope, ok := projected.(map[string]any)
	if !ok {
		return projected
	}
	for _, branch := range union.Values {
		if envelope[union.GetTypeKey()] != expr.UnionVariantTag(branch) {
			continue
		}
		normalized := collectionEnumValue(branch.Attribute, value)
		if union.Untagged {
			return normalized
		}
		return map[string]any{union.GetTypeKey(): expr.UnionVariantTag(branch), union.GetValueKey(): normalized}
	}
	return projected
}
