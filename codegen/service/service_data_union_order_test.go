package service

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
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

func TestNamedUnionUserTypeUsesUnionDefinition(t *testing.T) {
	root := expr.RunDSL(t, func() {
		start := dsl.Type("NamedUnionStart", func() {
			dsl.Attribute("start", dsl.String)
		})
		stop := dsl.Type("NamedUnionStop", func() {
			dsl.Attribute("stop", dsl.String)
		})
		command := dsl.Type("NamedUnionCommand", dsl.OneOf(start, stop))
		dsl.Service("NamedUnionService", func() {
			dsl.Method("Run", func() {
				dsl.Payload(command)
			})
		})
	})

	data := NewServicesData(root).Get("NamedUnionService")
	require.Len(t, data.unions, 1)
	require.Equal(t, "NamedUnionCommand", data.unions[0].Name)
	for _, userType := range data.userTypes {
		require.NotEqual(t, "NamedUnionCommand", userType.Name)
	}
}

func TestNamedUnionMethodProjectionUsesUnionDefinition(t *testing.T) {
	root := expr.RunDSL(t, func() {
		start := dsl.Type("ProjectedUnionStart", func() {
			dsl.Attribute("start", dsl.String)
		})
		stop := dsl.Type("ProjectedUnionStop", func() {
			dsl.Attribute("stop", dsl.String)
		})
		command := dsl.Type("ProjectedUnionCommand", dsl.OneOf(start, stop))
		dsl.Service("ProjectedUnionService", func() {
			dsl.Method("Run", func() {
				dsl.Result(command)
			})
		})
	})

	data := NewServicesData(root).Get("ProjectedUnionService")
	require.Len(t, data.Methods, 1)
	require.Equal(t, "ProjectedUnionCommand", data.Methods[0].Result)
	require.Equal(t, "*ProjectedUnionCommand", data.Methods[0].ResultRef)
	require.Empty(t, data.Methods[0].ResultDef)
	require.Len(t, data.unions, 1)
	require.Equal(t, "ProjectedUnionCommand", data.unions[0].Name)
}

func TestUntaggedUnionMetadataUsesEffectiveJSONNames(t *testing.T) {
	root := expr.RunDSL(t, func() {
		first := dsl.Type("JSONNameFirst", func() {
			dsl.Attribute("authored", dsl.String, func() {
				dsl.Meta("struct:tag:json:name", "wire_name")
			})
			dsl.Required("authored")
		})
		second := dsl.Type("JSONNameSecond", func() {
			dsl.Attribute("other", dsl.String)
			dsl.Required("other")
		})
		dsl.Service("JSONNameService", func() {
			dsl.Method("Show", func() {
				dsl.Result(dsl.OneOf(first, second), func() {
					dsl.Untagged()
				})
			})
		})
	})

	data := NewServicesData(root).Get("JSONNameService")
	require.Len(t, data.unions, 1)
	require.Equal(t, []string{"wire_name"}, data.unions[0].Fields[0].RequiredFields)
	require.Equal(t, []string{"wire_name"}, data.unions[0].Fields[0].NonNullableFields)
	require.Equal(t, []string{"wire_name"}, data.unions[0].Fields[0].JSONFields)
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

func TestRenderUnionUnmarshalJSONReturnsStructuredErrors(t *testing.T) {
	scope := codegen.NewNameScope()
	loc := &codegen.Location{
		RelImportPath: "gen/service",
	}
	data := buildUnionTypeData(makeTaggedUnionForTagTest(), scope, loc)

	body := renderUnionUnmarshalJSONBody(data)

	require.Contains(t, body, `return loom.MissingFieldError("value", "body")`)
	require.Contains(t, body, `len(raw.Value) == 0 || string(raw.Value) == "null"`)
	require.Contains(t, body, `return loom.InvalidEnumValueError("type", raw.Type, []any{`)
	require.NotContains(t, body, `unexpected Selection type`)
}

func TestRenderUntaggedUnionJSONUsesBareValidatedBranches(t *testing.T) {
	data := &UnionTypeData{
		Name:     "Outcome",
		KindName: "OutcomeKind",
		Untagged: true,
		Fields: []*UnionFieldData{
			{
				Name:                    "OK",
				FieldName:               "OK",
				FieldType:               "*OK",
				KindConst:               "OutcomeKindOK",
				ValidateCode:            "if v == nil {\n\terr = loom.MissingFieldError(\"ok\", \"v\")\n}",
				RequiredFields:          []string{"wire_name"},
				NonNullableFields:       []string{"wire_name"},
				JSONFields:              []string{"wire_name"},
				RejectUnknownJSONFields: true,
			},
			{
				Name:         "Failure",
				FieldName:    "Failure",
				FieldType:    "*Failure",
				KindConst:    "OutcomeKindFailure",
				ValidateCode: "if v == nil {\n\terr = loom.MissingFieldError(\"failure\", \"v\")\n}",
			},
		},
	}

	marshal := renderUnionMarshalJSONBody(data)
	require.Contains(t, marshal, "return json.Marshal(u.OK, json.Deterministic(true))")
	require.NotContains(t, marshal, "Type  string")

	unmarshal := renderUnionUnmarshalJSONBody(data)
	require.Contains(t, unmarshal, "if branchErr == nil")
	require.Contains(t, unmarshal, "if matches != 1")
	require.Contains(t, unmarshal, "untagged union matched %d branches")
	require.Contains(t, unmarshal, `filtered["wire_name"] = value`)
	require.Contains(t, unmarshal, "matched.kind = OutcomeKindOK")
	require.Contains(t, unmarshal, "*u = matched")
	require.NotContains(t, unmarshal, "u.kind = OutcomeKindOK")
}

func TestRenderTaggedUnionJSONPreservesDeterministicNestedOrdering(t *testing.T) {
	scope := codegen.NewNameScope()
	loc := &codegen.Location{
		RelImportPath: "gen/service",
	}
	data := buildUnionTypeData(makeTaggedUnionForTagTest(), scope, loc)

	marshal := renderUnionMarshalJSONBody(data)
	require.Contains(t, marshal, "}, json.Deterministic(true))")
}

func TestRenderUnionUnmarshalFormReturnsStructuredEnumError(t *testing.T) {
	scope := codegen.NewNameScope()
	loc := &codegen.Location{
		RelImportPath: "gen/service",
	}
	data := buildUnionTypeData(makeTaggedUnionForTagTest(), scope, loc)

	body := renderUnionUnmarshalFormBody(data)

	require.Contains(t, body, `return loom.InvalidEnumValueError("type", rawType, []any{`)
	require.NotContains(t, body, `unexpected Selection type`)
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
