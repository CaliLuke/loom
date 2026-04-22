package ir

import (
	"encoding/binary"
	"hash"

	"github.com/gohugoio/hashstructure"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/openapi"
)

func hashAttribute(att *expr.AttributeExpr, h hash.Hash64, seen map[string]*uint64) *uint64 {
	t := att.Type
	if cached, ok := seen[t.Hash()]; ok {
		return cached
	}
	var (
		res uint64
		ptr = &res
	)
	seen[t.Hash()] = ptr

	hv := hashValidation(att.Validation, h)
	switch t.Kind() {
	case expr.ObjectKind:
		o := expr.AsObject(t)
		for _, member := range *o {
			if !openapi.MustGenerate(member.Attribute.Meta) {
				continue
			}
			kh := hashString(member.Name, h)
			vh := hashAttribute(member.Attribute, h, seen)
			*ptr ^= orderedHash(kh, *vh, h)
		}
		if hv != 0 {
			*ptr = orderedHash(*ptr, hv, h)
		}
	case expr.ArrayKind:
		kh := hashString("[]", h)
		vh := hashAttribute(expr.AsArray(t).ElemType, h, seen)
		*ptr = orderedHash(kh, *vh, h)
		if hv != 0 {
			*ptr = orderedHash(*ptr, hv, h)
		}
	case expr.MapKind:
		m := expr.AsMap(t)
		kh := hashAttribute(m.KeyType, h, seen)
		vh := hashAttribute(m.ElemType, h, seen)
		*ptr = orderedHash(*kh, *vh, h)
		if hv != 0 {
			*ptr = orderedHash(*ptr, hv, h)
		}
	case expr.UserTypeKind:
		*ptr = *hashAttribute(t.(expr.UserType).Attribute(), h, seen)
	case expr.ResultTypeKind:
		rt := t.(*expr.ResultTypeExpr)
		*ptr = hashString(rt.Identifier, h)
		if view, ok := rt.Meta.Last(expr.ViewMetaKey); ok {
			*ptr = orderedHash(*ptr, hashString(view, h), h)
		}
	default:
		*ptr = hashString(t.Name(), h)
		if hv != 0 {
			*ptr = orderedHash(*ptr, hv, h)
		}
	}
	return ptr
}

func hashValidation(val *expr.ValidationExpr, h hash.Hash64) uint64 {
	if val == nil {
		return 0
	}
	res, err := hashstructure.Hash(val, &hashstructure.HashOptions{
		Hasher:          h,
		ZeroNil:         false,
		IgnoreZeroValue: true,
		SlicesAsSets:    true,
	})
	if err != nil {
		return 0
	}
	return res
}

func hashString(s string, h hash.Hash64) uint64 {
	h.Reset()
	if _, err := h.Write([]byte(s)); err != nil {
		panic(err)
	}
	return h.Sum64()
}

func orderedHash(aVal, bVal uint64, h hash.Hash64) uint64 {
	h.Reset()
	if err := binary.Write(h, binary.LittleEndian, aVal); err != nil {
		panic(err)
	}
	if err := binary.Write(h, binary.LittleEndian, bVal); err != nil {
		panic(err)
	}
	return h.Sum64()
}

func boolPtr(v bool) *bool {
	return &v
}
