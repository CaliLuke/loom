package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidationExprClauseCopyMergeAndMutation(t *testing.T) {
	validation := &ValidationExpr{
		Values:         []any{"current"},
		EnumClauses:    [][]any{{"current", "base"}},
		Pattern:        "^current",
		PatternClauses: []string{"base$", "^current"},
		Format:         FormatIPv4,
		FormatClauses:  []ValidationFormat{FormatIP, FormatIPv4},
	}
	require.Equal(t, []string{"^current", "base$"}, validation.Patterns())
	require.Equal(t, []ValidationFormat{FormatIPv4, FormatIP}, validation.Formats())
	require.Equal(t, [][]any{{"current"}, {"current", "base"}}, validation.Enums())
	require.False(t, validation.HasRequiredOnly())

	duplicate := validation.Dup()
	duplicate.Pattern = "^replacement"
	duplicate.PatternClauses[0] = "replacement$"
	duplicate.Format = FormatHostname
	duplicate.FormatClauses[0] = FormatEmail
	duplicate.EnumClauses[0][0] = "replacement"
	require.Equal(t, []string{"^current", "base$"}, validation.Patterns())
	require.Equal(t, []ValidationFormat{FormatIPv4, FormatIP}, validation.Formats())
	require.Equal(t, [][]any{{"current"}, {"current", "base"}}, validation.Enums())

	validation.Merge(&ValidationExpr{
		Pattern:        "middle",
		PatternClauses: []string{"base$"},
		Format:         FormatIP,
		FormatClauses:  []ValidationFormat{FormatHostname},
		EnumClauses:    [][]any{{"middle"}, {"current", "base"}},
	})
	require.Equal(t, []string{"^current", "base$", "middle"}, validation.Patterns())
	require.Equal(t, []ValidationFormat{FormatIPv4, FormatIP, FormatHostname}, validation.Formats())
	require.Equal(t, [][]any{{"current"}, {"current", "base"}, {"middle"}}, validation.Enums())
}

func TestValidationExprMergeConjoinsAuthoredEnums(t *testing.T) {
	tests := []struct {
		name     string
		left     *ValidationExpr
		right    *ValidationExpr
		want     [][]any
		authored bool
	}{
		{
			name:     "adopt when receiver absent",
			left:     &ValidationExpr{},
			right:    &ValidationExpr{Values: []any{"base"}},
			want:     [][]any{{"base"}},
			authored: true,
		},
		{
			name:     "equal deduplicates",
			left:     &ValidationExpr{Values: []any{"same"}},
			right:    &ValidationExpr{Values: []any{"same"}},
			want:     [][]any{{"same"}},
			authored: true,
		},
		{
			name:     "distinct values conjoin",
			left:     &ValidationExpr{Values: []any{"derived"}},
			right:    &ValidationExpr{Values: []any{"base"}, EnumClauses: [][]any{{"oldest"}}},
			want:     [][]any{{"derived"}, {"base"}, {"oldest"}},
			authored: true,
		},
		{
			name:     "explicit empty remains present",
			left:     &ValidationExpr{Values: []any{"derived"}},
			right:    &ValidationExpr{Values: []any{}},
			want:     [][]any{{"derived"}, {}},
			authored: true,
		},
		{
			name:  "carrier duplicate remains a clause",
			left:  &ValidationExpr{EnumClauses: [][]any{{"base"}}},
			right: &ValidationExpr{Values: []any{"base"}},
			want:  [][]any{{"base"}},
		},
		{
			name:  "carrier distinct stays receiver first",
			left:  &ValidationExpr{EnumClauses: [][]any{{"derived"}}},
			right: &ValidationExpr{Values: []any{"base"}},
			want:  [][]any{{"derived"}, {"base"}},
		},
		{
			name:  "carrier empty remains present and first",
			left:  &ValidationExpr{EnumClauses: [][]any{{}}},
			right: &ValidationExpr{Values: []any{"base"}},
			want:  [][]any{{}, {"base"}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			before := test.right.Dup()
			test.left.Merge(test.right)
			require.Equal(t, test.want, test.left.Enums())
			require.Equal(t, test.authored, test.left.Values != nil)
			require.Equal(t, before, test.right)
			if len(test.right.Values) > 0 {
				test.right.Values[0] = "mutated"
				require.Equal(t, test.want, test.left.Enums())
			}
		})
	}
}

func TestValidationExprMergePreservesCarrierClauseOrder(t *testing.T) {
	receiver := &ValidationExpr{
		EnumClauses:    [][]any{{"derived"}},
		PatternClauses: []string{"derived$"},
		FormatClauses:  []ValidationFormat{FormatIPv4},
	}
	receiver.Merge(&ValidationExpr{
		Values:  []any{"base"},
		Pattern: "^base",
		Format:  FormatIP,
	})

	require.Nil(t, receiver.Values)
	require.Empty(t, receiver.Pattern)
	require.Empty(t, receiver.Format)
	require.Equal(t, [][]any{{"derived"}, {"base"}}, receiver.Enums())
	require.Equal(t, []string{"derived$", "^base"}, receiver.Patterns())
	require.Equal(t, []ValidationFormat{FormatIPv4, FormatIP}, receiver.Formats())
}

func TestEffectiveEnumClausesPreserveConjunctionAndAuthorship(t *testing.T) {
	tests := []struct {
		name      string
		attribute *AttributeExpr
		accepted  []any
		rejected  []any
		wantEnum  []any
		wantError string
	}{
		{
			name: "authored refinement",
			attribute: &AttributeExpr{Type: String, Validation: &ValidationExpr{
				Values: []any{"ab"}, EnumClauses: [][]any{{"ab", "ax"}},
			}},
			accepted: []any{"ab"}, rejected: []any{"ax"}, wantEnum: []any{"ab"},
		},
		{
			name: "authored widening",
			attribute: &AttributeExpr{Type: String, Validation: &ValidationExpr{
				Values: []any{"ax"}, EnumClauses: [][]any{{"ab"}},
			}},
			wantError: `enum member "ax"`,
		},
		{
			name: "overlap",
			attribute: &AttributeExpr{Type: Int, Validation: &ValidationExpr{
				EnumClauses: [][]any{{1, 2}, {2, 3}},
			}},
			accepted: []any{2}, rejected: []any{1, 3}, wantEnum: []any{2},
		},
		{
			name: "disjoint empty contract",
			attribute: &AttributeExpr{Type: Int, Validation: &ValidationExpr{
				EnumClauses: [][]any{{1}, {2}},
			}},
			rejected: []any{1, 2}, wantEnum: []any{},
		},
		{
			name: "explicit empty clause",
			attribute: &AttributeExpr{Type: Int, Validation: &ValidationExpr{
				EnumClauses: [][]any{{}},
			}},
			rejected: []any{0, 1}, wantEnum: []any{},
		},
		{
			name: "bytes declared equality",
			attribute: &AttributeExpr{Type: Bytes, Validation: &ValidationExpr{
				Values: []any{[]byte("ok")}, EnumClauses: [][]any{{"ok"}},
			}},
			accepted: []any{[]byte("ok")}, rejected: []any{[]byte("no")}, wantEnum: []any{[]byte("ok")},
		},
		{
			name: "nullable",
			attribute: &AttributeExpr{Type: String, Nullable: true, Validation: &ValidationExpr{
				Values: []any{nil}, EnumClauses: [][]any{{nil, "set"}},
			}},
			accepted: []any{nil}, rejected: []any{"set"}, wantEnum: []any{nil},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			constraints, err := EffectiveConstraintsFor(test.attribute)
			if test.wantError != "" {
				require.ErrorContains(t, err, test.wantError)
				return
			}
			require.NoError(t, err)
			actual, present := constraints.EnumCandidates()
			require.True(t, present)
			require.Equal(t, test.wantEnum, actual)
			context := NewValueContext()
			occurrence, err := context.NewOccurrence(test.attribute)
			require.NoError(t, err)
			for _, raw := range test.accepted {
				result := context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: raw}), ValueRoleExample)
				require.Equal(t, ValueResolved, result.Outcome(), "%#v: %v", raw, result.Diagnostics())
			}
			for _, raw := range test.rejected {
				result := context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: raw}), ValueRoleExample)
				require.Equal(t, ValueInvalid, result.Outcome(), "%#v", raw)
			}
		})
	}
}

func TestValidationExprValidateChecksEveryExplicitClause(t *testing.T) {
	validation := &ValidationExpr{
		PatternClauses: []string{"["},
		FormatClauses:  []ValidationFormat{"not-a-format"},
	}
	errors := validation.Validate("attribute: ", &AttributeExpr{Type: String})
	require.ErrorContains(t, errors, `invalid pattern "["`)
	require.ErrorContains(t, errors, `unsupported format "not-a-format"`)
}
