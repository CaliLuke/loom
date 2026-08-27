package expr

// AllowsNull reports whether an attribute occurrence accepts an explicit null
// value. Any is intrinsically null-admitting, and nullable named types carry
// their nullability to each occurrence.
func AllowsNull(attribute *AttributeExpr) bool {
	return IsNullable(attribute) || attribute != nil && resolvesToAny(attribute.Type, make(map[string]struct{}))
}

// ArrayElementsAllowNull reports whether an array accepts explicit null
// elements. ArrayOfRequired overrides the element type's intrinsic nullability.
func ArrayElementsAllowNull(array *Array) bool {
	return array != nil && !array.NonNullableElems && AllowsNull(array.ElemType)
}

// ContainsNonNullableArrayElement reports whether an attribute contains an
// array whose element contract rejects null.
func ContainsNonNullableArrayElement(attribute *AttributeExpr) bool {
	return containsNonNullableArrayElement(attribute, make(map[string]struct{}))
}

func containsNonNullableArrayElement(attribute *AttributeExpr, seen map[string]struct{}) bool {
	if attribute == nil || attribute.Type == nil {
		return false
	}
	if userType, ok := attribute.Type.(UserType); ok {
		if _, ok := seen[userType.ID()]; ok {
			return false
		}
		seen[userType.ID()] = struct{}{}
		defer delete(seen, userType.ID())
		return containsNonNullableArrayElement(userType.Attribute(), seen)
	}
	if array := AsArray(attribute.Type); array != nil {
		return !ArrayElementsAllowNull(array) || containsNonNullableArrayElement(array.ElemType, seen)
	}
	if mapping := AsMap(attribute.Type); mapping != nil {
		return containsNonNullableArrayElement(mapping.KeyType, seen) ||
			containsNonNullableArrayElement(mapping.ElemType, seen)
	}
	if object := AsObject(attribute.Type); object != nil {
		for _, field := range *object {
			if containsNonNullableArrayElement(field.Attribute, seen) {
				return true
			}
		}
	}
	if union := AsUnion(attribute.Type); union != nil {
		for _, value := range union.Values {
			if containsNonNullableArrayElement(value.Attribute, seen) {
				return true
			}
		}
	}
	return false
}

// IsNullable reports whether nullability is explicitly declared on an
// attribute occurrence or inherited from a named type.
func IsNullable(attribute *AttributeExpr) bool {
	return isNullable(attribute, make(map[string]struct{}))
}

func isNullable(attribute *AttributeExpr, seen map[string]struct{}) bool {
	if attribute == nil {
		return false
	}
	if attribute.Nullable {
		return true
	}
	userType, ok := attribute.Type.(UserType)
	if !ok {
		return false
	}
	if _, ok := seen[userType.ID()]; ok {
		return false
	}
	seen[userType.ID()] = struct{}{}
	return isNullable(userType.Attribute(), seen)
}

func containsNullable(attribute *AttributeExpr) bool {
	return containsNullableRecursive(attribute, make(map[string]struct{}))
}

func containsUntaggedUnion(attribute *AttributeExpr) bool {
	return containsUntaggedUnionRecursive(attribute, make(map[string]struct{}))
}

func containsUntaggedUnionRecursive(attribute *AttributeExpr, seen map[string]struct{}) bool {
	if attribute == nil || attribute.Type == nil || attribute.Type == Empty {
		return false
	}
	if union := AsUnion(attribute.Type); union != nil {
		if union.Untagged {
			return true
		}
		for _, branch := range union.Values {
			if containsUntaggedUnionRecursive(branch.Attribute, seen) {
				return true
			}
		}
		return false
	}
	if userType, ok := attribute.Type.(UserType); ok {
		if _, ok := seen[userType.ID()]; ok {
			return false
		}
		seen[userType.ID()] = struct{}{}
		defer delete(seen, userType.ID())
		return containsUntaggedUnionRecursive(userType.Attribute(), seen)
	}
	if object := AsObject(attribute.Type); object != nil {
		for _, named := range *object {
			if containsUntaggedUnionRecursive(named.Attribute, seen) {
				return true
			}
		}
	}
	if array := AsArray(attribute.Type); array != nil {
		return containsUntaggedUnionRecursive(array.ElemType, seen)
	}
	if mapping := AsMap(attribute.Type); mapping != nil {
		return containsUntaggedUnionRecursive(mapping.KeyType, seen) ||
			containsUntaggedUnionRecursive(mapping.ElemType, seen)
	}
	return false
}

func containsUnsupportedGRPCPresence(attribute *AttributeExpr) bool {
	return containsUnsupportedGRPCPresenceRecursive(attribute, false, make(map[string]struct{}))
}

func containsUnsupportedGRPCPresenceRecursive(attribute *AttributeExpr, objectField bool, seen map[string]struct{}) bool {
	if attribute == nil || attribute.Type == nil || attribute.Type == Empty {
		return false
	}
	anyType := resolvesToAny(attribute.Type, make(map[string]struct{}))
	if IsNullable(attribute) && (!anyType || !objectField) {
		return true
	}
	if userType, ok := attribute.Type.(UserType); ok {
		if _, ok := seen[userType.ID()]; ok {
			return false
		}
		seen[userType.ID()] = struct{}{}
		defer delete(seen, userType.ID())
		return containsUnsupportedGRPCPresenceRecursive(userType.Attribute(), objectField, seen)
	}
	if array := AsArray(attribute.Type); array != nil {
		return containsUnsupportedGRPCPresenceRecursive(array.ElemType, false, seen)
	}
	if mapping := AsMap(attribute.Type); mapping != nil {
		return containsUnsupportedGRPCPresenceRecursive(mapping.KeyType, false, seen) ||
			containsUnsupportedGRPCPresenceRecursive(mapping.ElemType, false, seen)
	}
	if object := AsObject(attribute.Type); object != nil {
		for _, named := range *object {
			if containsUnsupportedGRPCPresenceRecursive(named.Attribute, true, seen) {
				return true
			}
		}
	}
	if union := AsUnion(attribute.Type); union != nil {
		for _, named := range union.Values {
			if containsUnsupportedGRPCPresenceRecursive(named.Attribute, false, seen) {
				return true
			}
		}
	}
	return false
}

func containsOptionalJSONStringTag(attribute *AttributeExpr) bool {
	return containsOptionalJSONStringTagRecursive(attribute, make(map[string]struct{}))
}

func containsOptionalJSONStringTagRecursive(attribute *AttributeExpr, seen map[string]struct{}) bool {
	if attribute == nil || attribute.Type == nil || attribute.Type == Empty {
		return false
	}
	if userType, ok := attribute.Type.(UserType); ok {
		if _, ok := seen[userType.ID()]; ok {
			return false
		}
		seen[userType.ID()] = struct{}{}
		defer delete(seen, userType.ID())
		return containsOptionalJSONStringTagRecursive(userType.Attribute(), seen)
	}
	if object := AsObject(attribute.Type); object != nil {
		for _, named := range *object {
			if !attribute.IsRequiredNoDefault(named.Name) {
				if tag, ok := named.Attribute.Meta.Last("struct:tag:json"); ok && jsonTagHasOption(tag, "string") {
					return true
				}
			}
			if containsOptionalJSONStringTagRecursive(named.Attribute, seen) {
				return true
			}
		}
	}
	if array := AsArray(attribute.Type); array != nil {
		return containsOptionalJSONStringTagRecursive(array.ElemType, seen)
	}
	if mapping := AsMap(attribute.Type); mapping != nil {
		return containsOptionalJSONStringTagRecursive(mapping.ElemType, seen)
	}
	if union := AsUnion(attribute.Type); union != nil {
		for _, named := range union.Values {
			if containsOptionalJSONStringTagRecursive(named.Attribute, seen) {
				return true
			}
		}
	}
	return false
}

func containsNullableRecursive(attribute *AttributeExpr, seen map[string]struct{}) bool {
	if attribute == nil || attribute.Type == nil || attribute.Type == Empty {
		return false
	}
	if AllowsNull(attribute) {
		return true
	}
	if userType, ok := attribute.Type.(UserType); ok {
		if _, ok := seen[userType.ID()]; ok {
			return false
		}
		seen[userType.ID()] = struct{}{}
		defer delete(seen, userType.ID())
		return containsNullableRecursive(userType.Attribute(), seen)
	}
	if array := AsArray(attribute.Type); array != nil {
		return containsNullableRecursive(array.ElemType, seen)
	}
	if mapping := AsMap(attribute.Type); mapping != nil {
		return containsNullableRecursive(mapping.KeyType, seen) ||
			containsNullableRecursive(mapping.ElemType, seen)
	}
	if object := AsObject(attribute.Type); object != nil {
		for _, named := range *object {
			if containsNullableRecursive(named.Attribute, seen) {
				return true
			}
		}
	}
	if union := AsUnion(attribute.Type); union != nil {
		for _, named := range union.Values {
			if containsNullableRecursive(named.Attribute, seen) {
				return true
			}
		}
	}
	return false
}

func resolvesToAny(dataType DataType, seen map[string]struct{}) bool {
	if dataType == Any {
		return true
	}
	userType, ok := dataType.(UserType)
	if !ok {
		return false
	}
	if _, ok := seen[userType.ID()]; ok {
		return false
	}
	seen[userType.ID()] = struct{}{}
	return resolvesToAny(userType.Attribute().Type, seen)
}
