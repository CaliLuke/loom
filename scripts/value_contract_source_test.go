package scripts_test

import (
	"testing"

	"github.com/CaliLuke/loom/expr"
	"github.com/stretchr/testify/require"
)

// TestValueContractNonObjectPreference characterizes the actual complete-match
// ranking used by the public compatibility adapter, in both declaration orders.
func TestValueContractNonObjectPreference(t *testing.T) {
	known := &expr.NamedAttributeExpr{Name: "known", Attribute: &expr.AttributeExpr{
		Type: &expr.Object{{Name: "name", Attribute: &expr.AttributeExpr{Type: expr.String}}},
		Meta: expr.MetaExpr{"oneof:type:tag": {"known"}},
	}}
	for _, tc := range []struct {
		name     string
		datatype expr.DataType
	}{
		{name: "Any", datatype: expr.Any},
		{name: "named Any", datatype: &expr.UserTypeExpr{TypeName: "AnyAlias", AttributeExpr: &expr.AttributeExpr{Type: expr.Any}}},
		{name: "map", datatype: &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: &expr.AttributeExpr{Type: expr.String}}},
		{name: "nested union", datatype: &expr.Union{TypeKey: "inner", ValueKey: "value", Values: []*expr.NamedAttributeExpr{known}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			flexible := &expr.NamedAttributeExpr{Name: "flexible", Attribute: &expr.AttributeExpr{
				Type: tc.datatype,
				Meta: expr.MetaExpr{"oneof:type:tag": {"flexible"}},
			}}
			input := map[string]any{"name": "alice"}
			sole := &expr.Union{TypeKey: "kind", ValueKey: "payload", Values: []*expr.NamedAttributeExpr{flexible}}
			selected := expr.CanonicalizeExample(&expr.AttributeExpr{Type: sole}, input)
			require.IsType(t, map[string]any{}, selected)
			require.Equal(t, "flexible", selected.(map[string]any)["kind"])
			for _, branches := range [][]*expr.NamedAttributeExpr{{known, flexible}, {flexible, known}} {
				union := &expr.Union{TypeKey: "kind", ValueKey: "payload", Values: branches}
				require.Equal(t, input, expr.CanonicalizeExample(&expr.AttributeExpr{Type: union}, input))
			}
		})
	}
}

func TestValueContractArrayElementNullOverride(t *testing.T) {
	for _, tc := range []struct {
		name     string
		datatype expr.DataType
	}{
		{name: "Any", datatype: expr.Any},
		{name: "nullable named type", datatype: &expr.UserTypeExpr{TypeName: "NullableText", AttributeExpr: &expr.AttributeExpr{Type: expr.String, Nullable: true}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			array := &expr.Array{ElemType: &expr.AttributeExpr{Type: tc.datatype}}
			require.True(t, expr.ArrayElementsAllowNull(array))
			array.NonNullableElems = true
			require.False(t, expr.ArrayElementsAllowNull(array))
		})
	}
}

// TestValueContractAnyHostEquality retains the host-language DeepEqual boundary
// before the legacy recursive numeric, string-map, and slice fallbacks.
func TestValueContractAnyHostEquality(t *testing.T) {
	for _, tc := range []struct {
		name    string
		allowed any
		input   any
		matches bool
	}{
		{name: "int keys", allowed: map[int]string{1: "one"}, input: map[int]string{1: "one"}, matches: true},
		{name: "int64 keys", allowed: map[int64]string{1: "one"}, input: map[int64]string{1: "one"}, matches: true},
		{name: "bool keys", allowed: map[bool]string{true: "yes"}, input: map[bool]string{true: "yes"}, matches: true},
		{name: "different key types", allowed: map[int]string{1: "one"}, input: map[int64]string{1: "one"}},
		{name: "different dynamic value types", allowed: map[int]any{1: int(1)}, input: map[int]any{1: int64(1)}},
		{name: "insertion order", allowed: map[int]string{1: "one", 2: "two"}, input: map[int]string{2: "two", 1: "one"}, matches: true},
		{name: "nested map equality", allowed: map[string]any{"inner": map[int]string{1: "one"}}, input: map[string]any{"inner": map[int]string{1: "one"}}, matches: true},
		{name: "nested host difference", allowed: map[string]any{"inner": map[int]any{1: int(1)}}, input: map[string]any{"inner": map[int]any{1: int64(1)}}},
		{name: "numeric fallback", allowed: map[string]any{"n": int(1)}, input: map[string]any{"n": int64(1)}, matches: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			branch := &expr.NamedAttributeExpr{Name: "raw", Attribute: &expr.AttributeExpr{
				Type:       expr.Any,
				Validation: &expr.ValidationExpr{Values: []any{tc.allowed}},
				Meta:       expr.MetaExpr{"oneof:type:tag": {"raw"}},
			}}
			union := &expr.Union{TypeKey: "kind", ValueKey: "payload", Values: []*expr.NamedAttributeExpr{branch}}
			actual := expr.CanonicalizeExample(&expr.AttributeExpr{Type: union}, tc.input)
			if tc.matches {
				require.IsType(t, map[string]any{}, actual)
				require.Equal(t, "raw", actual.(map[string]any)["kind"])
			} else {
				require.Equal(t, tc.input, actual)
			}
		})
	}
}
