package expr

// Convenience methods

// AsObject returns the type underlying object if any, nil otherwise.
func AsObject(dt DataType) *Object {
	switch t := dt.(type) {
	case *UserTypeExpr:
		return AsObject(t.Type)
	case *ResultTypeExpr:
		return AsObject(t.Type)
	case *Object:
		return t
	default:
		return nil
	}
}

// AsArray returns the type underlying array if any, nil otherwise.
func AsArray(dt DataType) *Array {
	switch t := dt.(type) {
	case *UserTypeExpr:
		return AsArray(t.Type)
	case *ResultTypeExpr:
		return AsArray(t.Type)
	case *Array:
		return t
	default:
		return nil
	}
}

// AsMap returns the type underlying map if any, nil otherwise.
func AsMap(dt DataType) *Map {
	switch t := dt.(type) {
	case *UserTypeExpr:
		return AsMap(t.Type)
	case *ResultTypeExpr:
		return AsMap(t.Type)
	case *Map:
		return t
	default:
		return nil
	}
}

// AsUnion returns the type underlying union if any, nil otherwise.
func AsUnion(dt DataType) *Union {
	switch t := dt.(type) {
	case *UserTypeExpr:
		return AsUnion(t.Type)
	case *ResultTypeExpr:
		return AsUnion(t.Type)
	case *Union:
		return t
	default:
		return nil
	}
}

// IsObject returns true if the data type is an object.
func IsObject(dt DataType) bool { return AsObject(dt) != nil }

// IsArray returns true if the data type is an array.
func IsArray(dt DataType) bool { return AsArray(dt) != nil }

// IsMap returns true if the data type is a map.
func IsMap(dt DataType) bool { return AsMap(dt) != nil }

// IsUnion returns true if the data type is a map.
func IsUnion(dt DataType) bool { return AsUnion(dt) != nil }

// IsPrimitive returns true if the data type is a primitive type.
func IsPrimitive(dt DataType) bool {
	switch t := dt.(type) {
	case Primitive:
		return true
	case *UserTypeExpr:
		return IsPrimitive(t.Type)
	case *ResultTypeExpr:
		return IsPrimitive(t.Type)
	default:
		return false
	}
}

// IsAlias returns true if the data type is a user type backed by a primitive
// type (so call aliased type).
func IsAlias(dt DataType) bool {
	_, isut := dt.(UserType)
	return isut && IsPrimitive(dt)
}

// Equal compares the types recursively and returns true if they are equal. Two
// types are equal if:
//
//   - both types have the same kind
//   - array types have elements whose types are equal
//   - map types have keys and elements whose types are equal
//   - objects have the same attribute names and the attribute types are equal
//
// Note: calling Equal is not equivalent to evaluating dt.Hash() == dt2.Hash()
// as the former may return true for two user types with different names and
// thus with different hash values.
func Equal(dt, dt2 DataType) bool {
	return Hash(dt, false, true, true) == Hash(dt2, false, true, true)
}

// DataType implementation

