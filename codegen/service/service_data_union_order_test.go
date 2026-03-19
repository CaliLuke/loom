package service

import (
	"testing"

	"github.com/stretchr/testify/require"

	"goa.design/goa/v3/codegen"
	"goa.design/goa/v3/expr"
)

func TestCollectUnionTypesDeterministicAcrossObjectOrder(t *testing.T) {
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

	loc := &codegen.Location{
		RelImportPath: "gen/service",
	}
	forwardNames := collectServiceUnionTypeNames(forward, loc)
	reverseNames := collectServiceUnionTypeNames(reverse, loc)

	require.Len(t, forwardNames, 2)
	require.Equal(t, forwardNames, reverseNames)
}

func TestBuildUnionTypeDataUsesExplicitVariantTags(t *testing.T) {
	scope := codegen.NewNameScope()
	loc := &codegen.Location{
		RelImportPath: "gen/service",
	}

	data := buildUnionTypeData(makeTaggedUnionForTagTest(), scope, loc)

	require.Len(t, data.Fields, 2)
	require.Equal(t, "single", data.Fields[0].TypeTag)
	require.Equal(t, "batch", data.Fields[1].TypeTag)
}

func TestBuildViewUnionTypeDataUsesExplicitVariantTags(t *testing.T) {
	scope := codegen.NewNameScope()
	loc := &codegen.Location{
		RelImportPath: "gen/service/views",
	}

	data := buildViewUnionTypeData(makeTaggedUnionForTagTest(), scope, loc)

	require.Len(t, data.Fields, 2)
	require.Equal(t, "single", data.Fields[0].TypeTag)
	require.Equal(t, "batch", data.Fields[1].TypeTag)
}

func TestBuildUnionTypeDataAllowsEmptyOptionalObjectBranches(t *testing.T) {
	scope := codegen.NewNameScope()
	loc := &codegen.Location{
		RelImportPath: "gen/service",
	}

	data := buildUnionTypeData(makeOptionalObjectUnionForFormTest(), scope, loc)

	require.Len(t, data.Fields, 2)
	require.True(t, data.Fields[0].FlatFormObject)
	require.True(t, data.Fields[0].FlatFormObjectAllowsEmpty)
	require.Contains(t, data.Fields[0].EmptyValueExpr, "{}")
	require.True(t, data.Fields[1].FlatFormObject)
	require.False(t, data.Fields[1].FlatFormObjectAllowsEmpty)
}

func TestBuildViewUnionTypeDataAllowsEmptyOptionalObjectBranches(t *testing.T) {
	scope := codegen.NewNameScope()
	loc := &codegen.Location{
		RelImportPath: "gen/service/views",
	}

	data := buildViewUnionTypeData(makeOptionalObjectUnionForFormTest(), scope, loc)

	require.Len(t, data.Fields, 2)
	require.True(t, data.Fields[0].FlatFormObjectAllowsEmpty)
	require.Contains(t, data.Fields[0].EmptyValueExpr, "{}")
	require.False(t, data.Fields[1].FlatFormObjectAllowsEmpty)
}

func collectServiceUnionTypeNames(att *expr.AttributeExpr, loc *codegen.Location) map[string]string {
	scope := codegen.NewNameScope()
	seen := make(map[string]struct{})
	unionByHash := make(map[string]*UnionTypeData)
	collectUnionTypes(att, scope, loc, unionByHash, seen)

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
