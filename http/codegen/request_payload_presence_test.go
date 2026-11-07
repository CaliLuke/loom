package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

func TestRequestPayloadPresenceDoesNotInventBody(t *testing.T) {
	payload := &expr.AttributeExpr{Type: &expr.Object{{Name: "query", Attribute: &expr.AttributeExpr{Type: expr.String}}}}
	request := &transportir.Request{Body: &expr.AttributeExpr{Type: expr.Empty}, Payload: payload}
	absent, optional := requestPayloadPresence(request, payload, "*Payload", false)
	require.Empty(t, absent, "a query-only payload does not own an HTTP body occurrence")
	require.False(t, optional)
}

func TestRequestPayloadPresenceUsesResolvedCarrier(t *testing.T) {
	for _, tc := range []struct {
		name      string
		attribute *expr.AttributeExpr
		ref, want string
	}{
		{"native union pointer", &expr.AttributeExpr{Type: &expr.Union{TypeName: "Choice"}}, "*Choice", `p == nil || p.Kind() == ""`},
		{"native union value", &expr.AttributeExpr{Type: &expr.Union{TypeName: "Choice"}}, "Choice", `p.Kind() == ""`},
		{"nullable union", &expr.AttributeExpr{Type: &expr.Union{TypeName: "Choice"}, Nullable: true}, "loom.Nullable[Choice]", "!p.Present()"},
		{"opaque union", &expr.AttributeExpr{Type: &expr.Union{TypeName: "Choice"}, Meta: expr.MetaExpr{"struct:field:type": {"custom.Value"}}}, "custom.Value", ""},
		{"opaque union pointer", &expr.AttributeExpr{Type: &expr.Union{TypeName: "Choice"}, Meta: expr.MetaExpr{"struct:field:type": {"*custom.Value"}}}, "*custom.Value", "p == nil"},
		{"nilable alias", &expr.AttributeExpr{Type: &expr.UserTypeExpr{TypeName: "Bytes", AttributeExpr: &expr.AttributeExpr{Type: expr.Bytes}}}, "Bytes", "p == nil"},
		{"raw JSON", &expr.AttributeExpr{Type: expr.Any}, "loom.JSONValue", "p == nil"},
		{"nullable", &expr.AttributeExpr{Type: expr.String, Nullable: true}, "loom.Nullable[string]", "!p.Present()"},
		{"custom JSON", &expr.AttributeExpr{Type: expr.Any, Meta: expr.MetaExpr{"struct:field:type": {"custom.Value"}}}, "custom.Value", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := &transportir.Request{Body: tc.attribute, Payload: tc.attribute, MustHaveBody: true}
			absent, optional := requestPayloadPresence(request, tc.attribute, tc.ref, false)
			require.Equal(t, tc.want, absent)
			require.False(t, optional)
		})
	}
}

func TestRequestMemberAbsenceUsesServiceField(t *testing.T) {
	union := &expr.Union{TypeName: "Choice"}
	for _, tc := range []struct {
		name      string
		attribute *expr.AttributeExpr
		required  bool
		want      string
	}{
		{"required Any", &expr.AttributeExpr{Type: expr.Any}, true, ""},
		{"optional Any", &expr.AttributeExpr{Type: expr.Any}, false, "p.Choice == nil"},
		{"optional native", &expr.AttributeExpr{Type: union}, false, `p.Choice == nil || p.Choice.Kind() == ""`},
		{"required native", &expr.AttributeExpr{Type: union}, true, `p.Choice.Kind() == ""`},
		{"optional nullable", &expr.AttributeExpr{Type: union, Nullable: true}, false, `!p.Choice.Present()`},
		{"required nullable", &expr.AttributeExpr{Type: union, Nullable: true}, true, `!p.Choice.Present()`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			payload := &expr.AttributeExpr{Type: &expr.Object{{Name: "choice", Attribute: tc.attribute}}}
			if tc.required {
				payload.Validation = &expr.ValidationExpr{Required: []string{"choice"}}
			}
			field := selectedServiceField{Name: "choice", GoName: "Choice"}
			require.Equal(t, tc.want, sourceMemberAbsence(payload, field, serviceContext("", nil), "p"))
		})
	}
}
