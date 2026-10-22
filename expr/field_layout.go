package expr

import "github.com/CaliLuke/loom/internal/encodingmeta"

// ServiceFieldLayout resolves the canonical service declaration's storage.
// Composition uses the same owner after inherited defaults and requiredness have
// been finalized. Type names remain the code generator's responsibility.
func ServiceFieldLayout(parent *AttributeExpr, name string, attribute *AttributeExpr, pointer, useDefault bool) encodingmeta.FieldLayout {
	return serviceFieldLayout(attribute, parent.IsRequired(name), pointer, func() bool {
		return parent.IsPrimitivePointer(name, useDefault)
	})
}

// ServiceLayout resolves native declaration storage from captured field facts.
// The attribute supplies current explicit type and codec decisions; callers must
// pair facts and attributes from the same stable object analysis.
func (f ObjectFieldFacts) ServiceLayout(attribute *AttributeExpr, pointer, useDefault bool) encodingmeta.FieldLayout {
	return serviceFieldLayout(attribute, f.Required, pointer, func() bool {
		return f.PrimitivePointer(useDefault)
	})
}

// JSONFieldStorage classifies an actual declaration's pointer and presence
// decisions. It never infers absence from application IsZero methods. Authored
// replacements retain their external codec boundary before builtin kind checks.
func (a *AttributeExpr) JSONFieldStorage(pointer, presence bool) encodingmeta.JSONStorage {
	_, primitive := a.Type.(Primitive)
	if encodingmeta.Replacement(a.Meta) != "" && (primitive || encodingmeta.ExplicitPresence(a.Meta, IsNullable(a), a.Type == Any)) {
		return encodingmeta.JSONCustom
	}
	if presence {
		return encodingmeta.JSONPresence
	}
	if pointer || IsArray(a.Type) || IsMap(a.Type) || unalias(a.Type) == Bytes {
		return encodingmeta.JSONNilable
	}
	return encodingmeta.JSONNative
}

func serviceFieldLayout(attribute *AttributeExpr, required, pointer bool, primitivePointer func() bool) encodingmeta.FieldLayout {
	presence := encodingmeta.ExplicitPresence(attribute.Meta, IsNullable(attribute), attribute.Type == Any)
	layout := encodingmeta.FieldLayout{
		WrapNullable: AllowsNull(attribute) && !IsNullable(attribute) && !IsAny(attribute.Type),
	}
	layout.Pointer = !presence && (IsObject(attribute.Type) ||
		(IsUnion(attribute.Type) && !required) ||
		primitivePointer() ||
		(pointer && IsPrimitive(attribute.Type) && !IsAny(attribute.Type) && attribute.Type.Kind() != BytesKind))
	layout.JSON = encodingmeta.JSONFieldPolicy{
		Storage:  attribute.JSONFieldStorage(layout.Pointer, presence || layout.WrapNullable),
		Optional: !required,
		Presence: presence || AllowsNull(attribute),
	}
	return layout
}
