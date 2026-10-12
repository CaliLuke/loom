package codegen

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestTypedCollectionDefaultRepresentation(t *testing.T) {
	scalar := func(kind expr.DataType) *expr.AttributeExpr {
		return &expr.AttributeExpr{Type: kind}
	}
	array := func(element *expr.AttributeExpr) *expr.AttributeExpr {
		return &expr.AttributeExpr{Type: &expr.Array{ElemType: element}}
	}
	count := &expr.UserTypeExpr{TypeName: "Count", AttributeExpr: scalar(expr.Int32)}
	located := &expr.UserTypeExpr{TypeName: "Located", AttributeExpr: &expr.AttributeExpr{Type: expr.Int32, Meta: expr.MetaExpr{"struct:pkg:path": {"types/counts"}}}}
	counts := &expr.UserTypeExpr{TypeName: "Counts", AttributeExpr: array(scalar(expr.Int32))}
	countsMap := &expr.UserTypeExpr{TypeName: "CountsMap", AttributeExpr: &expr.AttributeExpr{Type: &expr.Map{KeyType: scalar(expr.String), ElemType: scalar(expr.Int32)}}}
	cases := []struct {
		name   string
		target *expr.AttributeExpr
		value  any
		want   string
	}{
		{"int", array(scalar(expr.Int)), []int32{3}, "[]int{3}"},
		{"uint", array(scalar(expr.UInt)), []int{3}, "[]uint{3}"},
		{"uint64", array(scalar(expr.UInt64)), []int{3}, "[]uint64{3}"},
		{"float64", array(scalar(expr.Float64)), []int{3}, "[]float64{3}"},
		{"float32", array(scalar(expr.Float32)), []float64{1.5}, "[]float32{1.5}"},
		{"int64", array(scalar(expr.Int64)), []int{3}, "[]int64{3}"},
		{"uint32 nested", array(array(scalar(expr.UInt32))), [][]int{{3}}, "[][]uint32{[]uint32{3}}"},
		{"empty", array(scalar(expr.Int32)), []int{}, "[]int32{}"},
		{"alias elements", array(scalar(count)), []int{3}, "[]Count{3}"},
		{"empty alias elements", array(scalar(count)), []int{}, "[]Count{}"},
		{"nested map", array(&expr.AttributeExpr{Type: &expr.Map{KeyType: scalar(expr.String), ElemType: scalar(expr.Int32)}}), []map[string]int{{"k": 3}}, "[]map[string]int32{map[string]int32{\"k\": 3}}"},
		{"nested nil", array(array(scalar(expr.Int32))), [][]int{nil}, "[][]int32{[]int32(nil)}"},
		{"named outer array", scalar(counts), []int{3}, "Counts{3}"},
		{"named outer map", scalar(countsMap), map[string]int{"k": 3}, "CountsMap{\"k\": 3}"},
		{"located element", array(scalar(located)), []int{3}, "[]counts.Located{3}"},
		{"nested nil map", array(&expr.AttributeExpr{Type: &expr.Map{KeyType: scalar(expr.String), ElemType: scalar(expr.Int32)}}), []map[string]int{nil}, "[]map[string]int32{map[string]int32(nil)}"},
		{"raw JSON map key", &expr.AttributeExpr{Type: &expr.Map{KeyType: scalar(expr.Any), ElemType: scalar(expr.Any)}}, map[any]any{"k": "v"}, "map[any]loom.JSONValue{\"k\": loom.JSONValue(\"\\\"v\\\"\")}"},
		{"already typed", array(scalar(expr.Int32)), []int32{3}, "[]int32{3}"},
		{"map control", &expr.AttributeExpr{Type: &expr.Map{KeyType: scalar(expr.String), ElemType: scalar(expr.Int32)}}, map[string]int{"k": 3}, "map[string]int32{\"k\": 3}"},
		{"raw JSON control", array(scalar(expr.Any)), []any{"value"}, "[]loom.JSONValue{loom.JSONValue(\"\\\"value\\\"\")}"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := NewAttributeContext(false, false, true, "", NewNameScope())
			got, err := ctx.DefaultValue(tc.target, tc.value)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
		t.Run(tc.name+"/transform", func(t *testing.T) {
			scope := NewNameScope()
			sourceCtx := NewAttributeContext(true, false, false, "", scope)
			targetCtx := NewAttributeContext(false, false, true, "", scope)
			source := &expr.AttributeExpr{Type: &expr.Object{{Name: "values", Attribute: expr.DupAtt(tc.target)}}}
			targetMember := expr.DupAtt(tc.target)
			targetMember.DefaultValue = tc.value
			target := &expr.AttributeExpr{Type: &expr.Object{{Name: "values", Attribute: targetMember}}}
			code, _, err := GoTransform(source, target, "source", "target", sourceCtx, targetCtx, "", true)
			require.NoError(t, err)
			require.Contains(t, code, "target.Values = "+tc.want)
		})
	}
}

func TestTypedCollectionDefaultApplicationGates(t *testing.T) {
	for mask := range 32 {
		sourceDefaults := mask&1 != 0
		targetDefaults := mask&2 != 0
		pointer := mask&4 != 0
		required := mask&8 != 0
		jsonPresence := mask&16 != 0
		name := fmt.Sprintf("source_defaults_%t/target_defaults_%t/pointer_%t/required_%t/json_%t", sourceDefaults, targetDefaults, pointer, required, jsonPresence)
		t.Run(name, func(t *testing.T) {
			field := &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.Float32}}}
			source := &expr.AttributeExpr{Type: &expr.Object{{Name: "values", Attribute: field}}}
			if required {
				source.Validation = &expr.ValidationExpr{Required: []string{"values"}}
			}
			target := expr.DupAtt(source)
			target.Find("values").DefaultValue = []float64{1.5}
			scope := NewNameScope()
			sourceCtx := NewAttributeContext(false, false, sourceDefaults, "", scope)
			sourceCtx.JSONPresence = jsonPresence
			targetCtx := NewAttributeContext(pointer, false, targetDefaults, "", scope)
			code, _, err := GoTransform(source, target, "source", "target", sourceCtx, targetCtx, "", true)
			require.NoError(t, err)
			if targetDefaults && !pointer && !required {
				require.Contains(t, code, "target.Values = []float32{1.5}")
			} else {
				require.NotContains(t, code, "target.Values = []float32{1.5}")
			}
		})
	}
}
