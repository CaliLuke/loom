package designfingerprint

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
)

type provenanceRoot struct {
	// Name identifies this test root in the evaluated registry.
	Name string
	// Value is semantic plugin state.
	Value string
	// Next exercises recursive plugin state without changing graph encoding.
	Next *provenanceRoot
}

func TestDesignProvenanceRejectsInvalidIdentity(t *testing.T) {
	core := &expr.RootExpr{}
	var nilCore *expr.RootExpr
	for _, tc := range []struct {
		name    string
		design  Design
		message string
	}{
		{"missing origin", Design{Roots: []eval.Root{core}}, "invalid design package"},
		{"relative origin", Design{Package: "./design", Roots: []eval.Root{core}}, "invalid design package"},
		{"invalid origin", Design{Package: "example.com/a b", Roots: []eval.Root{core}}, "invalid design package"},
		{"missing core", Design{Package: "example.com/design"}, "no core root"},
		{"nil root", Design{Package: "example.com/design", Roots: []eval.Root{nil}}, "is nil"},
		{"typed nil", Design{Package: "example.com/design", Roots: []eval.Root{nilCore}}, "is nil"},
		{"duplicate core", Design{Package: "example.com/design", Roots: []eval.Root{core, &expr.RootExpr{}}}, "duplicate evaluated root"},
		{"duplicate plugin", Design{Package: "example.com/design", Roots: []eval.Root{core, &provenanceRoot{Name: "plugin"}, &provenanceRoot{Name: "plugin"}}}, "duplicate evaluated root"},
		{"missing root name", Design{Package: "example.com/design", Roots: []eval.Root{core, &provenanceRoot{}}}, "has no name"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Digest(tc.design, "gen", "example.com/gen", 3)
			require.ErrorContains(t, err, tc.message)
		})
	}
}

func TestDesignProvenanceIncludesOrderedCyclicPluginRoots(t *testing.T) {
	root := &expr.RootExpr{}
	a := &provenanceRoot{Name: "a", Value: "initial"}
	b := &provenanceRoot{Name: "b", Value: "second"}
	a.Next = b
	b.Next = a
	design := Design{Package: "example.com/design", Roots: []eval.Root{root, a, b}}
	first, err := Digest(design, "gen", "example.com/gen", 3)
	require.NoError(t, err)
	repeat, err := Digest(design, "gen", "example.com/gen", 3)
	require.NoError(t, err)
	require.Equal(t, first, repeat)
	for _, tc := range []struct {
		name   string
		change func()
	}{
		{"plugin value", func() {
			a.Value = "changed"
		}},
		{"root order", func() {
			a.Value = "initial"
			design.Roots = []eval.Root{root, b, a}
		}},
		{"origin", func() {
			design.Roots = []eval.Root{root, a, b}
			design.Package = "example.com/otherdesign"
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.change()
			changed, err := Digest(design, "gen", "example.com/gen", 3)
			require.NoError(t, err)
			require.NotEqual(t, first, changed)
		})
	}
}

func (*provenanceRoot) WalkSets(eval.SetWalker) {
}

func (r *provenanceRoot) EvalName() string {
	return r.Name
}

func (*provenanceRoot) DependsOn() []eval.Root {
	return nil
}

func (*provenanceRoot) Packages() []string {
	return nil
}
