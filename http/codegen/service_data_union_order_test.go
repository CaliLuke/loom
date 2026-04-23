package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	cg "github.com/CaliLuke/loom/codegen"
	svc "github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
)

func TestCollectHTTPUnionTypesDeterministicAcrossObjectOrder(t *testing.T) {
	sourceFromSignals := makeUnionForOrderTest("source",
		"physical_point",
		"synthetic_series",
	)
	sourceFromInputs := makeUnionForOrderTest("source",
		"time_series",
		"energy_rates",
	)

	forward := &expr.AttributeExpr{
		Type: &expr.Object{
			{
				Name: "alpha",
				Attribute: &expr.AttributeExpr{
					Type: sourceFromSignals,
				},
			},
			{
				Name: "beta",
				Attribute: &expr.AttributeExpr{
					Type: sourceFromInputs,
				},
			},
		},
	}
	reverse := &expr.AttributeExpr{
		Type: &expr.Object{
			{
				Name: "beta",
				Attribute: &expr.AttributeExpr{
					Type: sourceFromInputs,
				},
			},
			{
				Name: "alpha",
				Attribute: &expr.AttributeExpr{
					Type: sourceFromSignals,
				},
			},
		},
	}

	forwardNames := collectHTTPUnionTypeNames(forward)
	reverseNames := collectHTTPUnionTypeNames(reverse)

	require.Len(t, forwardNames, 2)
	require.Equal(t, forwardNames, reverseNames)
}

func TestBuildHTTPUnionTypeDataUsesExplicitVariantTags(t *testing.T) {
	scope := cg.NewNameScope()

	data := buildHTTPUnionTypeData(makeTaggedUnionForTagTest(), scope)

	require.Len(t, data.Fields, 2)
	require.Equal(t, "single", data.Fields[0].TypeTag)
	require.Equal(t, "batch", data.Fields[1].TypeTag)
}

func TestBuildHTTPUnionTypeDataAllowsEmptyOptionalObjectBranches(t *testing.T) {
	scope := cg.NewNameScope()

	data := buildHTTPUnionTypeData(makeOptionalObjectUnionForFormTest(), scope)

	require.Len(t, data.Fields, 2)
	require.True(t, data.Fields[0].FlatFormObject)
	require.True(t, data.Fields[0].FlatFormObjectAllowsEmpty)
	require.Contains(t, data.Fields[0].EmptyValueExpr, "{}")
	require.True(t, data.Fields[1].FlatFormObject)
	require.False(t, data.Fields[1].FlatFormObjectAllowsEmpty)
}

func TestRenderHTTPUnionUnmarshalJSONReturnsStructuredErrors(t *testing.T) {
	scope := cg.NewNameScope()
	data := buildHTTPUnionTypeData(makeTaggedUnionForTagTest(), scope)

	body := renderHTTPUnionUnmarshalJSONBody(data)

	require.Contains(t, body, `return loom.MissingFieldError("value", "body")`)
	require.Contains(t, body, `return loom.InvalidEnumValueError("type", raw.Type, []any{`)
	require.NotContains(t, body, `unexpected Selection type`)
}

func TestRenderHTTPUnionUnmarshalFormReturnsStructuredEnumError(t *testing.T) {
	scope := cg.NewNameScope()
	data := buildHTTPUnionTypeData(makeTaggedUnionForTagTest(), scope)

	body := renderHTTPUnionUnmarshalFormBody(data)

	require.Contains(t, body, `return loom.InvalidEnumValueError("type", rawType, []any{`)
	require.NotContains(t, body, `unexpected Selection type`)
}

func collectHTTPUnionTypeNames(att *expr.AttributeExpr) map[string]string {
	scope := cg.NewNameScope()
	seen := make(map[string]struct{})
	unionByHash := make(map[string]*svc.UnionTypeData)
	collectHTTPUnionTypes(att, scope, unionByHash, seen)

	names := make(map[string]string, len(unionByHash))
	for hash, data := range unionByHash {
		names[hash] = data.Name
	}
	return names
}

func makeUnionForOrderTest(typeName string, variants ...string) *expr.Union {
	values := make([]*expr.NamedAttributeExpr, len(variants))
	for i, variant := range variants {
		values[i] = &expr.NamedAttributeExpr{
			Name: variant,
			Attribute: &expr.AttributeExpr{
				Type: expr.String,
			},
		}
	}
	return &expr.Union{
		TypeName: typeName,
		Values:   values,
	}
}

func makeTaggedUnionForTagTest() *expr.Union {
	return &expr.Union{
		TypeName: "Selection",
		Values: []*expr.NamedAttributeExpr{
			{
				Name: "Single",
				Attribute: &expr.AttributeExpr{
					Type: expr.String,
					Meta: expr.MetaExpr{"oneof:type:tag": []string{"single"}},
				},
			},
			{
				Name: "Batch",
				Attribute: &expr.AttributeExpr{
					Type: expr.String,
					Meta: expr.MetaExpr{"oneof:type:tag": []string{"batch"}},
				},
			},
		},
	}
}

func makeOptionalObjectUnionForFormTest() *expr.Union {
	return &expr.Union{
		TypeName: "Grant",
		Values: []*expr.NamedAttributeExpr{
			{
				Name: "Refresh",
				Attribute: &expr.AttributeExpr{
					Type: &expr.Object{},
				},
			},
			{
				Name: "AuthorizationCode",
				Attribute: &expr.AttributeExpr{
					Type: &expr.Object{
						{
							Name: "code",
							Attribute: &expr.AttributeExpr{
								Type: expr.String,
							},
						},
					},
					Validation: &expr.ValidationExpr{Required: []string{"code"}},
				},
			},
		},
	}
}
