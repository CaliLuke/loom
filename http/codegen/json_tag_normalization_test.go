package codegen

import (
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/representation"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
	"github.com/stretchr/testify/require"
)

func TestJSONAuthoredTagParity(t *testing.T) {
	for _, tag := range []string{"wire", "wire,omitempty", "wire,omitzero"} {
		t.Run(tag, func(t *testing.T) {
			field := &expr.AttributeExpr{Type: expr.String, Meta: expr.MetaExpr{"struct:tag:json": {tag}}}
			parent := &expr.AttributeExpr{Type: &expr.Object{{Name: "value", Attribute: field}}}
			serviceTag := codegen.AttributeTagsWithName(parent, "value", field)
			bodyTag := representation.AttributeTags(field, "value", true, false)
			require.Contains(t, serviceTag, `json:"`+tag+`"`, "documented full-tag control")
			require.Contains(t, bodyTag, `json:"`+tag+`"`, "ordinary encoder must honor the same authored full tag")
		})
	}
}

func TestJSONAuthoredTagNormalizationControls(t *testing.T) {
	for _, tc := range []struct {
		name     string
		meta     expr.MetaExpr
		optional bool
		zero     bool
		want     string
	}{
		{"automatic optional", nil, true, false, "value,omitempty"},
		{"automatic required", nil, false, false, "value"},
		{"name only optional", expr.MetaExpr{"struct:tag:json:name": {"wire"}}, true, false, "wire,omitempty"},
		{"full retains empty", expr.MetaExpr{"struct:tag:json": {"wire"}}, true, false, "wire"},
		{"full zero omission", expr.MetaExpr{"struct:tag:json": {"wire,omitzero"}}, true, false, "wire,omitzero"},
		{"full empty omission", expr.MetaExpr{"struct:tag:json": {"wire,omitempty"}}, true, false, "wire,omitempty"},
		{"full precedence", expr.MetaExpr{"struct:tag:json": {"wire"}, "struct:tag:json:name": {"other"}}, true, false, "wire"},
		{"full extra option", expr.MetaExpr{"struct:tag:json": {"wire,string,omitzero"}}, true, false, "wire,string,omitzero"},
		{"full explicit empty", expr.MetaExpr{"struct:tag:json": {""}}, true, false, ""},
		{"nullable normalizes", expr.MetaExpr{"struct:tag:json": {"wire,omitempty"}}, true, true, "wire,omitzero"},
		{"full skip", expr.MetaExpr{"struct:tag:json": {"-"}}, true, false, "-"},
		{"name only skip", expr.MetaExpr{"struct:tag:json:name": {"-"}}, true, false, "-"},
		{"custom non JSON only", expr.MetaExpr{"struct:tag:xml": {"wire"}}, true, false, ""},
		{"custom plus JSON name", expr.MetaExpr{"struct:tag:xml": {"wire"}, "struct:tag:json:name": {"wire"}}, true, false, "wire,omitempty"},
		{"RPC ID remains omitted", expr.MetaExpr{"jsonrpc:id": {}}, false, false, "value,omitempty"},
		{"RPC ID full tag remains omitted", expr.MetaExpr{"jsonrpc:id": {}, "struct:tag:json": {"wire"}}, false, false, "wire,omitempty"},
		{"nullable default retains zero omission", nil, false, true, "value,omitzero"},
		{"required full zero omission", expr.MetaExpr{"struct:tag:json": {"wire,omitzero"}}, false, false, "wire,omitzero"},
		{"full literal dash", expr.MetaExpr{"struct:tag:json": {"-,"}}, true, false, "-,"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			field := &expr.AttributeExpr{Type: expr.String, Meta: tc.meta}
			rendered := representation.AttributeTags(field, "value", tc.optional, tc.zero)
			raw, err := strconv.Unquote(strings.TrimSpace(rendered))
			require.NoError(t, err)
			require.Equal(t, tc.want, reflect.StructTag(raw).Get("json"))
		})
	}
}

func TestJSONFullTagValueProjection(t *testing.T) {
	for _, tc := range []struct {
		name  string
		typ   expr.DataType
		empty any
		wire  string
	}{
		{"string", expr.String, "", `{"value":""}`},
		{"bytes", expr.Bytes, []byte{}, `{"value":""}`},
		{"array", &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}}, []string{}, `{"value":[]}`},
		{"map", &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: &expr.AttributeExpr{Type: expr.String}}, map[string]string{}, `{"value":{}}`},
		{"object", &expr.Object{}, map[string]any{}, `{"value":{}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			attribute := &expr.AttributeExpr{Type: &expr.Object{{Name: "value", Attribute: &expr.AttributeExpr{Type: tc.typ, Meta: expr.MetaExpr{"struct:tag:json": {"value,omitzero"}}}}}}
			context := expr.NewValueContext()
			occurrence, err := context.NewOccurrence(attribute)
			require.NoError(t, err)
			resolved := context.Resolve(occurrence, context.SupplyValue(expr.ValueInput{Raw: map[string]any{"value": tc.empty}}), expr.ValueRoleExample)
			require.Equal(t, expr.ValueResolved, resolved.Outcome())
			target := &transportir.ValueTarget{Source: &service.ValueData{Context: context, Occurrence: occurrence, Example: resolved}, Codec: expr.ValueCodecJSON}
			planned := representation.BuildValuePlan(target, attribute, httpContext(codegen.NewNameScope(), false, true), expr.ValuePlanRuntime)
			require.NoError(t, planned.Error)
			projected := context.ProjectJSON(resolved, planned.Plan)
			require.Equal(t, expr.ProjectionEmitted, projected.Outcome(), projected.Diagnostics())
			wire, present := projected.JSON()
			require.True(t, present)
			require.JSONEq(t, tc.wire, string(wire))
		})
	}
}
