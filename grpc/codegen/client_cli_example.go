package codegen

import (
	"fmt"
	"reflect"

	"github.com/CaliLuke/loom/expr"
)

// protoJSONExample returns an example value of the protocol buffer message
// att whose JSON encoding protojson decodes into the Go type of the message.
// The command-line client decodes the value of the message flag with
// protojson, since encoding/json/v2 cannot set oneof fields.
//
// The protocol buffer JSON mapping names each field after its protocol buffer
// name and sets a oneof by the name of the selected oneof field, at the level
// of the message that holds the oneof. A value of a type that holds no union
// is the example that att generates, with its object keys renamed to the
// protocol buffer names, so that the example of such a message and the
// examples generated after it do not change. The examples of the types that
// hold a union select a branch of each union, and omit a field whose type
// refers to a type that is already being generated.
func protoJSONExample(att *expr.AttributeExpr, r *expr.ExampleGenerator) any {
	return protoJSONExampleR(att, r, make(map[string]struct{}))
}

// protoJSONExampleR is the recursive implementation of protoJSONExample.
// seen holds the identifiers of the user types being generated.
func protoJSONExampleR(att *expr.AttributeExpr, r *expr.ExampleGenerator, seen map[string]struct{}) any {
	if _, ok := att.Meta["struct:field:proto"]; ok || !holdsUnion(att, make(map[string]struct{})) {
		value, _ := protoJSONValue(att, att.Example(r))
		return value
	}
	if ut, ok := att.Type.(expr.UserType); ok {
		if _, ok := seen[ut.ID()]; ok {
			return nil
		}
		seen[ut.ID()] = struct{}{}
		defer delete(seen, ut.ID())
		return protoJSONExampleR(ut.Attribute(), r, seen)
	}
	switch actual := att.Type.(type) {
	case *expr.Object:
		return protoJSONObjectExample(actual, r, seen)
	case *expr.Union:
		return protoJSONUnionExample(att, actual, r, seen)
	case *expr.Array:
		return protoJSONArrayExample(att, actual, r, seen)
	case *expr.Map:
		return protoJSONMapExample(actual, r, seen)
	default:
		return att.Example(r)
	}
}

// protoJSONObjectExample returns the example of the message with the fields
// obj. A union field is set by the name of the oneof field of its selected
// branch.
func protoJSONObjectExample(obj *expr.Object, r *expr.ExampleGenerator, seen map[string]struct{}) map[string]any {
	names := newProtoMessageNames(obj)
	res := make(map[string]any, len(*obj))
	for _, nat := range *obj {
		if union := expr.AsUnion(nat.Attribute.Type); union != nil {
			if key, value := protoJSONOneofExample(union, names.oneofFields(nat.Name), r, seen); key != "" {
				res[key] = value
			}
			continue
		}
		if value := protoJSONExampleR(nat.Attribute, r, seen); value != nil {
			res[names.field(nat.Name)] = value
		}
	}
	return res
}

// protoJSONUnionExample returns the example of the union att that is a
// message on its own, such as array elements: the message with a single
// oneof, see protoBufUnionMessageDef. It returns nil when no branch has an
// example.
func protoJSONUnionExample(att *expr.AttributeExpr, union *expr.Union, r *expr.ExampleGenerator, seen map[string]struct{}) any {
	name := union.Name()
	names := newProtoMessageNames(&expr.Object{{Name: name, Attribute: att}})
	key, value := protoJSONOneofExample(union, names.oneofFields(name), r, seen)
	if key == "" {
		return nil
	}
	return map[string]any{key: value}
}

// protoJSONArrayExample returns the example of the repeated field att with
// the elements of array.
func protoJSONArrayExample(att *expr.AttributeExpr, array *expr.Array, r *expr.ExampleGenerator, seen map[string]struct{}) []any {
	count := expr.NewLength(att, r)
	res := make([]any, 0, count)
	for range count {
		if value := protoJSONExampleR(array.ElemType, r, seen); value != nil {
			res = append(res, value)
		}
	}
	return res
}

// protoJSONMapExample returns the example of the map field m.
func protoJSONMapExample(m *expr.Map, r *expr.ExampleGenerator, seen map[string]struct{}) map[string]any {
	count := r.Int()%3 + 1
	res := make(map[string]any, count)
	for range count {
		key := m.KeyType.Example(r)
		value := protoJSONExampleR(m.ElemType, r, seen)
		if key != nil && value != nil {
			res[protoJSONMapKey(key)] = value
		}
	}
	return res
}

// protoJSONOneofExample returns the name of the oneof field of the selected
// branch of union, whose oneof fields have the names fields, and the example
// of the branch. It selects a branch at random and the next branch that has
// an example when the branch refers to a type being generated. It returns an
// empty name when no branch has an example.
func protoJSONOneofExample(union *expr.Union, fields []string, r *expr.ExampleGenerator, seen map[string]struct{}) (string, any) {
	if len(union.Values) == 0 {
		return "", nil
	}
	start := r.Int() % len(union.Values)
	for i := range union.Values {
		branch := (start + i) % len(union.Values)
		if value := protoJSONExampleR(union.Values[branch].Attribute, r, seen); value != nil {
			return fields[branch], value
		}
	}
	return "", nil
}

// protoJSONValue returns the value example of the attribute att with the keys
// of the objects renamed to the protocol buffer field names and the keys that
// name no field removed. It also returns whether the value changed, and
// returns value itself when it did not, so that the example keeps its Go
// types and its JSON encoding.
func protoJSONValue(att *expr.AttributeExpr, value any) (any, bool) {
	if value == nil {
		return nil, false
	}
	if _, ok := att.Meta["struct:field:proto"]; ok {
		return value, false
	}
	if ut, ok := att.Type.(expr.UserType); ok {
		return protoJSONValue(ut.Attribute(), value)
	}
	switch actual := att.Type.(type) {
	case *expr.Object:
		return protoJSONObjectValue(actual, value)
	case *expr.Array:
		return protoJSONArrayValue(actual, value)
	case *expr.Map:
		return protoJSONMapValue(actual, value)
	default:
		return value, false
	}
}

// protoJSONObjectValue returns the object value of the message with the
// fields obj with its keys renamed to the protocol buffer names, see
// protoJSONValue.
func protoJSONObjectValue(obj *expr.Object, value any) (any, bool) {
	m, ok := value.(map[string]any)
	if !ok {
		return value, false
	}
	names := newProtoMessageNames(obj)
	res := make(map[string]any, len(m))
	changed := false
	for _, nat := range *obj {
		v, ok := m[nat.Name]
		if !ok {
			continue
		}
		converted, c := protoJSONValue(nat.Attribute, v)
		name := names.field(nat.Name)
		changed = changed || c || name != nat.Name
		res[name] = converted
	}
	if !changed && len(res) == len(m) {
		return value, false
	}
	return res, true
}

// protoJSONArrayValue returns the slice value with the elements of array
// converted by protoJSONValue.
func protoJSONArrayValue(array *expr.Array, value any) (any, bool) {
	rv := reflect.ValueOf(value)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return value, false
	}
	res := make([]any, rv.Len())
	changed := false
	for i := range rv.Len() {
		converted, c := protoJSONValue(array.ElemType, rv.Index(i).Interface())
		changed = changed || c
		res[i] = converted
	}
	if !changed {
		return value, false
	}
	return res, true
}

// protoJSONMapValue returns the map value with the values of m converted by
// protoJSONValue.
func protoJSONMapValue(m *expr.Map, value any) (any, bool) {
	rv := reflect.ValueOf(value)
	if rv.Kind() != reflect.Map {
		return value, false
	}
	res := make(map[string]any, rv.Len())
	changed := false
	iter := rv.MapRange()
	for iter.Next() {
		converted, c := protoJSONValue(m.ElemType, iter.Value().Interface())
		changed = changed || c
		res[protoJSONMapKey(iter.Key().Interface())] = converted
	}
	if !changed {
		return value, false
	}
	return res, true
}

// protoJSONMapKey returns the JSON object key of the map key example key.
// The protocol buffer JSON mapping encodes integer and boolean keys as their
// decimal and literal text.
func protoJSONMapKey(key any) string {
	if s, ok := key.(string); ok {
		return s
	}
	return fmt.Sprint(key)
}

// holdsUnion reports whether the type of att is a union or holds a union in
// an object field, array element or map value. seen holds the identifiers of
// the user types already visited.
func holdsUnion(att *expr.AttributeExpr, seen map[string]struct{}) bool {
	if ut, ok := att.Type.(expr.UserType); ok {
		if _, ok := seen[ut.ID()]; ok {
			return false
		}
		seen[ut.ID()] = struct{}{}
		return holdsUnion(ut.Attribute(), seen)
	}
	switch actual := att.Type.(type) {
	case *expr.Union:
		return true
	case *expr.Object:
		for _, nat := range *actual {
			if holdsUnion(nat.Attribute, seen) {
				return true
			}
		}
	case *expr.Array:
		return holdsUnion(actual.ElemType, seen)
	case *expr.Map:
		return holdsUnion(actual.ElemType, seen)
	}
	return false
}
