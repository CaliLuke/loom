package codegen

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/expr"
)

// DefaultApplication describes a field's selected default and its application
// during conversion. A selected default alone does not establish source absence.
type DefaultApplication struct {
	// Value is the selected default, or nil when none is declared. Ordinary
	// conversion selects the target; admitted-source conversion may select the source.
	Value any
	// SourcePresence identifies the source wrapper. When Apply is true for
	// NativePresence, the source has nil-capable storage and only nil is absent.
	SourcePresence PresenceKind
	// Apply permits materializing Value only in the source's absent branch.
	// Supplied zeros, nonnil empty values and explicit null never take it.
	Apply bool

	source, target *expr.AttributeExpr
	admittedSource bool
	targetPointer  bool
	err            error
}

// DefaultForField resolves conversion default application from the actual source
// carrier and target materialization context. Required input must be admitted
// before conversion; defaults never satisfy its original presence requirement.
// Ordinary pointer targets retain absence for later selected-view validation.
// An admitted source default is already a supplied value before that conversion.
func (ta *TransformAttrs) DefaultForField(source, target *expr.MappedAttributeExpr, name string) DefaultApplication {
	attribute := source.Find(name)
	plan := DefaultApplication{
		Value:          target.GetDefault(name),
		SourcePresence: ta.SourceCtx.FieldPresence(source, name, attribute),
		target:         target.Find(name),
		targetPointer:  nativeObjectFieldPointer(target, name, target.Find(name), ta.TargetCtx),
	}
	absent := plan.SourcePresence != NativePresence ||
		ta.SourceCtx.IsPrimitivePointer(name, source.AttributeExpr) || defaultSourceUsesNil(attribute)
	plan.Apply = plan.Value != nil && ta.TargetCtx.UseDefault && !ta.TargetCtx.Pointer &&
		!source.IsRequired(name) && absent
	if ta.Options.AdmittedSource == "" || source.IsRequired(name) || !absent {
		return plan
	}
	value := source.GetDefault(name)
	if value == nil {
		return plan
	}
	selectedTarget := target.Find(name)
	if plan.Apply {
		equivalent, err := equivalentNativeScalarDefaults(attribute, selectedTarget, value, plan.Value)
		if err != nil {
			plan.err = err
		} else if equivalent {
			return plan
		}
	}
	plan.Value, plan.Apply, plan.admittedSource = value, true, true
	plan.source, plan.target = attribute, selectedTarget
	plan.targetPointer = nativeObjectFieldPointer(target, name, selectedTarget, ta.TargetCtx)
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

// equivalentNativeScalarDefaults proves only the native scalar subset for which
// the already selected target fallback performs the same absent-value operation.
// The caller has proved identical absence and applicable target storage first.
func equivalentNativeScalarDefaults(source, target *expr.AttributeExpr, sourceValue, targetValue any) (bool, error) {
	if expr.IsNullable(source) || expr.IsNullable(target) {
		return false, nil
	}
	if name, _ := GetMetaType(source); name != "" {
		return false, nil
	}
	if name, _ := GetMetaType(target); name != "" {
		return false, nil
	}
	kind := unalias(source.Type).Kind()
	if kind != unalias(target.Type).Kind() {
		return false, nil
	}
	switch kind {
	case expr.BooleanKind, expr.StringKind, expr.IntKind, expr.Int32Kind, expr.Int64Kind,
		expr.UIntKind, expr.UInt32Kind, expr.UInt64Kind:
	default:
		return false, nil
	}
	_, _, sourceResolved, err := resolveDefaultValue(source, sourceValue)
	if err != nil {
		return false, &DefaultValueError{Attribute: source, Err: err}
	}
	_, _, targetResolved, err := resolveDefaultValue(target, targetValue)
	if err != nil {
		return false, &DefaultValueError{Attribute: target, Err: err}
	}
	scalar, ok := sourceResolved.Scalar()
	other, otherOK := targetResolved.Scalar()
	if !ok || !otherOK {
		return false, fmt.Errorf("resolved native default has no scalar value")
	}
	switch scalar.(type) {
	case bool, string, int, int32, int64, uint, uint32, uint64:
		return scalar == other, nil
	default:
		return false, nil
	}
}
