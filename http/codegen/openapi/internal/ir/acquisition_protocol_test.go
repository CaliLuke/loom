package ir

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestAcquisitionProtocolStructuralReuse(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		t.Run(map[bool]string{false: "first_then_second", true: "second_then_first"}[reverse], func(t *testing.T) {
			first := acquisitionProtocolAttribute("First", "first", expr.Int, nil)
			second := acquisitionProtocolAttribute("Second", "second", expr.Int, nil)
			if reverse {
				first, second = second, first
			}
			analyzer := NewAnalyzer(nil, false, WithExampleValue(nil))
			firstSchema := analyzer.AnalyzeSchema(first)
			secondSchema := analyzer.AnalyzeSchema(second)
			require.NoError(t, analyzer.Err())
			require.NotEmpty(t, firstSchema.Ref)
			require.Equal(t, firstSchema.Ref, secondSchema.Ref)
			require.Len(t, analyzer.Components(), 1)
		})
	}
}

func TestAcquisitionProtocolPreferredAndCanonicalClaims(t *testing.T) {
	canonical := expr.MetaExpr{"openapi:typename": {"Shared"}, "openapi:typename:canonical": {"true"}}
	preferred := expr.MetaExpr{"openapi:typename": {"Shared"}}
	t.Run("occupied_name_preferred_suffix_then_canonical_conflict", func(t *testing.T) {
		analyzer := NewAnalyzer(nil, false, WithExampleValue(nil))
		occupied := analyzer.AnalyzeSchema(acquisitionProtocolAttribute("Occupied", "occupied", expr.String, canonical))
		require.Equal(t, "#/components/schemas/Shared", occupied.Ref)
		hinted := analyzer.AnalyzeSchema(acquisitionProtocolAttribute("Hinted", "hinted", expr.Int, preferred))
		require.NotEmpty(t, hinted.Ref)
		require.NotEqual(t, occupied.Ref, hinted.Ref)
		require.Len(t, analyzer.Components(), 2)
		require.PanicsWithValue(t, "openapi: explicit component name \"Shared\" is claimed by multiple different schemas; use distinct Meta(\"openapi:typename\", ...) values", func() {
			analyzer.AnalyzeSchema(acquisitionProtocolAttribute("Claimant", "claimant", expr.Int, canonical))
		})
	})
	t.Run("canonical_then_equivalent_preferred_reuse", func(t *testing.T) {
		analyzer := NewAnalyzer(nil, false, WithExampleValue(nil))
		claimed := analyzer.AnalyzeSchema(acquisitionProtocolAttribute("Claimant", "claimant", expr.Int, canonical))
		hinted := analyzer.AnalyzeSchema(acquisitionProtocolAttribute("Hinted", "hinted", expr.Int, preferred))
		require.NoError(t, analyzer.Err())
		require.Equal(t, "#/components/schemas/Shared", claimed.Ref)
		require.Equal(t, claimed.Ref, hinted.Ref)
		require.Len(t, analyzer.Components(), 1)
	})
}

func TestAcquisitionProtocolContributorReentry(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		t.Run(map[bool]string{false: "parent_first", true: "child_first"}[reverse], func(t *testing.T) {
			root := codegen.RunDSL(t, func() {
				dsl.API("acquisition", func() {})
				detail := dsl.Type("Detail", func() {
					dsl.Meta("openapi:typename:canonical", "true")
					dsl.Attribute("note", dsl.String)
					dsl.Required("note")
					dsl.Example(map[string]any{"note": "declared detail"})
				})
				parent := dsl.Type("Pet", func() {
					dsl.Meta("openapi:typename:canonical", "true")
					dsl.Meta("openapi:discriminator:propertyName", "kind")
					dsl.Attribute("kind", dsl.String, func() {
						dsl.Example("cat")
					})
					dsl.Attribute("detail", detail, func() {
						dsl.Description("Detail at the parent occurrence")
						dsl.Example(map[string]any{"note": "parent detail"})
					})
					dsl.Required("kind")
				})
				child := dsl.Type("Cat", func() {
					dsl.Meta("openapi:typename:canonical", "true")
					dsl.Extend(parent)
					dsl.Meta("openapi:allOf:reference", "true")
					dsl.Attribute("name", dsl.String, func() {
						dsl.Example("Mira")
					})
				})
				dsl.Service("pets", func() {
					methods := []struct {
						name   string
						result expr.UserType
					}{{"parent", parent}, {"child", child}}
					if reverse {
						methods[0], methods[1] = methods[1], methods[0]
					}
					for _, method := range methods {
						dsl.Method(method.name, func() {
							dsl.Result(method.result)
							dsl.HTTP(func() {
								dsl.GET("/" + method.name)
							})
						})
					}
				})
			})
			t.Log("DSL admitted before contributor analysis")
			document, err := BuildDocument(root.API, root.Types, root.ResultTypes)
			require.NoError(t, err)
			require.Equal(t, map[string][]string{"Pet": {"Cat"}}, document.Components.DiscriminatorChildren)
			pet, cat := document.Components.Schemas["Pet"], document.Components.Schemas["Cat"]
			require.NotNil(t, pet)
			require.NotNil(t, cat)
			require.NotNil(t, pet.Discriminator)
			require.Equal(t, "kind", pet.Discriminator.PropertyName)
			require.Contains(t, pet.Properties, "kind")
			require.GreaterOrEqual(t, len(cat.AllOf), 2)
			require.Equal(t, "#/components/schemas/Pet", cat.AllOf[0].Ref)
			require.Contains(t, cat.AllOf[1].Properties, "name")
			property := pet.Properties["detail"]
			require.NotNil(t, property)
			require.Equal(t, "Detail at the parent occurrence", property.Description)
			require.Equal(t, map[string]any{"note": "parent detail"}, property.Example)
			reference := property.Ref
			if reference == "" {
				require.NotEmpty(t, property.AllOf)
				reference = property.AllOf[0].Ref
			}
			require.Equal(t, "#/components/schemas/Detail", reference)
			require.Contains(t, document.Components.Schemas["Detail"].Properties, "note")
		})
	}
}

func acquisitionProtocolAttribute(name, id string, kind expr.DataType, meta expr.MetaExpr) *expr.AttributeExpr {
	return &expr.AttributeExpr{Type: &expr.UserTypeExpr{
		TypeName: name,
		UID:      id,
		AttributeExpr: &expr.AttributeExpr{
			Meta: meta,
			Type: &expr.Object{{Name: "value", Attribute: &expr.AttributeExpr{Type: kind}}},
		},
	}}
}
