package ir

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/representation"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

func TestInitExamplesRequiresPreparedTarget(t *testing.T) {
	attribute := &expr.AttributeExpr{
		Type:         expr.String,
		UserExamples: []*expr.ExampleExpr{{Value: "must-not-fallback"}},
	}

	require.Panics(t, func() {
		initExamples(&MediaType{}, attribute, false, nil)
	})
	require.Panics(t, func() {
		initExamples(&MediaType{}, attribute, false,
			&transportir.ValueTarget{})
	})
}

func TestInitExamplesUsesOnlyPreparedTarget(t *testing.T) {
	attribute := &expr.AttributeExpr{
		Type:         expr.String,
		UserExamples: []*expr.ExampleExpr{{Value: "prepared"}},
	}
	prepared, err := representation.PrepareStandaloneExamples(attribute, expr.NewRandom("prepared"))
	require.NoError(t, err)
	target := &MediaType{}

	initExamples(target, attribute, false, prepared)

	require.Equal(t, "prepared", target.Example)
}

func TestInitExamplesPreservesPreparedOmission(t *testing.T) {
	attribute := &expr.AttributeExpr{Type: expr.String}
	target := &MediaType{}

	initExamples(target, attribute, false,
		&transportir.ValueTarget{ExamplesPrepared: true})

	require.Nil(t, target.Example)
	require.Empty(t, target.Examples)
}

func TestInitExamplesPreservesSingletonComponentMetadata(t *testing.T) {
	tests := []struct {
		name           string
		componentName  string
		wantStructured bool
	}{
		{name: "unnamed"},
		{name: "whitespace-only name", componentName: " \t "},
		{name: "explicit name", componentName: "  SingletonSearchExample  ", wantStructured: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			meta := expr.MetaExpr{
				"openapi:example:summary": []string{"Authored singleton summary"},
			}
			if test.componentName != "" {
				meta["openapi:component:example"] = []string{test.componentName}
			}
			attribute := &expr.AttributeExpr{
				Type: expr.String,
				UserExamples: []*expr.ExampleExpr{{
					Summary:     "primary",
					Description: "Retained singleton description.",
					Meta:        meta,
					Value:       "retained singleton value",
				}},
			}
			prepared, err := representation.PrepareStandaloneExamples(attribute, expr.NewRandom("singleton-component"))
			require.NoError(t, err)
			target := &MediaType{}

			initExamples(target, attribute, false, prepared)

			if !test.wantStructured {
				require.Equal(t, "retained singleton value", target.Example)
				require.Empty(t, target.Examples)
				return
			}
			require.Nil(t, target.Example)
			require.Contains(t, target.Examples, "primary")
			example := target.Examples["primary"].Value
			require.NotNil(t, example)
			require.Equal(t, "Authored singleton summary", example.Summary)
			require.Equal(t, "Retained singleton description.", example.Description)
			require.Equal(t, test.componentName, example.ComponentName)
			require.Equal(t, "retained singleton value", example.Value)
		})
	}
}
