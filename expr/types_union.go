package expr

import (
	"fmt"
	"reflect"
)

// Kind implements DataKind.
func (*Union) Kind() Kind { return UnionKind }

// Name returns the type name.
func (u *Union) Name() string { return u.TypeName }

// Hash returns a unique hash value for m.
func (u *Union) Hash() string {
	return Hash(u, true, false, true)
}

// IsCompatible returns true if u describes the (Go) type of val.
func (u *Union) IsCompatible(val any) bool {
	for _, nat := range u.Values {
		if nat.Attribute.Type.IsCompatible(val) {
			return true
		}
	}
	return false
}

// Example returns a random example value.
func (u *Union) Example(r *ExampleGenerator) any {
	if len(u.Values) == 0 {
		return nil
	}
	return u.Values[r.Int()%len(u.Values)].Attribute.Example(r)
}

// GetTypeKey returns the discriminator field name for JSON marshaling.
// Defaults to "type" if not explicitly set via Meta("oneof:type:field").
func (u *Union) GetTypeKey() string {
	if u.TypeKey != "" {
		return u.TypeKey
	}
	return "type"
}

// GetValueKey returns the value field name for JSON marshaling.
// Defaults to "value" if not explicitly set via Meta("oneof:value:field").
func (u *Union) GetValueKey() string {
	if u.ValueKey != "" {
		return u.ValueKey
	}
	return "value"
}

// QualifiedTypeName returns the qualified type name for the given data type.
// The qualified type name includes the name of the type of the elements of
// array or map types. This is useful in reporting types in error messages,
// examples of qualified type names:
//
//	"array<string>"
//	"map<string, string>"
//	"map<string, array<int32>>"
func QualifiedTypeName(t DataType) string {
	switch t.Kind() {
	case ArrayKind:
		a := t.(*Array)
		return fmt.Sprintf("%s<%s>",
			t.Name(),
			QualifiedTypeName(a.ElemType.Type),
		)
	case MapKind:
		h := t.(*Map)
		return fmt.Sprintf("%s<%s, %s>",
			t.Name(),
			QualifiedTypeName(h.KeyType.Type),
			QualifiedTypeName(h.ElemType.Type),
		)
	}
	return t.Name()
}

// toReflectType converts the DataType to reflect.Type.
func toReflectType(dtype DataType) reflect.Type {
	switch dtype.Kind() {
	case BooleanKind:
		return reflect.TypeOf(true)
	case IntKind:
		return reflect.TypeOf(int(0))
	case Int32Kind:
		return reflect.TypeOf(int32(0))
	case Int64Kind:
		return reflect.TypeOf(int64(0))
	case UIntKind:
		return reflect.TypeOf(uint(0))
	case UInt32Kind:
		return reflect.TypeOf(uint32(0))
	case UInt64Kind:
		return reflect.TypeOf(uint64(0))
	case Float32Kind:
		return reflect.TypeOf(float32(0))
	case Float64Kind:
		return reflect.TypeOf(float64(0))
	case StringKind:
		return reflect.TypeOf("")
	case BytesKind:
		return reflect.TypeOf([]byte{})
	case ObjectKind:
		return reflect.TypeOf(map[string]any{})
	case UserTypeKind:
		return toReflectType(dtype.(*UserTypeExpr).Attribute().Type)
	case ResultTypeKind:
		return toReflectType(dtype.(*ResultTypeExpr).Attribute().Type)
	case ArrayKind:
		return reflect.SliceOf(toReflectType(dtype.(*Array).ElemType.Type))
	case MapKind:
		m := dtype.(*Map)
		// avoid complication: not allow object as the map key
		var ktype reflect.Type
		if m.KeyType.Type.Kind() != ObjectKind {
			ktype = toReflectType(m.KeyType.Type)
		} else {
			ktype = reflect.TypeOf([]any{}).Elem()
		}
		return reflect.MapOf(ktype, toReflectType(m.ElemType.Type))
	default:
		return reflect.TypeOf([]any{}).Elem()
	}
}
