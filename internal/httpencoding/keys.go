package httpencoding

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
)

type (
	// KeySet is the set of wire keys owned by one endpoint binding.
	KeySet struct {
		// Literals contains exact decoded key names.
		Literals []string
		// Prefixes contains deepObject names, each owning name[key].
		Prefixes []string
		// Residual claims every key not owned by a sibling binding.
		Residual bool
	}

	// Binding is one independently authored parameter or synthetic transport claim.
	// Name is a diagnostic label, not its identity; equal names remain distinct.
	Binding struct {
		// Name describes this binding in validation diagnostics.
		Name string
		// Keys identifies the wire namespace owned by this binding.
		Keys KeySet
	}
)

// CheckNamespaces rejects overlapping claims between endpoint bindings.
// It retains every binding, including those with equal diagnostic names.
func CheckNamespaces(loc Location, bindings []Binding) error {
	ordered := slices.Clone(bindings)
	slices.SortStableFunc(ordered, func(left, right Binding) int {
		return cmp.Compare(left.Name, right.Name)
	})
	for i, left := range ordered {
		for _, right := range ordered[i+1:] {
			if loc == WholeQuery || keysOverlap(left.Keys, right.Keys) {
				return fmt.Errorf("bindings %q and %q claim overlapping parameter namespaces", left.Name, right.Name)
			}
		}
	}
	return nil
}

func bindingKeys(loc Location, name string, shape BranchShape, codec Codec) (KeySet, error) {
	keys := KeySet{Literals: []string{name}}
	if loc == Path || loc == Header {
		return keys, nil
	}
	if codec.Style == DeepObject {
		if strings.ContainsAny(name, "[]") {
			return KeySet{}, fmt.Errorf("deepObject name cannot contain brackets")
		}
		if shape.Shape == ScalarMap {
			return KeySet{Prefixes: []string{name}}, nil
		}
		keys = KeySet{}
		for _, property := range shape.Properties {
			if strings.ContainsAny(property.WireName, "[]") {
				return KeySet{}, fmt.Errorf("deepObject property name cannot contain brackets")
			}
			keys.Literals = append(keys.Literals, name+"["+property.WireName+"]")
		}
		return keys, nil
	}
	if codec.Explode && (codec.Style == Form || codec.Style == CookieStyle) {
		switch shape.Shape {
		case ScalarMap:
			return KeySet{Residual: true}, nil
		case Object:
			keys = KeySet{}
			for _, property := range shape.Properties {
				keys.Literals = append(keys.Literals, property.WireName)
			}
		}
	}
	return keys, nil
}

func keysOverlap(left, right KeySet) bool {
	if left.Residual && right.Residual {
		return true
	}
	for _, literal := range left.Literals {
		if slices.Contains(right.Literals, literal) || prefixOwns(right.Prefixes, literal) {
			return true
		}
	}
	for _, literal := range right.Literals {
		if prefixOwns(left.Prefixes, literal) {
			return true
		}
	}
	for _, prefix := range left.Prefixes {
		if slices.Contains(right.Prefixes, prefix) {
			return true
		}
	}
	return false
}

func prefixOwns(prefixes []string, name string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(name, prefix+"[") && strings.HasSuffix(name, "]") {
			return true
		}
	}
	return false
}

func mergeKeys(left, right KeySet) KeySet {
	for _, name := range right.Literals {
		if !slices.Contains(left.Literals, name) {
			left.Literals = append(left.Literals, name)
		}
	}
	for _, prefix := range right.Prefixes {
		if !slices.Contains(left.Prefixes, prefix) {
			left.Prefixes = append(left.Prefixes, prefix)
		}
	}
	left.Residual = left.Residual || right.Residual
	return left
}
