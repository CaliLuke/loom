package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEffectiveConstraintFactsDefaultSources(t *testing.T) {
	for _, tc := range []struct {
		name    string
		typ     DataType
		raw     any
		present bool
	}{
		{name: "absent", typ: String},
		{name: "false", typ: Boolean, raw: false, present: true},
		{name: "zero", typ: Int, raw: 0, present: true},
		{name: "empty string", typ: String, raw: "", present: true},
		{name: "empty array", typ: &Array{ElemType: &AttributeExpr{Type: String}}, raw: []string{}, present: true},
		{name: "empty map", typ: &Map{KeyType: &AttributeExpr{Type: String}, ElemType: &AttributeExpr{Type: String}}, raw: map[string]string{}, present: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			base := &UserTypeExpr{TypeName: "Base", UID: "Base", AttributeExpr: &AttributeExpr{Type: tc.typ, DefaultValue: tc.raw}}
			att := &AttributeExpr{Type: &UserTypeExpr{TypeName: "Alias", UID: "Alias", AttributeExpr: &AttributeExpr{Type: base}}}
			strict, err := EffectiveConstraintsFor(att)
			require.NoError(t, err)
			facts, err := EffectiveConstraintFactsFor(att)
			require.NoError(t, err)
			want, present := strict.Default()
			got, found := facts.DefaultDeclaration()
			require.Equal(t, tc.present, present)
			require.Equal(t, present, found)
			require.Equal(t, present, facts.HasDefault())
			require.Equal(t, want, got)
		})
	}
}

func TestEffectiveConstraintFactsInheritance(t *testing.T) {
	parent := func(value int) *AttributeExpr {
		return &AttributeExpr{Type: Int, DefaultValue: value}
	}
	for _, tc := range []struct {
		name     string
		local    any
		parents  []*AttributeExpr
		want     any
		conflict bool
	}{
		{name: "one inherited", parents: []*AttributeExpr{parent(2)}, want: 2},
		{name: "equal inherited", parents: []*AttributeExpr{parent(2), parent(2)}, want: 2},
		{name: "conflicting inherited remains rejected", parents: []*AttributeExpr{parent(2), parent(3)}, want: 2, conflict: true},
		{name: "local zero resolves conflict", local: 0, parents: []*AttributeExpr{parent(2), parent(3)}, want: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			att := &AttributeExpr{Type: Int, DefaultValue: 99, constraintLocal: &AttributeExpr{Type: Int, DefaultValue: tc.local}, constraintParents: tc.parents}
			facts, err := EffectiveConstraintFactsFor(att)
			require.NoError(t, err)
			value, present := facts.DefaultDeclaration()
			require.True(t, present)
			require.True(t, facts.HasDefault())
			require.Equal(t, tc.want, value)
			strict, err := EffectiveConstraintsFor(att)
			if tc.conflict {
				require.ErrorContains(t, err, "conflicting inherited defaults")
				return
			}
			require.NoError(t, err)
			value, present = strict.Default()
			require.True(t, present)
			require.Equal(t, tc.want, value)
		})
	}
}

func TestEffectiveConstraintFactsPreserveAdmissionBoundary(t *testing.T) {
	base := &AttributeExpr{Type: String, DefaultValue: "short"}
	att := DupAtt(base)
	minimum := 8
	att.Validation = &ValidationExpr{MinLength: &minimum, Pattern: "^s"}
	facts, err := EffectiveConstraintFactsFor(att)
	require.NoError(t, err)
	require.True(t, facts.HasDefault())
	require.Equal(t, 8, *facts.Validation().Lowered().MinLength)
	require.Equal(t, []string{"^s"}, facts.Validation().Lowered().Patterns())
	_, err = EffectiveConstraintsFor(att)
	require.ErrorContains(t, err, "violates the effective contract")
	_, err = NewValueContext().NewOccurrence(att)
	require.ErrorContains(t, err, "violates the effective contract")
	require.Equal(t, "short", base.DefaultValue)
}

func TestEffectiveConstraintFactsRejectPredicateErrors(t *testing.T) {
	minimum, maximum := 8, 2
	for _, tc := range []struct {
		name    string
		att     *AttributeExpr
		message string
	}{
		{name: "contradictory length", att: &AttributeExpr{Type: String, Validation: &ValidationExpr{MinLength: &minimum, MaxLength: &maximum}}, message: "min length is greater than max length"},
		{name: "invalid enum type", att: &AttributeExpr{Type: Int, Validation: &ValidationExpr{Values: []any{"bad"}}}, message: "violates the effective contract"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := EffectiveConstraintFactsFor(tc.att)
			require.ErrorContains(t, err, tc.message)
			_, err = EffectiveConstraintsFor(tc.att)
			require.ErrorContains(t, err, tc.message)
		})
	}
}

func TestEffectiveConstraintFactsDetachedPhysicalRules(t *testing.T) {
	source := &AttributeExpr{Type: &Object{{Name: "content", Attribute: &AttributeExpr{Type: String}}}, Validation: &ValidationExpr{Required: []string{"content"}}, DefaultValue: map[string]any{"content": "original"}}
	facts, err := EffectiveConstraintFactsFor(source)
	require.NoError(t, err)
	source.Validation.Required[0] = "changed"
	source.DefaultValue.(map[string]any)["content"] = "changed"
	require.Equal(t, []string{"content"}, facts.Validation().Lowered().Required)
	value, present := facts.DefaultDeclaration()
	require.True(t, present)
	require.Equal(t, map[string]any{"content": "original"}, value)
	value.(map[string]any)["content"] = "caller mutation"
	next, present := facts.DefaultDeclaration()
	require.True(t, present)
	require.Equal(t, map[string]any{"content": "original"}, next)
}
