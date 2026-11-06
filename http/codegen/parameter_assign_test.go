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
