package unionnames

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestPackageDeclarations(t *testing.T) {
	union := &expr.Union{}
	for _, name := range []string{"Kind", "a-b", "a_b", "AB2"} {
		union.Values = append(union.Values, &expr.NamedAttributeExpr{Name: name})
	}
	symbols := PackageDeclarations(union, "Choice3")
	require.Equal(t, "Choice3Kind", symbols.KindName)
	require.Equal(t, BranchSymbols{KindConst: "Choice3KindKind", Constructor: "NewChoice3Kind"}, symbols.Branches["Kind"])
	require.Equal(t, BranchSymbols{KindConst: "Choice3KindAB3", Constructor: "NewChoice3AB3"}, symbols.Branches["a_b"])
	members := Resolve(union)
	for name, member := range members {
		require.Equal(t, BranchDeclarations("Choice3", member.Method), symbols.Branches[name])
	}
	seen := make(map[string]bool)
	for _, name := range symbols.Names {
		require.False(t, seen[name], "duplicate %s", name)
		seen[name] = true
	}
	require.Len(t, seen, 1+2*len(union.Values))
	slices.Reverse(union.Values)
	reversed := PackageDeclarations(union, "Choice3")
	require.Equal(t, symbols.Branches, reversed.Branches)
	require.ElementsMatch(t, symbols.Names, reversed.Names)
}
