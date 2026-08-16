package dsl

import (
	"testing"

	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
	"github.com/stretchr/testify/require"
)

func TestNullable(t *testing.T) {
	attribute := &expr.AttributeExpr{Type: expr.String}
	eval.Execute(func() {
		Nullable()
	}, attribute)
	require.True(t, attribute.Nullable)
}

func TestNullExample(t *testing.T) {
	attribute := &expr.AttributeExpr{Type: expr.String, Nullable: true}
	eval.Execute(func() {
		Example(Null())
	}, attribute)
	require.Len(t, attribute.UserExamples, 1)
	require.True(t, attribute.UserExamples[0].ExplicitNull)
	require.Nil(t, attribute.UserExamples[0].Value)
}

func TestNullExampleRejectsNonNullableAttribute(t *testing.T) {
	attribute := &expr.AttributeExpr{Type: expr.String}
	eval.Execute(func() {
		Example(Null())
	}, attribute)
	require.Empty(t, attribute.UserExamples)
}

func TestDefaultRejectsTypedNilCollections(t *testing.T) {
	tests := []struct {
		name  string
		value any
	}{
		{name: "slice", value: []string(nil)},
		{name: "map", value: map[string]string(nil)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			attribute := &expr.AttributeExpr{Type: expr.Any, Nullable: true}
			eval.Execute(func() {
				Default(test.value)
			}, attribute)
			require.Nil(t, attribute.DefaultValue)
		})
	}
}

func TestExampleRejectsTypedNilCollections(t *testing.T) {
	attribute := &expr.AttributeExpr{Type: expr.Any, Nullable: true}
	eval.Execute(func() {
		Example([]string(nil))
	}, attribute)
	require.Empty(t, attribute.UserExamples)
}
