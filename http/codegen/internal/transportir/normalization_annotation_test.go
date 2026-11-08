package transportir

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

// Flattening a named alias chain removes the declaration edges, so the
// annotations those declarations own must be captured with the edge, inner
// declarations first and the nearer authored value winning.
func TestNormalizationFlatteningCapturesRemovedEdgeAnnotations(t *testing.T) {
	for _, test := range []struct {
		name       string
		alias      expr.MetaExpr
		aliasDesc  string
		occurrence string
		deprecated string
		desc       string
	}{
		{name: "inherited", deprecated: "true", desc: "Base description."},
		{name: "alias override false", alias: expr.MetaExpr{"openapi:deprecated": {"false"}}, deprecated: "false", desc: "Base description."},
		{name: "alias description", aliasDesc: "Alias description.", deprecated: "true", desc: "Alias description."},
		{name: "occurrence description", occurrence: "Occurrence description.", deprecated: "true", desc: "Occurrence description."},
	} {
		t.Run(test.name, func(t *testing.T) {
			base := &expr.UserTypeExpr{TypeName: "DeprecatedValue", AttributeExpr: &expr.AttributeExpr{
				Type: expr.String, Description: "Base description.",
				Meta: expr.MetaExpr{"openapi:deprecated": {"true"}, "openapi:typename": {"DeprecatedValue"}},
			}}
			aliasMeta := expr.MetaExpr{"openapi:typename": {"ValueAlias"}}
			for key, values := range test.alias {
				aliasMeta[key] = values
			}
			alias := &expr.UserTypeExpr{TypeName: "ValueAlias", AttributeExpr: &expr.AttributeExpr{Type: base, Description: test.aliasDesc, Meta: aliasMeta}}
			body := &expr.AttributeExpr{Type: &expr.Object{{Name: "value", Attribute: &expr.AttributeExpr{Type: alias, Description: test.occurrence}}}}

			value := expr.AsObject(normalizeHTTPAttribute(body).Type).Attribute("value")
			require.Equal(t, expr.String, value.Type, "the alias chain is still flattened")
			deprecated, _ := value.Meta.Last("openapi:deprecated")
			require.Equal(t, test.deprecated, deprecated)
			require.Equal(t, test.desc, value.Description)
			require.NotContains(t, value.Meta, "openapi:typename", "identity stays with the removed declaration")
			require.Equal(t, "Base description.", base.Description, "the shared declaration is not mutated")
		})
	}
}

func TestNormalizationRepeatedAliasAnnotations(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		for _, override := range []bool{false, true} {
			name := fmt.Sprintf("reverse=%t/override=%t", reverse, override)
			t.Run(name, func(t *testing.T) {
				base := &expr.UserTypeExpr{TypeName: "Base", AttributeExpr: &expr.AttributeExpr{
					Type: expr.String, Title: "Base title", Description: "Base description.",
					Meta: expr.MetaExpr{"openapi:deprecated": {"true"}},
				}}
				alias := &expr.UserTypeExpr{TypeName: "Alias", AttributeExpr: &expr.AttributeExpr{Type: base}}
				first, second := &expr.AttributeExpr{Type: alias}, &expr.AttributeExpr{Type: alias}
				if override {
					first.Description = "Local description."
					first.Meta = expr.MetaExpr{"openapi:deprecated": {"false"}}
				}
				fields := expr.Object{{Name: "first", Attribute: first}, {Name: "second", Attribute: second}}
				if reverse {
					fields[0], fields[1] = fields[1], fields[0]
				}
				actual := expr.AsObject(normalizeHTTPAttribute(&expr.AttributeExpr{Type: &fields}).Type)
				for _, field := range []string{"first", "second"} {
					member := actual.Attribute(field)
					description, deprecated := "Base description.", "true"
					if field == "first" && override {
						description, deprecated = "Local description.", "false"
					}
					require.Equal(t, expr.String, member.Type, field)
					require.Equal(t, "Base title", member.Title, field)
					require.Equal(t, description, member.Description, field)
					require.Equal(t, []string{deprecated}, member.Meta["openapi:deprecated"], field)
				}
				require.Same(t, base, alias.Type, "normalization owns a copy of the declaration graph")
				require.Empty(t, alias.Description)
				require.Equal(t, "Base description.", base.Description)
				require.Equal(t, []string{"true"}, base.Meta["openapi:deprecated"])
			})
		}
	}
}

func TestNormalizationRepeatedAliasCycleAdmission(t *testing.T) {
	alias := &expr.UserTypeExpr{TypeName: "Cycle", AttributeExpr: &expr.AttributeExpr{}}
	alias.Type = alias
	require.Panics(t, func() {
		normalizeHTTPAttribute(&expr.AttributeExpr{Type: alias})
	}, "the existing occurrence owner rejects an alias-only cycle before normalization")
}
