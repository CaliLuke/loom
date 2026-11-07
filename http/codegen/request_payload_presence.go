package codegen

import (
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
	"github.com/CaliLuke/loom/internal/encodingmeta"
)

// requestPayloadPresence records absence from the resolved service carrier,
// before a transport conversion can dereference it. Nil outer envelopes cannot
// supply selected member bodies or separately mapped bindings.
func requestPayloadPresence(request *transportir.Request, payload *expr.AttributeExpr, ref string, credentials bool) (string, bool) {
	if request.Body == nil || request.Body.Type == expr.Empty || request.SkipBodyEncode {
		return "", false
	}
	absent := requestValueAbsence(payload, "p", strings.HasPrefix(ref, "*"))
	optional := request.BodyOrigin == "" && !request.MustHaveBody && !credentials &&
		len(request.PathParams)+len(request.QueryParams)+len(request.Headers)+len(request.Cookies) == 0
	return absent, optional
}

// requestMemberAbsence uses the service declaration's field storage, before
// selected-body normalization changes the transport carrier.
func requestMemberAbsence(payload *expr.AttributeExpr, field selectedServiceField) string {
	if field.Name == "" {
		return ""
	}
	key, attribute := payload.FindAttribute(field.Name)
	// Required Any members have no absent state: their nil JSON value is
	// encoded as null. Whole-body JSON values keep the root presence contract.
	if payload.IsRequired(key) && expr.IsAny(attribute.Type) && !expr.IsNullable(attribute) {
		return ""
	}
	layout := expr.ServiceFieldLayout(payload, key, attribute, false, true)
	return requestValueAbsence(attribute, "p."+field.GoName, layout.Pointer)
}

// requestValueAbsence observes only framework-owned presence operations.
// Empty native union kinds are absence even when the carrier is nonnil;
// an explicitly selected null branch remains present.
func requestValueAbsence(attribute *expr.AttributeExpr, value string, pointer bool) string {
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
