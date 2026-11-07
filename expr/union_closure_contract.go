package expr

import "reflect"

type unionClosureContractPair struct {
	left, right *AttributeExpr
}

// sameUnionClosureContract compares finalized occurrences within one nominal
// identity. DSL-localized copies can change keys, branches, constraints or
// metadata without changing their original ID. Sharing a closed declaration
// requires the complete retained contract, not just that nominal identity.
func sameUnionClosureContract(left, right UserType) bool {
	return equalUnionClosureType(left, right, make(map[unionClosureContractPair]bool))
}

func equalUnionClosureType(left, right DataType, seen map[unionClosureContractPair]bool) bool {
	if left == right {
		return true
	}
	if left == nil || right == nil || left.Kind() != right.Kind() {
		return false
	}
	switch a := left.(type) {
	case UserType:
		b, ok := right.(UserType)
		if !ok || a.Name() != b.Name() || typeCopyKey(a) != typeCopyKey(b) {
			return false
		}
		if ar, ok := a.(*ResultTypeExpr); ok {
			br, ok := b.(*ResultTypeExpr)
			if !ok || !reflect.DeepEqual(ar.Views, br.Views) {
				return false
			}
		}
		return equalUnionClosureAttribute(a.Attribute(), b.Attribute(), seen)
	case *Union:
		b, ok := right.(*Union)
		if !ok {
			return false
		}
		ac, bc := *a, *b
		ac.Values, bc.Values = nil, nil
		return reflect.DeepEqual(ac, bc) && equalUnionClosureFields(a.Values, b.Values, seen)
	case *Object:
		b, ok := right.(*Object)
		return ok && equalUnionClosureFields(*a, *b, seen)
	case *Array:
		b, ok := right.(*Array)
		return ok && a.NonNullableElems == b.NonNullableElems && equalUnionClosureAttribute(a.ElemType, b.ElemType, seen)
	case *Map:
		b, ok := right.(*Map)
		return ok && equalUnionClosureAttribute(a.KeyType, b.KeyType, seen) && equalUnionClosureAttribute(a.ElemType, b.ElemType, seen)
	}
	return false
}

func equalUnionClosureAttribute(left, right *AttributeExpr, seen map[unionClosureContractPair]bool) bool {
	if left == right {
		return true
	}
	if left == nil || right == nil {
		return false
	}
	pair := unionClosureContractPair{left, right}
	if seen[pair] {
		return true
	}
	seen[pair] = true
	a, b := *left, *right
	for _, attribute := range []*AttributeExpr{&a, &b} {
		// Executed callbacks, copy provenance and derived caches are not part
		// of a declaration's contract. Compare graph edges separately below.
		attribute.DSLFunc, attribute.declarationDSL = nil, nil
		attribute.Type, attribute.AdditionalAttributes = nil, nil
		attribute.Bases, attribute.References, attribute.compositionBases = nil, nil, nil
		attribute.constraintParents, attribute.constraintLocal = nil, nil
		attribute.omittedMembers = nil
		attribute.valueOrigin, attribute.valueSourceOrigin, attribute.additionalCarrier = nil, nil, nil
	}
	if !reflect.DeepEqual(a, b) || !equalUnionClosureType(left.Type, right.Type, seen) ||
		!equalUnionClosureAttribute(left.AdditionalAttributes, right.AdditionalAttributes, seen) ||
		!equalUnionClosureAttribute(left.constraintLocal, right.constraintLocal, seen) ||
		!equalUnionClosureFields(left.omittedMembers, right.omittedMembers, seen) {
		return false
	}
	if len(left.constraintParents) != len(right.constraintParents) {
		return false
	}
	for index, parent := range left.constraintParents {
		if !equalUnionClosureAttribute(parent, right.constraintParents[index], seen) {
			return false
		}
	}
	for _, edges := range [][2][]DataType{
		{left.Bases, right.Bases}, {left.References, right.References},
		{left.compositionBases, right.compositionBases},
	} {
		if len(edges[0]) != len(edges[1]) {
			return false
		}
		for index, typ := range edges[0] {
			if !equalUnionClosureType(typ, edges[1][index], seen) {
				return false
			}
		}
	}
	return true
}

func equalUnionClosureFields(left, right []*NamedAttributeExpr, seen map[unionClosureContractPair]bool) bool {
	if len(left) != len(right) {
		return false
	}
	for index, field := range left {
		if field.Name != right[index].Name || !equalUnionClosureAttribute(field.Attribute, right[index].Attribute, seen) {
			return false
		}
	}
	return true
}
