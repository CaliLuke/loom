package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHTTPBodyContributorRetainsOccurrenceSelection(t *testing.T) {
	for _, excluded := range [][]string{{"thumb"}, {"kind", "thumb"}} {
		t.Run(excluded[0], func(t *testing.T) {
			parent := &UserTypeExpr{TypeName: "Parent", AttributeExpr: &AttributeExpr{
				Type:       &Object{{Name: "kind", Attribute: &AttributeExpr{Type: String, DefaultValue: "parent"}}},
				Validation: &ValidationExpr{Required: []string{"kind"}},
			}}
			child := &UserTypeExpr{TypeName: "Child", AttributeExpr: &AttributeExpr{
				Type: &Object{
					{Name: "kind", Attribute: &AttributeExpr{Type: String, DefaultValue: "child"}},
					{Name: "thumb", Attribute: &AttributeExpr{Type: String}},
				},
			}}
			selection := &HTTPBodyProjection{Source: child.Attribute(), Excluded: append([]string(nil), excluded...)}
			projected := ProjectHTTPBodyContributor(parent, selection)
			record := projected.HTTPBodyProjection()
			require.NotNil(t, record)
			require.Equal(t, excluded, record.Excluded, "publication must retain exclusions absent from this contributor")
			require.Same(t, parent.Attribute(), record.Source, "hint binding retains the contributor's own source")
			if excluded[0] == "thumb" {
				require.Equal(t, "parent", AsObject(projected.Type).Attribute("kind").DefaultValue)
			} else {
				require.Empty(t, *AsObject(projected.Type), "physical removal intersects this declaration's members")
			}
			published := ProjectHTTPBodyContributor(child, record)
			require.Nil(t, AsObject(published.Type).Attribute("thumb"), "publication must not reintroduce the excluded child-only field")
			selection.Excluded[0] = "caller mutation"
			record.Excluded[0] = "returned record mutation"
			require.Equal(t, excluded, projected.HTTPBodyProjection().Excluded, "stored selection must be detached")
			require.NotNil(t, AsObject(parent.Type).Attribute("kind"))
			require.NotNil(t, AsObject(child.Type).Attribute("thumb"))
		})
	}
}
