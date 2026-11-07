package codegen

import (
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/encodingmeta"
)

// sourceMemberAbsence uses the actual declaration context and field storage, before
// selected-body normalization changes the transport carrier.
func sourceMemberAbsence(payload *expr.AttributeExpr, field selectedServiceField, context *codegen.AttributeContext, source string) string {
	if field.Name == "" {
		return ""
	}
	key, attribute := payload.FindAttribute(field.Name)
	// Required Any members have no absent state: their nil JSON value is
	// encoded as null. Whole-body JSON values keep the root presence contract.
	if payload.IsRequired(key) && expr.IsAny(attribute.Type) && !expr.IsNullable(attribute) {
		return ""
	}
	layout := expr.ServiceFieldLayout(payload, key, attribute, context.Pointer, context.UseDefault)
	return sourceValueAbsence(attribute, source+"."+field.GoName, layout.Pointer)
}

// sourceValueAbsence observes only framework-owned presence operations.
// Empty native union kinds are absence even when the carrier is nonnil;
// an explicitly selected null branch remains present.
func sourceValueAbsence(attribute *expr.AttributeExpr, value string, pointer bool) string {
	replacement, _ := codegen.GetMetaType(attribute)
	nilable := pointer || strings.HasPrefix(replacement, "*")
	var absent string
	switch {
	case replacement != "" && !strings.HasPrefix(replacement, "loom.Nullable["):
		// Opaque value codecs own their presence; no wrapper methods are implied.
		if nilable {
			return value + " == nil"
		}
		return ""
	case expr.IsNullable(attribute) || codegen.IsExplicitPresenceType(attribute) && !expr.IsAny(attribute.Type):
		absent = "!" + value + ".Present()"
	case expr.IsUnion(attribute.Type):
		absent = value + `.Kind() == ""`
	case expr.IsAny(attribute.Type), attribute.JSONFieldStorage(pointer, false) == encodingmeta.JSONNilable:
		nilable = true
	}
	if nilable {
		if absent == "" {
			return value + " == nil"
		}
		return value + " == nil || " + absent
	}
	return absent
}
