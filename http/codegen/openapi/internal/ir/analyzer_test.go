package ir

import (
	"testing"

	"github.com/stretchr/testify/require"

	"goa.design/goa/v3/expr"
)

func TestAnalyzerKeepsExplicitTypenamesDistinct(t *testing.T) {
	t.Parallel()

	analyzer := NewAnalyzer(expr.NewRandom("ir"), false)
	first := &expr.AttributeExpr{
		Meta: expr.MetaExpr{"openapi:typename": []string{"FooPayload"}},
		Type: &expr.UserTypeExpr{
			AttributeExpr: &expr.AttributeExpr{
				Type: &expr.Object{{Name: "value", Attribute: &expr.AttributeExpr{Type: expr.String}}},
			},
			TypeName: "Payload",
		},
	}
	second := &expr.AttributeExpr{
		Meta: expr.MetaExpr{"openapi:typename": []string{"BarPayload"}},
		Type: &expr.UserTypeExpr{
			AttributeExpr: &expr.AttributeExpr{
				Type: &expr.Object{{Name: "value", Attribute: &expr.AttributeExpr{Type: expr.String}}},
			},
			TypeName: "Payload",
		},
	}

	firstSchema := analyzer.AnalyzeSchema(first)
	secondSchema := analyzer.AnalyzeSchema(second)

	require.Equal(t, "#/components/schemas/FooPayload", firstSchema.Ref)
	require.Equal(t, "#/components/schemas/BarPayload", secondSchema.Ref)
}

func TestAnalyzerDeduplicatesUnionEnvelopeSchemas(t *testing.T) {
	t.Parallel()

	alpha := &expr.NamedAttributeExpr{
		Name: "Alpha",
		Attribute: &expr.AttributeExpr{
			Type: &expr.Object{{Name: "alpha", Attribute: &expr.AttributeExpr{Type: expr.String}}},
		},
	}
	beta := &expr.NamedAttributeExpr{
		Name: "Beta",
		Attribute: &expr.AttributeExpr{
			Type: &expr.Object{{Name: "beta", Attribute: &expr.AttributeExpr{Type: expr.String}}},
		},
	}
	union := &expr.Union{TypeName: "Selection", Values: []*expr.NamedAttributeExpr{alpha, beta}}
	analyzer := NewAnalyzer(expr.NewRandom("ir"), false)

	first := analyzer.AnalyzeSchema(&expr.AttributeExpr{Type: union})
	second := analyzer.AnalyzeSchema(&expr.AttributeExpr{Type: union})

	require.Equal(t, first.Discriminator.Mapping, second.Discriminator.Mapping)
	require.Len(t, analyzer.Components(), 2)
}

func TestAnalyzerClaimExplicitNamePanicsOnConflict(t *testing.T) {
	t.Parallel()

	analyzer := NewAnalyzer(expr.NewRandom("ir"), false)
	analyzer.schemaHashes["AuthSessionResponseBody"] = 1

	require.PanicsWithValue(t,
		"openapi: explicit component name \"AuthSessionResponseBody\" is claimed by multiple different schemas; use distinct Meta(\"openapi:typename\", ...) values",
		func() {
			analyzer.ClaimExplicitName("AuthSessionResponseBody", 2)
		},
	)
}
