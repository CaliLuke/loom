package codegen

import (
	"strings"

	"github.com/CaliLuke/loom/expr"
)

// DefaultApplication describes a field's selected default and its application
// during conversion. A selected default alone does not establish source absence.
type DefaultApplication struct {
	// Value is the target field's selected default, or nil when none is declared.
	Value any
	// SourcePresence identifies the source wrapper. When Apply is true for
	// NativePresence, the source has nil-capable storage and only nil is absent.
	SourcePresence PresenceKind
	// Apply permits materializing Value only in the source's absent branch.
	// Supplied zeros, nonnil empty values and explicit null never take it.
	Apply bool
}

// DefaultForField resolves conversion default application from the actual source
// carrier and target materialization context. Required input must be admitted
// before conversion; defaults never satisfy its original presence requirement.
// Pointer targets retain absence for later selected-view validation.
func (ta *TransformAttrs) DefaultForField(source, target *expr.MappedAttributeExpr, name string) DefaultApplication {
	attribute := source.Find(name)
	plan := DefaultApplication{
		Value:          target.GetDefault(name),
		SourcePresence: ta.SourceCtx.FieldPresence(source, name, attribute),
	}
	if plan.Value == nil || !ta.TargetCtx.UseDefault || ta.TargetCtx.Pointer || source.IsRequired(name) {
		return plan
	}
	plan.Apply = plan.SourcePresence != NativePresence ||
		ta.SourceCtx.IsPrimitivePointer(name, source.AttributeExpr) || defaultSourceUsesNil(attribute)
	return plan
}

// defaultSourceUsesNil classifies native physical storage after pointer and
// wrapper selection. Metadata overrides precede semantic kinds; an opaque custom
// type is a supplied value unless its nil-capable representation is known.
func defaultSourceUsesNil(attribute *expr.AttributeExpr) bool {
	if typeName, _ := GetMetaType(attribute); typeName != "" {
		return strings.HasPrefix(typeName, "[]") || strings.HasPrefix(typeName, "map[") ||
			strings.HasPrefix(typeName, "*") || typeName == "jsontext.Value"
	}
	switch unalias(attribute.Type).Kind() {
	case expr.BytesKind, expr.AnyKind, expr.ObjectKind, expr.ArrayKind, expr.MapKind, expr.UnionKind:
		return true
	default:
		return false
	}
}
