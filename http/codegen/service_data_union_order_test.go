package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	cg "github.com/CaliLuke/loom/codegen"
	svc "github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/unionjson"
	loom "github.com/CaliLuke/loom/pkg"
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

	require.Len(t, forwardNames, 4)
	require.Equal(t, forwardNames, reverseNames)
}

func TestBuildHTTPUnionTypeDataUsesExplicitVariantTags(t *testing.T) {
	scope := cg.NewNameScope()

	data := buildHTTPUnionTypeData(makeTaggedUnionForTagTest(), scope, false)

	require.Len(t, data.Fields, 2)
	require.Equal(t, "single", data.Fields[0].TypeTag)
	require.Equal(t, "batch", data.Fields[1].TypeTag)
}

func TestBuildHTTPUnionTypeDataAllowsEmptyOptionalObjectBranches(t *testing.T) {
	scope := cg.NewNameScope()

	data := buildHTTPUnionTypeData(makeOptionalObjectUnionForFormTest(), scope, false)

	require.Len(t, data.Fields, 2)
	require.True(t, data.Fields[0].FlatFormObject)
	require.True(t, data.Fields[0].FlatFormObjectAllowsEmpty)
	require.Contains(t, data.Fields[0].EmptyValueExpr, "{}")
	require.True(t, data.Fields[1].FlatFormObject)
	require.False(t, data.Fields[1].FlatFormObjectAllowsEmpty)
}

func TestNeedInitSupportsRootUnion(t *testing.T) {
	union := makeTaggedUnionForTagTest()

	require.True(t, needInit(&expr.AttributeExpr{Type: union}))
}

func TestRenderHTTPUntaggedUnionUsesSharedMatcher(t *testing.T) {
	shape := &loom.JSONShape{Kind: "object", Fields: []loom.JSONShapeField{{Name: "wire_name", Required: true, Shape: &loom.JSONShape{Kind: "string"}}}}
	data := &svc.UnionTypeData{Name: "Outcome", Untagged: true, JSON: &unionjson.Union{Name: "Outcome", Branches: []*unionjson.Branch{
		{Type: "*OK", Field: "OK", Kind: "OutcomeKindOK", Schema: shape, Runtime: shape, Validate: "err = ValidateOK(v)"},
	}}}
	body := renderHTTPUnionUnmarshalJSONBody(data) + data.JSON.Matcher()
	require.Contains(t, body, `Name: "wire_name", Required: true`)
	require.Contains(t, body, "loom.MatchUntaggedJSON")
	require.Contains(t, body, "err = ValidateOK(v)")
	require.Contains(t, body, "return Outcome{kind: OutcomeKindOK, OK: value0}, matched, nil")
	require.NotContains(t, body, "u.kind = OutcomeKindOK")
}

func TestRenderHTTPUnionMarshalJSONPreservesDeterministicNestedOrdering(t *testing.T) {
	scope := cg.NewNameScope()
	tagged := buildHTTPUnionTypeData(makeTaggedUnionForTagTest(), scope, false)

	require.Contains(t, renderHTTPUnionMarshalJSONBody(tagged), "}, loom.JSONOptions(), json.Deterministic(true))")

	shape := &loom.JSONShape{Kind: "object"}
	untagged := &svc.UnionTypeData{Name: "Outcome", Untagged: true, JSON: &unionjson.Union{Name: "Outcome", Branches: []*unionjson.Branch{
		{Type: "*OK", Field: "OK", Kind: "OutcomeKindOK", Schema: shape, Runtime: shape},
	}}}
	require.Contains(t, renderHTTPUnionMarshalJSONBody(untagged), "json.Marshal(u.OK, loom.JSONOptions(), json.Deterministic(true))")
}

func TestRenderHTTPUnionUnmarshalJSONReturnsStructuredErrors(t *testing.T) {
	scope := cg.NewNameScope()
	data := buildHTTPUnionTypeData(makeTaggedUnionForTagTest(), scope, false)

	body := renderHTTPUnionUnmarshalJSONBody(data)

	require.Contains(t, body, "json.Unmarshal(raw.Value, &v, loom.JSONOptions())")
	require.Contains(t, body, `return loom.MissingFieldError("value", "body")`)
	require.Contains(t, body, `len(raw.Value) == 0 || string(raw.Value) == "null"`)
	require.Contains(t, body, `return loom.InvalidEnumValueError("type", raw.Type, []any{`)
	require.NotContains(t, body, `unexpected Selection type`)
}

func TestRenderHTTPUnionUnmarshalFormReturnsStructuredEnumError(t *testing.T) {
	scope := cg.NewNameScope()
	data := buildHTTPUnionTypeData(makeTaggedUnionForTagTest(), scope, false)

	body := renderHTTPUnionUnmarshalFormBody(data)

	require.Contains(t, body, `return loom.InvalidEnumValueError("type", rawType, []any{`)
	require.NotContains(t, body, `unexpected Selection type`)
}

func collectHTTPUnionTypeNames(att *expr.AttributeExpr) map[string]string {
	scope := cg.NewNameScope()
	seen := make(map[string]struct{})
	unionByName := make(map[string]*svc.UnionTypeData)
	collectHTTPUnionTypes(att, scope, unionByName, seen, false)

	names := make(map[string]string, len(unionByName))
	for _, data := range unionByName {
		// Key by the branch contract, so this detects swapped allocated names.
		for _, field := range data.Fields {
			names[field.TypeTag] = data.Name
		}
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
