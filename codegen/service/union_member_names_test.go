package service

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

func TestUnionMemberNamesAvoidGeneratedMethods(t *testing.T) {
	for _, builder := range []struct {
		name  string
		build func(expr.DataType, *codegen.NameScope, *codegen.Location) *UnionTypeData
	}{
		{"service", buildUnionTypeData},
		{"view", buildViewUnionTypeData},
	} {
		t.Run(builder.name, func(t *testing.T) {
			union := &expr.Union{TypeName: "Choice"}
			for _, name := range []string{"Kind", "Kind2", "Validate", "MarshalJSON", "UnmarshalJSON", "MarshalFormValues", "UnmarshalFormValues", "Text", "AsText", "SetText"} {
				union.Values = append(union.Values, &expr.NamedAttributeExpr{Name: name, Attribute: &expr.AttributeExpr{Type: expr.String}})
			}
			data := builder.build(union, codegen.NewNameScope(), nil)
			for i, field := range data.Fields {
				if field.Name == "Kind2" || field.Name == "Text" {
					require.Equal(t, field.Name, field.FieldName)
				} else {
					require.NotEqual(t, field.Name, field.FieldName, "branch %d collides with a generated method", i)
				}
			}
		})
	}
}
