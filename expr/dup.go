package expr

import (
	"fmt"
)

// Dup creates a copy the given data type.
func Dup(d DataType) DataType {
	return newDupper().DupType(d)
}

// DupKeeping creates a copy of the given data type like Dup, except that it
// keeps the references to the user types in keep instead of copying them. It
// also returns the attributes of the copy that reference a kept user type.
// Use it to copy a type that references user types whose DSL is still
// running, so that the copy does not snapshot their incomplete definitions.
func DupKeeping(d DataType, keep []UserType) (DataType, []*AttributeExpr) {
	dupper := newDupper()
	dupper.keep = make(map[string]struct{}, len(keep))
	for _, ut := range keep {
		dupper.keep[ut.ID()] = struct{}{}
	}
	return dupper.DupType(d), dupper.kept
}

// DupAtt creates a copy of the given attribute.
func DupAtt(att *AttributeExpr) *AttributeExpr {
	dupper := newDupper()
	duppedBases := make([]DataType, len(att.Bases))
	for i, b := range att.Bases {
		duppedBases[i] = dupper.DupType(b)
	}
	res := dupper.DupAttribute(att)
	res.Bases = duppedBases
	return res
}

// dupper implements recursive and cycle safe copy of data types.
type dupper struct {
	uts map[string]UserType
	ats map[*AttributeExpr]struct{}
	// keep lists the IDs of the user types that the dupper does not copy.
	keep map[string]struct{}
	// kept lists the copied attributes that reference a kept user type.
	kept []*AttributeExpr
}

// newDupper returns a new initialized dupper.
func newDupper() *dupper {
	return &dupper{
		uts: make(map[string]UserType),
		ats: make(map[*AttributeExpr]struct{}),
	}
}

// DupAttribute creates a copy of the given attribute.
func (d *dupper) DupAttribute(att *AttributeExpr) *AttributeExpr {
	if _, ok := d.ats[att]; ok {
		return att
	}
	var valDup *ValidationExpr
	if att.Validation != nil {
		valDup = att.Validation.Dup()
	}
	var metaDup MetaExpr
	if att.Meta != nil {
		metaDup = att.Meta.Dup()
	}
	dup := AttributeExpr{
		Type:         d.DupType(att.Type),
		Description:  att.Description,
		Title:        att.Title,
		References:   att.References,
		Bases:        att.Bases,
		Validation:   valDup,
		Meta:         metaDup,
		DefaultValue: att.DefaultValue,
		Nullable:     att.Nullable,
		DSLFunc:      att.DSLFunc,
		UserExamples: att.UserExamples,
		finalized:    att.finalized,
	}
	d.ats[&dup] = struct{}{}
	if d.isKept(dup.Type) {
		d.kept = append(d.kept, &dup)
	}
	return &dup
}

// isKept reports whether t is a user type that d does not copy.
func (d *dupper) isKept(t DataType) bool {
	ut, ok := t.(UserType)
	if !ok {
		return false
	}
	_, ok = d.keep[ut.ID()]
	return ok
}

// DupType creates a copy of the given data type.
func (d *dupper) DupType(t DataType) DataType {
	if t == Empty {
		// Don't dup Empty so that code may check against it.
		return t
	}
	switch actual := t.(type) {
	case Primitive:
		return t
	case *Array:
		return &Array{
			ElemType:         d.DupAttribute(actual.ElemType),
			NonNullableElems: actual.NonNullableElems,
		}
	case *Object:
		res := &Object{}
		for _, nat := range *actual {
			res.Set(nat.Name, d.DupAttribute(nat.Attribute))
		}
		return res
	case *Map:
		return &Map{
			KeyType:  d.DupAttribute(actual.KeyType),
			ElemType: d.DupAttribute(actual.ElemType),
		}
	case *Union:
		dp := Union{
			TypeName:         actual.TypeName,
			ExplicitTypeName: actual.ExplicitTypeName,
			Untagged:         actual.Untagged,
			Values:           make([]*NamedAttributeExpr, len(actual.Values)),
			TypeKey:          actual.TypeKey,
			ValueKey:         actual.ValueKey,
		}
		for i, nat := range actual.Values {
			dp.Values[i] = &NamedAttributeExpr{Name: nat.Name, Attribute: d.DupAttribute(nat.Attribute)}
		}
		return &dp
	case UserType:
		if u, ok := d.uts[actual.ID()]; ok {
			return u
		}
		if d.isKept(actual) {
			return actual
		}
		dp := actual.Dup(nil)
		d.uts[actual.ID()] = dp
		dupAtt := d.DupAttribute(actual.Attribute())
		dp.SetAttribute(dupAtt)

		// Make sure that if we are dupping a generated type we also put
		// the dup in the generated type list so that it gets properly
		// eval'd.
		if rt, ok := dp.(*ResultTypeExpr); ok {
			if GeneratedResultType(rt.Identifier) != nil {
				GeneratedResultTypes.Append(rt)
			}
		}

		return dp
	}
	panic("unknown type " + fmt.Sprintf("%T", t))
}
