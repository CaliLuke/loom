package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEffectiveUnionDefaultRestriction(t *testing.T) {
	makeUnion := func() *Union {
		return &Union{TypeName: "Choice", Untagged: true, Values: []*NamedAttributeExpr{
			{Name: "text", Attribute: &AttributeExpr{Type: String}},
			{Name: "number", Attribute: &AttributeExpr{Type: Int}},
		}}
	}
	for _, name := range []string{"direct", "alias", "inherited"} {
		t.Run(name, func(t *testing.T) {
			var typ DataType = makeUnion()
			if name != "direct" {
				typ = &UserTypeExpr{TypeName: "Alias", AttributeExpr: &AttributeExpr{Type: typ}}
			}
			attribute := &AttributeExpr{Type: typ, DefaultValue: "hello"}
			if name == "inherited" {
				typ.(UserType).Attribute().DefaultValue = "hello"
				attribute.DefaultValue = nil
			}
			_, err := EffectiveConstraintsFor(attribute)
			require.ErrorContains(t, err, "default values are not supported for union attributes")
		})
	}
	t.Run("branch and container defaults remain valid", func(t *testing.T) {
		union := makeUnion()
		union.Values[0].Attribute.DefaultValue = "branch"
		attribute := &AttributeExpr{Type: &Map{KeyType: &AttributeExpr{Type: String}, ElemType: &AttributeExpr{Type: union}}, DefaultValue: map[string]any{"key": "hello"}}
		_, err := EffectiveConstraintsFor(attribute)
		require.NoError(t, err)
		context := NewValueContext()
		occurrence, err := context.NewOccurrence(attribute)
		require.NoError(t, err)
		result := context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: attribute.DefaultValue}), ValueRoleDefault)
		require.Equal(t, ValueResolved, result.Outcome(), "%v", result.Diagnostics())
	})
}
