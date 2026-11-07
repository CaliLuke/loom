package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

func TestParameterValueAssignment(t *testing.T) {
	text := &expr.AttributeExpr{Type: expr.String}
	integer := &expr.AttributeExpr{Type: expr.Int32}
	record := &expr.UserTypeExpr{TypeName: "Record", AttributeExpr: &expr.AttributeExpr{
		Type:       &expr.Object{{Name: "r:R", Attribute: integer}, {Name: "g:G", Attribute: text}},
		Validation: &expr.ValidationExpr{Required: []string{"r"}},
	}}
	for _, tc := range []struct {
		name      string
		typ       expr.DataType
		pointer   bool
		fragments []string
	}{
		{"scalar", expr.String, false, []string{"value = string(raw.Text)"}},
		{"optional", expr.Int32, true, []string{"strconv.ParseInt(raw.Text, 10, 32)", "value = &"}},
		{"list", &expr.Array{ElemType: integer}, false, []string{"make([]int32, len(raw.Items))", "range raw.Items", "strconv.ParseInt"}},
		{"map", &expr.Map{KeyType: integer, ElemType: text}, false, []string{"make(map[int32]string, len(raw.Pairs))", "duplicate converted key", "strconv.ParseInt"}},
		{"object", record, false, []string{"new(svc.Record)", `case "R":`, `case "G":`, "missing property R", "unknown property", "value.G = &"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			att := &expr.AttributeExpr{Type: tc.typ}
			ctx := codegen.NewAttributeContext(false, false, true, "svc", codegen.NewNameScope())
			got := renderParameterAssignment(att, att, ctx, "raw", "value", "filter", tc.pointer)
			for _, fragment := range tc.fragments {
				require.Contains(t, got, fragment)
			}
			require.NotContains(t, got, "err2.Error()")
		})
	}
}

func TestParameterAssignmentSharedJSONLayout(t *testing.T) {
	for _, tc := range []struct {
		name string
		typ  expr.DataType
		want string
	}{
		{"list", &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}}, "value[parameterIndex] = loom.OptionalValue(parameterNative)"},
		{"map", &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: &expr.AttributeExpr{Type: expr.String}}, "value[parameterKey] = loom.OptionalValue(parameterItem)"},
		{"object", &expr.Object{{Name: "text", Attribute: &expr.AttributeExpr{Type: expr.String}}}, "value.Text = loom.OptionalValue(parameterNative)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			named := &expr.UserTypeExpr{TypeName: "Shared", AttributeExpr: &expr.AttributeExpr{Type: tc.typ}}
			att := &expr.AttributeExpr{Type: named}
			ctx := httpContext(codegen.NewNameScope(), true, true)
			ctx.JSONPresence, ctx.CollectionElementPresence = false, false
			ctx.JSONPresenceTypes = map[string]bool{named.Hash(): true}
			ctx.PresencePointerTypes = map[string]bool{named.Hash(): true}
			got := renderParameterAssignment(att, att, ctx, "raw", "value", "field", false)
			require.Contains(t, got, tc.want)
			require.False(t, ctx.JSONPresence, "the occurrence context remains unchanged")
		})
	}
}
