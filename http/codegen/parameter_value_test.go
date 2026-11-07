package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

func TestParameterValueEncoding(t *testing.T) {
	scalar := &expr.AttributeExpr{Type: expr.String}
	integer := &expr.AttributeExpr{Type: expr.Int}
	record := &expr.UserTypeExpr{TypeName: "Record", AttributeExpr: &expr.AttributeExpr{
		Type:       &expr.Object{{Name: "r:R", Attribute: integer}, {Name: "g:G", Attribute: scalar}},
		Validation: &expr.ValidationExpr{Required: []string{"r"}},
	}}
	for _, tc := range []struct {
		name      string
		typ       expr.DataType
		pointer   bool
		fragments []string
	}{
		{"scalar", expr.String, false, []string{"Kind: loomhttp.ParamScalar", "Present: true", "Text = value"}},
		{"optional scalar", expr.String, true, []string{"if value != nil", "Text = *value"}},
		{"list", &expr.Array{ElemType: integer}, false, []string{"if value != nil", "make([]string, 0, len(value))", "strconv.Itoa"}},
		{"map", &expr.Map{KeyType: scalar, ElemType: integer}, false, []string{"loomhttp.ParamPairs", "sort.Slice", ".Key <"}},
		{"object", record, false, []string{"if value != nil", `Key: "R"`, "strconv.Itoa(value.R)", "if value.G != nil", `Key: "G"`, "*value.G"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			code := renderParameterValue(&expr.AttributeExpr{Type: tc.typ}, &expr.AttributeExpr{Type: tc.typ}, serviceContext("", codegen.NewNameScope()), "value", "paramValue", tc.pointer, "")
			for _, fragment := range tc.fragments {
				require.Contains(t, code, fragment)
			}
		})
	}
}
