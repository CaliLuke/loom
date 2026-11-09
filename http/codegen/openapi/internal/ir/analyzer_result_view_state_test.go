package ir

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestResultTypeViewConsumesSelection(t *testing.T) {
	for _, tc := range []struct {
		name        string
		declaration string
		occurrence  string
		applied     bool
		want        string
	}{
		{name: "no selection"},
		{name: "declaration", declaration: "tiny", want: "tiny"},
		{name: "occurrence overrides declaration", declaration: "tiny", occurrence: "default", want: "default"},
		{name: "consumed declaration", declaration: "tiny", applied: true},
		{name: "consumed occurrence", applied: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := &expr.ResultTypeExpr{
				UserTypeExpr: &expr.UserTypeExpr{
					AttributeExpr: &expr.AttributeExpr{Meta: make(expr.MetaExpr)},
				},
			}
			attribute := &expr.AttributeExpr{Type: result, Meta: make(expr.MetaExpr)}
			if tc.declaration != "" {
				result.Meta[expr.ViewMetaKey] = []string{tc.declaration}
			}
			if tc.occurrence != "" {
				attribute.Meta[expr.ViewMetaKey] = []string{tc.occurrence}
			}
			if tc.applied {
				attribute.Meta[projectedResultMetaKey] = []string{"true"}
			}
			view, selected := resultTypeView(attribute, result)
			require.Equal(t, tc.want, view)
			require.Equal(t, tc.want != "", selected)
		})
	}
}

func TestAnalyzerAcceptsAlreadySelectedDeclarationView(t *testing.T) {
	fields := &expr.Object{{Name: "id", Attribute: &expr.AttributeExpr{Type: expr.String}}}
	result := &expr.ResultTypeExpr{
		Identifier: "application/vnd.record; view=tiny",
		UserTypeExpr: &expr.UserTypeExpr{
			TypeName: "RecordTiny",
			UID:      "record-tiny",
			AttributeExpr: &expr.AttributeExpr{
				Type: fields,
				Meta: expr.MetaExpr{expr.ViewMetaKey: {"tiny"}},
			},
		},
		Views: []*expr.ViewExpr{{Name: expr.DefaultView, AttributeExpr: &expr.AttributeExpr{Type: fields}}},
	}
	result.Views[0].Parent = result
	analyzer := NewAnalyzer(expr.NewRandom("selected-view"), false)
	schema := analyzer.AnalyzeSchema(&expr.AttributeExpr{Type: result})
	require.NotEmpty(t, schema.Ref)
	require.Len(t, analyzer.Components(), 1)
	for _, component := range analyzer.Components() {
		require.Contains(t, component.Properties, "id")
		require.Len(t, component.Properties, 1)
	}
}
