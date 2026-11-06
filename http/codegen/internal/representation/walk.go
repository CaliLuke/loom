package representation

import "github.com/CaliLuke/loom/expr"

// WalkUserTypes visits each generated declaration identity reachable from att
// once, including recursive types and the types of the typed additional
// members of objects.
func WalkUserTypes(att *expr.AttributeExpr, cb func(expr.UserType), seen ...map[string]struct{}) {
	if att == nil || att.Type == expr.Empty {
		return
	}
	var s map[string]struct{}
	if len(seen) > 0 {
		s = seen[0]
	} else {
		s = make(map[string]struct{})
	}
	switch actual := att.Type.(type) {
	case *expr.Object:
		for _, nat := range *actual {
			WalkUserTypes(nat.Attribute, cb, s)
		}
		WalkUserTypes(att.AdditionalAttributes, cb, s)
	case *expr.Union:
		for _, nat := range actual.Values {
			WalkUserTypes(nat.Attribute, cb, s)
		}
	case *expr.Array:
		WalkUserTypes(actual.ElemType, cb, s)
	case *expr.Map:
		WalkUserTypes(actual.KeyType, cb, s)
		WalkUserTypes(actual.ElemType, cb, s)
	case expr.UserType:
		if _, ok := s[actual.Hash()]; ok {
			return
		}
		s[actual.Hash()] = struct{}{}
		cb(actual)
		WalkUserTypes(actual.Attribute(), cb, s)
	}
}
