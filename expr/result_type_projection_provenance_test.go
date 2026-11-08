package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// A transport body copy of a result type keeps its pre-rename name as
// provenance. A generated projection description names that declaration, so an
// authored contract and its body copy project the same description; authored
// descriptions are unchanged.
func TestProjectedResultDescriptionUsesRenameProvenance(t *testing.T) {
	for _, test := range []struct {
		name        string
		description string
		meta        MetaExpr
		want        string
	}{
		{name: "declaration", want: "Viewedevent result type (tiny view)"},
		{name: "renamed body copy", meta: MetaExpr{"name:original": {"Viewedevent"}}, want: "Viewedevent result type (tiny view)"},
		{name: "renamed twice", meta: MetaExpr{"name:original": {"Viewedevent", "WatchResponseBody"}}, want: "Viewedevent result type (tiny view)"},
		{name: "authored", description: "Authored event.", meta: MetaExpr{"name:original": {"Viewedevent"}}, want: "Authored event. (tiny view)"},
	} {
		t.Run(test.name, func(t *testing.T) {
			typeName := "Viewedevent"
			if test.meta != nil {
				typeName = "WatchResponseBody"
			}
			rt := &ResultTypeExpr{UserTypeExpr: &UserTypeExpr{TypeName: typeName, AttributeExpr: &AttributeExpr{Type: &Object{}, Description: test.description, Meta: test.meta}}}
			require.Equal(t, test.want, projectedResultDescription(rt, "tiny"))
		})
	}
}
