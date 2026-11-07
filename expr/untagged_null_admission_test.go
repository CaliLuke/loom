package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUntaggedNullAdmission(t *testing.T) {
	union := func(branches ...*AttributeExpr) *AttributeExpr {
		value := &Union{TypeName: "Choice", Untagged: true}
		for i, branch := range branches {
			value.Values = append(value.Values, &NamedAttributeExpr{Name: string(rune('A' + i)), Attribute: branch})
		}
		return &AttributeExpr{Type: value}
	}
	nullable := &AttributeExpr{Type: String, Nullable: true}
	integer := &AttributeExpr{Type: Int}
	arbitrary := &AttributeExpr{Type: Any}
	constrainedAny := &AttributeExpr{Type: Any, Validation: &ValidationExpr{Values: []any{"text"}}}
	cycle := union(integer)
	AsUnion(cycle.Type).Values = append(AsUnion(cycle.Type).Values, &NamedAttributeExpr{Name: "Self", Attribute: cycle})
	for _, tc := range []struct {
		name      string
		attribute *AttributeExpr
		want      bool
	}{
		{"nullable branch", union(nullable, integer), true},
		{"any branch", union(arbitrary, integer), true},
		{"two nullable", union(nullable, nullable), false},
		{"nested unique", union(union(arbitrary, integer), integer), true},
		{"nested ambiguous", union(union(arbitrary, nullable), integer), false},
		{"constrained any", union(constrainedAny, nullable), true},
		{"any enum", constrainedAny, false},
		{"any enum null", &AttributeExpr{Type: Any, Validation: &ValidationExpr{Values: []any{nil}}}, true},
		{"nullable enum", &AttributeExpr{Type: String, Nullable: true, Validation: &ValidationExpr{Values: []any{"text"}}}, true},
		{"cycle", cycle, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, AllowsNull(tc.attribute))
		})
	}
}
