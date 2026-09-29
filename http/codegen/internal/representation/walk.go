package representation

import "github.com/CaliLuke/loom/expr"

// WalkUserTypes visits each generated declaration identity once, including recursive types.
func WalkUserTypes(dt expr.DataType, cb func(expr.UserType), seen ...map[string]struct{}) {
	if dt == expr.Empty {
		return
	}
	var s map[string]struct{}
	if len(seen) > 0 {
		s = seen[0]
	} else {
		s = make(map[string]struct{})
	}
	switch actual := dt.(type) {
	case *expr.Object:
		for _, nat := range *actual {
			WalkUserTypes(nat.Attribute.Type, cb, s)
		}
	case *expr.Union:
		for _, nat := range actual.Values {
			WalkUserTypes(nat.Attribute.Type, cb, s)
		}
	case *expr.Array:
		WalkUserTypes(actual.ElemType.Type, cb, s)
	case *expr.Map:
		WalkUserTypes(actual.KeyType.Type, cb, s)
		WalkUserTypes(actual.ElemType.Type, cb, s)
	case expr.UserType:
		if _, ok := s[actual.Hash()]; ok {
			return
		}
		s[actual.Hash()] = struct{}{}
		cb(actual)
		WalkUserTypes(actual.Attribute().Type, cb, s)
	}
}
