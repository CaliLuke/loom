package representation

import (
	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/encodingmeta"
)

type (
	// Field is one physical body field and its resolved encoding contract.
	Field struct {
		// Layout owns pointer and wrapper choices used by the declaration emitter.
		Layout encodingmeta.FieldLayout
		// Encoding owns emitted tags and their corresponding JSON observation.
		Encoding FieldEncoding
	}

	// FieldEncoding pairs all codec tags with the shared resolved JSON contract.
	FieldEncoding struct {
		// Tags is the rendered Go struct tag.
		Tags string
		// JSON describes the tag that is actually present in Tags.
		JSON encodingmeta.JSONField
	}
)

// ResolveField selects the field layout once for declaration and projection.
// Callers restore registered named layouts before supplying occurrence flags.
func ResolveField(parent *expr.ObjectFields, name string, attribute *expr.AttributeExpr, pointer, useDefault, jsonPresence bool) Field {
	facts := parent.FieldFacts(name)
	wireOptional := !facts.Required || facts.HasDefault
	explicit := codegen.IsExplicitPresenceType(attribute)
	layout := encodingmeta.FieldLayout{
		WrapNullable: expr.AllowsNull(attribute) && !expr.IsNullable(attribute) && !expr.IsAny(attribute.Type),
	}
	layout.WrapOptional = !layout.WrapNullable && jsonPresence && wireOptional && !expr.AllowsNull(attribute)
	switch {
	case explicit, jsonPresence && wireOptional:
	case expr.IsPrimitive(attribute.Type):
		layout.Pointer = (pointer || facts.PrimitivePointer(useDefault)) && attribute.Type != expr.Bytes && !expr.IsAny(attribute.Type)
	case expr.IsObject(attribute.Type), expr.IsUnion(attribute.Type) && !facts.Required:
		layout.Pointer = true
	}
	optional := objectFieldOptional(facts, pointer, useDefault)
	normalize := wireOptional && (jsonPresence || explicit || expr.AllowsNull(attribute))
	layout.JSON = encodingmeta.JSONFieldPolicy{
		Storage:       attribute.JSONFieldStorage(layout.Pointer, explicit || layout.WrapNullable || layout.WrapOptional),
		Optional:      optional || normalize,
		Presence:      normalize,
		ForceOmission: isJSONRPCID(attribute),
	}
	return Field{Layout: layout, Encoding: FieldTags(attribute, facts.ElementName, optional, layout.JSON)}
}

func objectFieldOptional(facts expr.ObjectFieldFacts, pointer, useDefault bool) bool {
	switch {
	case pointer:
		return true
	case useDefault:
		return !facts.Required && !facts.HasDefault
	default:
		return !facts.Required
	}
}
