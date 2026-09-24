package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestNameScope_Unique(t *testing.T) {
	sequence := []struct {
		Input    string
		Suffix   []string
		Expected string
	}{
		{Input: "a", Expected: "a"},
		{Input: "a", Expected: "a2"},
		{Input: "a", Expected: "a3"},
		{Input: "a", Expected: "a4"},
		{Input: "b", Expected: "b"},
		{Input: "c", Expected: "c"},
		{Input: "hel", Expected: "hel"},
		{Input: "hel", Suffix: []string{"lo"}, Expected: "hello"},
		{Input: "hello", Expected: "hello2"},
		{Input: "hello", Suffix: []string{"1"}, Expected: "hello1"},
		{Input: "hello", Suffix: []string{"1"}, Expected: "hello12"},
		{Input: "hello", Suffix: []string{"2"}, Expected: "hello22"},
		{Input: "hello", Suffix: []string{"2"}, Expected: "hello23"},
		{Input: "hello,world", Expected: "hello,world"},
		{Input: "hello,world1", Expected: "hello,world1"},
		{Input: "hello,world2", Expected: "hello,world2"},
		{Input: "hello", Suffix: []string{",world"}, Expected: "hello,world3"},
	}

	scope := NewNameScope()
	for i, v := range sequence {
		if got := scope.Unique(v.Input, v.Suffix...); v.Expected != got {
			t.Errorf("#%v, expected %v, got %v", i, v.Expected, got)
		}
	}
}

func TestNameScope_GoFullTypeName_UsesScopedNameWhenQualified(t *testing.T) {
	scope := NewNameScope()

	// Simulate the service generator reserving/using "Request" for a different
	// identifier before naming a user type that also wants to be "Request".
	scope.Unique("Request")

	ut := &expr.UserTypeExpr{
		AttributeExpr: &expr.AttributeExpr{Type: expr.String},
		TypeName:      "Request",
		UID:           "t",
	}
	att := &expr.AttributeExpr{Type: ut}

	if got, want := scope.GoTypeName(att), "Request2"; got != want {
		t.Fatalf("expected scoped type name %q, got %q", want, got)
	}
	if got, want := scope.GoFullTypeName(att, "svc"), "svc.Request2"; got != want {
		t.Fatalf("expected qualified scoped name %q, got %q", want, got)
	}

	fresh := NewNameScope()
	if got, want := fresh.GoFullTypeName(att, "svc"), "svc.Request"; got != want {
		t.Fatalf("expected qualified base name %q with fresh scope, got %q", want, got)
	}
}

func TestNameScopeGoFullTypeNameKeepsAnyMapKeysComparable(t *testing.T) {
	attribute := &expr.AttributeExpr{Type: &expr.Map{
		KeyType:  &expr.AttributeExpr{Type: expr.Any},
		ElemType: &expr.AttributeExpr{Type: expr.String},
	}}

	if got, want := NewNameScope().GoFullTypeName(attribute, "service"), "map[any]string"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestNameScope_PeekUnique_MatchesUniqueWithoutMutation(t *testing.T) {
	seed := func(scope *NameScope) {
		scope.Unique("a")
		scope.Unique("a")
		scope.Unique("a2")
		scope.Unique("hello")
		scope.Unique("hello1")
	}

	peek := NewNameScope()
	seed(peek)

	mutating := NewNameScope()
	seed(mutating)

	if got, want := peek.PeekUnique("a"), mutating.Unique("a"); got != want {
		t.Fatalf("expected peek %q, got %q", want, got)
	}
	if got, want := peek.PeekUnique("hel", "lo"), mutating.Unique("hel", "lo"); got != want {
		t.Fatalf("expected peek %q, got %q", want, got)
	}
	if got, want := peek.PeekUnique("hello", "1"), mutating.Unique("hello", "1"); got != want {
		t.Fatalf("expected peek %q, got %q", want, got)
	}

	// PeekUnique must not mutate the scope.
	if got, want := peek.Unique("a"), "a3"; got != want {
		t.Fatalf("expected scope unchanged, got %q", got)
	}
}

func TestNameScope_PeekUniqueNeverReturnsReservedName(t *testing.T) {
	cases := []struct {
		Name     string
		Reserved []string
		Input    string
		Expected string
	}{
		{Name: "free", Input: "Foo", Expected: "Foo"},
		{Name: "base reserved", Reserved: []string{"Foo"}, Input: "Foo", Expected: "Foo2"},
		{Name: "base reserved twice", Reserved: []string{"Foo", "Foo"}, Input: "Foo", Expected: "Foo3"},
		{Name: "suffixed name reserved directly", Reserved: []string{"Foo", "Foo2"}, Input: "Foo", Expected: "Foo3"},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			scope := NewNameScope()
			for _, r := range c.Reserved {
				scope.Unique(r)
			}
			got := scope.PeekUnique(c.Input)
			require.Equal(t, c.Expected, got)
			require.Equal(t, got, scope.PeekUnique(c.Input), "PeekUnique must be idempotent")
			require.Equal(t, c.Expected, scope.Unique(c.Input), "PeekUnique must not reserve")
		})
	}
}

func TestNameScopeGoFullTypeRefWithPackagesAliasesNestedLocations(t *testing.T) {
	entry := &expr.UserTypeExpr{
		AttributeExpr: &expr.AttributeExpr{
			Type: &expr.Object{{Name: "id", Attribute: &expr.AttributeExpr{Type: expr.String}}},
			Meta: expr.MetaExpr{"struct:pkg:path": []string{"types/log"}},
		},
		TypeName: "Entry",
		UID:      "entry",
	}
	local := &expr.UserTypeExpr{
		AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{{Name: "id", Attribute: &expr.AttributeExpr{Type: expr.String}}}},
		TypeName:      "Local",
		UID:           "local",
	}
	array := func(elem expr.DataType) *expr.Array {
		return &expr.Array{ElemType: &expr.AttributeExpr{Type: elem}}
	}
	mapOf := func(key, elem expr.DataType) *expr.Map {
		return &expr.Map{KeyType: &expr.AttributeExpr{Type: key}, ElemType: &expr.AttributeExpr{Type: elem}}
	}
	alias := func(loc *Location) string {
		return map[string]string{"types/log": "log2"}[loc.RelImportPath]
	}
	cases := []struct {
		Name    string
		Type    expr.DataType
		Want    string
		WantOwn string
	}{
		{Name: "user-type", Type: entry, Want: "*log2.Entry", WantOwn: "*log.Entry"},
		{Name: "array", Type: array(entry), Want: "[]*log2.Entry", WantOwn: "[]*log.Entry"},
		{Name: "map-value", Type: mapOf(expr.String, entry), Want: "map[string]*log2.Entry", WantOwn: "map[string]*log.Entry"},
		{Name: "map-key", Type: mapOf(entry, expr.String), Want: "map[*log2.Entry]string", WantOwn: "map[*log.Entry]string"},
		{
			Name:    "nested",
			Type:    mapOf(expr.String, array(mapOf(expr.String, array(entry)))),
			Want:    "map[string][]map[string][]*log2.Entry",
			WantOwn: "map[string][]map[string][]*log.Entry",
		},
		{Name: "service-local", Type: array(local), Want: "[]*svc.Local", WantOwn: "[]*svc.Local"},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			att := &expr.AttributeExpr{Type: c.Type}
			require.Equal(t, c.Want, NewNameScope().GoFullTypeRefWithPackages(att, "svc", alias))
			require.Equal(t, c.WantOwn, NewNameScope().GoFullTypeRef(att, pkgWithDefault(c.Type, "svc", nil)))
		})
	}
}
