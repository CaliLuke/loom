package expr

// typeComparer compares data types structurally. It records the pairs of
// user types and objects under comparison so that recursive types terminate:
// a pair met again is assumed equal, and any difference found elsewhere still
// makes the whole comparison fail. Each pair is compared at most once, so the
// cost is bounded by the product of the sizes of the two type graphs.
type typeComparer struct {
	// seen holds the pairs of user types or objects already compared.
	seen map[[2]DataType]struct{}
	// visits counts every lookup of a pair of user types or objects,
	// including pairs already seen, so it measures the comparison work.
	visits int
}

// Equal compares the types recursively and returns true if they are equal. Two
// types are equal if:
//
//   - both types have the same kind
//   - primitive types have the same name
//   - array types have elements whose types are equal
//   - map types have keys and elements whose types are equal
//   - union types have the same type name, tagging, and variant names, and
//     the variant types are equal
//   - user and result types have the same nullability and their underlying
//     types are equal, whatever their names
//   - objects have the same attribute names and the attribute types are equal
//   - attributes being compared have the same nullability
//
// Equal ignores type names, struct field tags, and validations. Recursive
// types are compared coinductively, so a type that references itself is never
// equal to a finite type.
//
// Note: calling Equal is not equivalent to evaluating dt.Hash() == dt2.Hash()
// as the former may return true for two user types with different names and
// thus with different hash values.
func Equal(dt, dt2 DataType) bool {
	c := &typeComparer{seen: make(map[[2]DataType]struct{})}
	return c.equal(dt, dt2)
}

func isPrimitiveKind(k Kind) bool {
	switch k {
	case BooleanKind, IntKind, Int32Kind, Int64Kind, UIntKind, UInt32Kind, UInt64Kind, Float32Kind, Float64Kind, StringKind, BytesKind, AnyKind:
		return true
	default:
		return false
	}
}

func isNamedKind(k Kind) bool {
	return k == UserTypeKind || k == ResultTypeKind
}

func sortedUnionValues(u *Union) []*NamedAttributeExpr {
	return sorted((*Object)(&u.Values))
}

func (c *typeComparer) equal(a, b DataType) bool {
	ka, kb := a.Kind(), b.Kind()
	switch {
	case isPrimitiveKind(ka):
		return isPrimitiveKind(kb) && a.Name() == b.Name()
	case isNamedKind(ka):
		if !isNamedKind(kb) {
			return false
		}
		if c.revisit(a, b) {
			return true
		}
		ua, ub := a.(UserType).Attribute(), b.(UserType).Attribute()
		return IsNullable(ua) == IsNullable(ub) && c.equal(ua.Type, ub.Type)
	case ka != kb:
		return false
	}
	switch ta := a.(type) {
	case *Array:
		return c.equalAttributes(ta.ElemType, b.(*Array).ElemType)
	case *Map:
		tb := b.(*Map)
		return c.equalAttributes(ta.KeyType, tb.KeyType) && c.equalAttributes(ta.ElemType, tb.ElemType)
	case *Union:
		return c.equalUnions(ta, b.(*Union))
	case *Object:
		if c.revisit(a, b) {
			return true
		}
		return c.equalNamedAttributes(sorted(ta), sorted(b.(*Object)))
	default:
		return false
	}
}

// revisit reports whether the pair is already being or has been compared and
// records it otherwise.
func (c *typeComparer) revisit(a, b DataType) bool {
	c.visits++
	key := [2]DataType{a, b}
	if _, ok := c.seen[key]; ok {
		return true
	}
	c.seen[key] = struct{}{}
	return false
}

func (c *typeComparer) equalUnions(a, b *Union) bool {
	if a.TypeName != b.TypeName || a.Untagged != b.Untagged {
		return false
	}
	return c.equalNamedAttributes(sortedUnionValues(a), sortedUnionValues(b))
}

func (c *typeComparer) equalNamedAttributes(a, b []*NamedAttributeExpr) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Name != b[i].Name || !c.equalAttributes(a[i].Attribute, b[i].Attribute) {
			return false
		}
	}
	return true
}

func (c *typeComparer) equalAttributes(a, b *AttributeExpr) bool {
	return IsNullable(a) == IsNullable(b) && c.equal(a.Type, b.Type)
}
