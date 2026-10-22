package transportir

import (
	"testing"

	"github.com/CaliLuke/loom/expr"
	"github.com/stretchr/testify/require"
)

func TestNormalizationSnapshotAllocationScaling(t *testing.T) {
	allocations := func(depth int) float64 {
		attribute := nestedNormalizationAttribute(depth)
		return testing.AllocsPerRun(3, func() {
			normalized := normalizeHTTPAttribute(attribute)
			if normalized == nil {
				panic("normalization returned no output")
			}
		})
	}
	small, large := allocations(16), allocations(64)
	t.Logf("nested depths 16/64 allocate %.0f/%.0f objects", small, large)
	require.Less(t, large/small, float64(7), "four times the input depth must not recapture each remaining subtree")
}

func TestNormalizationSnapshotLocalOverridesAndFreshCalls(t *testing.T) {
	declaration := &expr.UserTypeExpr{TypeName: "Shared", UID: "Shared", AttributeExpr: &expr.AttributeExpr{
		Type: expr.String, DefaultValue: "base", Nullable: true,
	}}
	left := &expr.AttributeExpr{Type: declaration, DefaultValue: "left"}
	right := &expr.AttributeExpr{Type: declaration, DefaultValue: "right"}
	attribute := &expr.AttributeExpr{Type: &expr.Object{
		{Name: "left", Attribute: left}, {Name: "right", Attribute: right},
	}}
	first := normalizeHTTPAttribute(attribute)
	members := *expr.AsObject(first.Type)
	for index, expected := range []string{"left", "right"} {
		require.Equal(t, expr.String, members[index].Attribute.Type)
		require.Equal(t, expected, members[index].Attribute.DefaultValue)
		require.True(t, members[index].Attribute.Nullable)
	}
	require.Same(t, declaration, left.Type)
	require.Same(t, declaration, right.Type)
	require.False(t, left.Nullable)
	require.Equal(t, "base", declaration.DefaultValue)
	left.DefaultValue = "revised"
	second := normalizeHTTPAttribute(attribute)
	require.Equal(t, "revised", (*expr.AsObject(second.Type))[0].Attribute.DefaultValue)
	require.Equal(t, "left", members[0].Attribute.DefaultValue)
}

func TestNormalizationSnapshotRecursiveDeclaration(t *testing.T) {
	node := &expr.UserTypeExpr{TypeName: "Node", UID: "Node"}
	label := &expr.UserTypeExpr{TypeName: "Label", UID: "Label", AttributeExpr: &expr.AttributeExpr{
		Type: expr.String, DefaultValue: "label",
	}}
	node.AttributeExpr = &expr.AttributeExpr{Type: &expr.Object{
		{Name: "label", Attribute: &expr.AttributeExpr{Type: label}},
		{Name: "children", Attribute: &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: node}}}},
	}}
	output := normalizeHTTPAttribute(&expr.AttributeExpr{Type: node})
	actual, ok := output.Type.(expr.UserType)
	require.True(t, ok)
	members := *expr.AsObject(actual.Attribute().Type)
	require.Equal(t, expr.String, members[0].Attribute.Type)
	require.Equal(t, "label", members[0].Attribute.DefaultValue)
	require.Same(t, actual, expr.AsArray(members[1].Attribute.Type).ElemType.Type)
	require.Same(t, label, (*expr.AsObject(node.Type))[0].Attribute.Type)
}

func TestNormalizationSnapshotRejectsInvalidNestedContract(t *testing.T) {
	invalid := &expr.AttributeExpr{Type: expr.String, DefaultValue: 42}
	attribute := &expr.AttributeExpr{Type: &expr.Object{{Name: "bad", Attribute: invalid}}}
	require.Panics(t, func() {
		normalizeHTTPAttribute(attribute)
	})
	require.Equal(t, 42, invalid.DefaultValue)
	require.Equal(t, expr.String, invalid.Type)
	require.Nil(t, normalizeHTTPAttribute(nil))
}

func BenchmarkNormalizationSnapshotDepth(b *testing.B) {
	for _, tc := range []struct {
		name  string
		depth int
	}{{"16", 16}, {"64", 64}, {"256", 256}} {
		b.Run(tc.name, func(b *testing.B) {
			attribute := nestedNormalizationAttribute(tc.depth)
			b.ReportAllocs()
			for b.Loop() {
				normalizeHTTPAttribute(attribute)
			}
		})
	}
}

func TestNormalizationSnapshotPreservesNamedBoundaries(t *testing.T) {
	for _, kind := range []string{"canonical", "result"} {
		t.Run(kind, func(t *testing.T) {
			declaration := &expr.UserTypeExpr{TypeName: "Retained", UID: "Retained", AttributeExpr: &expr.AttributeExpr{
				Type: expr.String, DefaultValue: "retained",
			}}
			var named expr.UserType = declaration
			if kind == "canonical" {
				declaration.Meta = expr.MetaExpr{"openapi:typename:canonical": {"true"}}
			} else {
				named = &expr.ResultTypeExpr{UserTypeExpr: declaration, Identifier: "application/vnd.retained"}
			}
			output := normalizeHTTPAttribute(&expr.AttributeExpr{Type: named})
			actual, ok := output.Type.(expr.UserType)
			require.True(t, ok)
			require.IsType(t, named, actual)
			require.Equal(t, named.Name(), actual.Name())
			require.Equal(t, "retained", actual.Attribute().DefaultValue)
			require.NotSame(t, named.Attribute(), actual.Attribute())
		})
	}
}

func TestNormalizationSnapshotSharedNamedCollections(t *testing.T) {
	key := &expr.UserTypeExpr{TypeName: "Key", UID: "Key", AttributeExpr: &expr.AttributeExpr{
		Type: expr.String, Validation: &expr.ValidationExpr{Pattern: "^key"},
	}}
	value := &expr.UserTypeExpr{TypeName: "Value", UID: "Value", AttributeExpr: &expr.AttributeExpr{
		Type: expr.String, DefaultValue: "value", Nullable: true,
	}}
	index := &expr.UserTypeExpr{TypeName: "Index", UID: "Index", AttributeExpr: &expr.AttributeExpr{
		Type: &expr.Map{KeyType: &expr.AttributeExpr{Type: key}, ElemType: &expr.AttributeExpr{Type: value}},
	}}
	alias := &expr.UserTypeExpr{TypeName: "Alias", UID: "Alias", AttributeExpr: &expr.AttributeExpr{Type: index}}
	input := &expr.AttributeExpr{Type: &expr.Object{
		{Name: "first", Attribute: &expr.AttributeExpr{Type: alias}},
		{Name: "second", Attribute: &expr.AttributeExpr{Type: alias}},
	}}
	output := normalizeHTTPAttribute(input)
	for _, member := range *expr.AsObject(output.Type) {
		actual, ok := member.Attribute.Type.(*expr.Map)
		require.True(t, ok)
		require.Equal(t, expr.String, actual.KeyType.Type)
		require.Equal(t, []string{"^key"}, actual.KeyType.Validation.Patterns())
		require.Equal(t, expr.String, actual.ElemType.Type)
		require.True(t, actual.ElemType.Nullable)
		require.Equal(t, "value", actual.ElemType.DefaultValue)
	}
	require.Same(t, index, alias.Type)
	require.Same(t, key, expr.AsMap(index.Type).KeyType.Type)
}

func nestedNormalizationAttribute(depth int) *expr.AttributeExpr {
	attribute := &expr.AttributeExpr{Type: expr.String}
	for range depth {
		attribute = &expr.AttributeExpr{Type: &expr.Object{{Name: "child", Attribute: attribute}}}
	}
	return attribute
}
