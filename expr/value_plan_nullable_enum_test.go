package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xeipuuv/gojsonschema"
)

func TestNullableEnumMatchingAgreement(t *testing.T) {
	base := &UserTypeExpr{TypeName: "Status", AttributeExpr: &AttributeExpr{
		Type: String, Nullable: true, Validation: &ValidationExpr{Values: []any{"open", "closed"}},
	}}
	for _, tc := range []struct {
		name           string
		attribute      *AttributeExpr
		valid, invalid string
	}{
		{"string", base.Attribute(), `"open"`, `"other"`},
		{"inherited", &AttributeExpr{Type: base}, `"open"`, `"other"`},
		{"inherited refined", &AttributeExpr{Type: base, Validation: &ValidationExpr{Values: []any{"open"}}}, `"open"`, `"closed"`},
		{"conjoined", &AttributeExpr{Type: String, Nullable: true, Validation: &ValidationExpr{EnumClauses: [][]any{{"open", "closed"}, {"open"}}}}, `"open"`, `"closed"`},
		{"number", &AttributeExpr{Type: Int, Nullable: true, Validation: &ValidationExpr{Values: []any{1, 2}}}, `1`, `3`},
		{"boolean", &AttributeExpr{Type: Boolean, Nullable: true, Validation: &ValidationExpr{Values: []any{true}}}, `true`, `false`},
		{"array", &AttributeExpr{Type: &Array{ElemType: &AttributeExpr{Type: String}}, Nullable: true, Validation: &ValidationExpr{Values: []any{[]string{"open"}}}}, `["open"]`, `[]`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			context := NewValueContext()
			occurrence, err := context.NewOccurrence(tc.attribute)
			require.NoError(t, err)
			plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{Target: DupAtt(tc.attribute), Codec: ValueCodecJSON, Use: ValuePlanDocumentation})
			require.NoError(t, err)
			shape, err := plan.JSONShape()
			require.NoError(t, err)
			schema, err := InlineJSONSchema(tc.attribute)
			require.NoError(t, err)
			for _, value := range []struct {
				wire string
				want bool
			}{
				{"null", true}, {tc.valid, true}, {tc.invalid, false},
			} {
				t.Run(value.wire, func(t *testing.T) {
					result, err := gojsonschema.Validate(gojsonschema.NewBytesLoader(schema), gojsonschema.NewStringLoader(value.wire))
					require.NoError(t, err)
					require.Equal(t, value.want, result.Valid(), "%s", schema)
					t.Run("prepared", func(t *testing.T) {
						matched, err := plan.MatchJSONSchema([]byte(value.wire))
						require.NoError(t, err)
						require.Equal(t, value.want, matched)
					})
					t.Run("lowered", func(t *testing.T) {
						matched, err := shape.Match([]byte(value.wire))
						require.NoError(t, err)
						require.Equal(t, value.want, matched)
					})
				})
			}
		})
	}
}
