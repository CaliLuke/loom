package expr

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

type hashCase struct {
	name      string
	a, b      func() DataType
	wantEqual bool
}

func TestHashFieldsRecursiveEquality(t *testing.T) {
	cases := []hashCase{
		{
			name:      "identical self recursive types",
			a:         func() DataType { return hashSelfRecursive("T") },
			b:         func() DataType { return hashSelfRecursive("T") },
			wantEqual: true,
		},
		{
			name: "self recursive types with different names",
			a:    func() DataType { return hashSelfRecursive("T") },
			b:    func() DataType { return hashSelfRecursive("U") },
		},
		{
			name: "self recursive type and its one step unrolling",
			a:    func() DataType { return hashSelfRecursive("T") },
			b: func() DataType {
				return hashNamedObject("T", "a", &AttributeExpr{Type: hashSelfRecursive("T")})
			},
			wantEqual: true,
		},
		{
			name: "self recursive type and finite type with same names",
			a:    func() DataType { return hashSelfRecursive("T") },
			b: func() DataType {
				return hashNamedObject("T", "a", &AttributeExpr{Type: hashNamedEmpty("T")})
			},
		},
		{
			name: "self recursive type and two step finite type",
			a:    func() DataType { return hashSelfRecursive("T") },
			b: func() DataType {
				inner := hashNamedObject("T", "a", &AttributeExpr{Type: hashNamedEmpty("T")})
				return hashNamedObject("T", "a", &AttributeExpr{Type: inner})
			},
		},
		{
			name:      "identical mutually recursive types",
			a:         func() DataType { return hashMutual("A", "B") },
			b:         func() DataType { return hashMutual("A", "B") },
			wantEqual: true,
		},
		{
			name: "mutually recursive type and finite type with same names",
			a:    func() DataType { return hashMutual("A", "B") },
			b: func() DataType {
				b := hashNamedObject("B", "a", &AttributeExpr{Type: hashNamedEmpty("A")})
				return hashNamedObject("A", "b", &AttributeExpr{Type: b})
			},
		},
		{
			name:      "mutually recursive types with different names",
			a:         func() DataType { return hashMutual("A", "B") },
			b:         func() DataType { return hashMutual("A", "C") },
			wantEqual: false,
		},
		{
			name: "recursive types differing in nullability of back edge",
			a:    func() DataType { return hashSelfRecursive("T") },
			b: func() DataType {
				ut := hashNamedEmpty("T")
				ut.Type = &Object{{Name: "a", Attribute: &AttributeExpr{Type: ut, Nullable: true}}}
				return ut
			},
		},
		{
			name: "recursive types differing in field name",
			a:    func() DataType { return hashSelfRecursive("T") },
			b: func() DataType {
				ut := hashNamedEmpty("T")
				ut.Type = &Object{{Name: "b", Attribute: &AttributeExpr{Type: ut}}}
				return ut
			},
		},
		{
			name: "recursive through array and map",
			a: func() DataType {
				ut := hashNamedEmpty("T")
				ut.Type = &Object{
					{Name: "list", Attribute: &AttributeExpr{Type: &Array{ElemType: &AttributeExpr{Type: ut}}}},
					{Name: "dict", Attribute: &AttributeExpr{Type: &Map{KeyType: &AttributeExpr{Type: String}, ElemType: &AttributeExpr{Type: ut}}}},
				}
				return ut
			},
			b: func() DataType {
				ut := hashNamedEmpty("T")
				ut.Type = &Object{
					{Name: "dict", Attribute: &AttributeExpr{Type: &Map{KeyType: &AttributeExpr{Type: String}, ElemType: &AttributeExpr{Type: ut}}}},
					{Name: "list", Attribute: &AttributeExpr{Type: &Array{ElemType: &AttributeExpr{Type: ut}}}},
				}
				return ut
			},
			wantEqual: true,
		},
		{
			name: "recursive through union with different tagging",
			a:    func() DataType { return hashRecursiveUnion(false) },
			b:    func() DataType { return hashRecursiveUnion(true) },
		},
		{
			name: "identical non recursive objects",
			a: func() DataType {
				return &Object{{Name: "a", Attribute: &AttributeExpr{Type: Int}}, {Name: "b", Attribute: &AttributeExpr{Type: String}}}
			},
			b: func() DataType {
				return &Object{{Name: "b", Attribute: &AttributeExpr{Type: String}}, {Name: "a", Attribute: &AttributeExpr{Type: Int}}}
			},
			wantEqual: true,
		},
		{
			name: "non recursive objects with different field types",
			a:    func() DataType { return &Object{{Name: "a", Attribute: &AttributeExpr{Type: Int}}} },
			b:    func() DataType { return &Object{{Name: "a", Attribute: &AttributeExpr{Type: Int64}}} },
		},
		{
			name: "non recursive user types with different names",
			a:    func() DataType { return hashNamedObject("A", "x", &AttributeExpr{Type: Int}) },
			b:    func() DataType { return hashNamedObject("B", "x", &AttributeExpr{Type: Int}) },
		},
		{
			name: "non recursive user types with different struct tags",
			a: func() DataType {
				return hashNamedObject("A", "x", &AttributeExpr{Type: Int, Meta: MetaExpr{"struct:field:json": []string{"x"}}})
			},
			b: func() DataType {
				return hashNamedObject("A", "x", &AttributeExpr{Type: Int, Meta: MetaExpr{"struct:field:json": []string{"y"}}})
			},
		},
		{
			name: "shared and duplicated nested types",
			a: func() DataType {
				inner := hashNamedObject("I", "v", &AttributeExpr{Type: String})
				return &Object{{Name: "a", Attribute: &AttributeExpr{Type: inner}}, {Name: "b", Attribute: &AttributeExpr{Type: inner}}}
			},
			b: func() DataType {
				return &Object{
					{Name: "a", Attribute: &AttributeExpr{Type: hashNamedObject("I", "v", &AttributeExpr{Type: String})}},
					{Name: "b", Attribute: &AttributeExpr{Type: hashNamedObject("I", "v", &AttributeExpr{Type: String})}},
				}
			},
			wantEqual: true,
		},
		{
			name: "array element nullability",
			a:    func() DataType { return &Array{ElemType: &AttributeExpr{Type: String}} },
			b:    func() DataType { return &Array{ElemType: &AttributeExpr{Type: String, Nullable: true}} },
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ha := Hash(tc.a(), false, false, false)
			hb := Hash(tc.b(), false, false, false)
			if tc.wantEqual {
				require.Equal(t, ha, hb)
			} else {
				require.NotEqual(t, ha, hb)
			}
			require.Equal(t, ha, Hash(tc.a(), false, false, false), "hash must be deterministic")
		})
	}
}

func TestHashFieldsIgnoreNames(t *testing.T) {
	a := Hash(hashMutual("A", "B"), false, true, true)
	b := Hash(hashMutual("X", "Y"), false, true, true)
	require.Equal(t, a, b)
	require.NotEqual(t, Hash(hashMutual("A", "B"), false, false, true), Hash(hashMutual("X", "Y"), false, false, true))
}

func TestHashIgnoreFieldsUsesNamesOnly(t *testing.T) {
	recursive := Hash(hashSelfRecursive("T"), true, false, true)
	finite := Hash(hashNamedObject("T", "a", &AttributeExpr{Type: Int}), true, false, true)
	require.Equal(t, "_t_T", recursive)
	require.Equal(t, recursive, finite)
}

func TestHashFieldsCliqueIsPolynomial(t *testing.T) {
	for n := 10; n <= 12; n++ {
		t.Run(fmt.Sprintf("n=%d", n), func(t *testing.T) {
			clique := hashClique(n)
			h := newFieldHasher(true, true)
			got := h.hash(clique[0])
			// The graph has n user types and n objects joined by n*n+n
			// edges; each refinement round visits every node and edge once
			// and there are at most as many rounds as nodes.
			bound := 2 * n * (2*n + n*n + n)
			require.LessOrEqual(t, h.steps, bound)
			require.Positive(t, h.steps)
			require.Less(t, len(got), 64*n*n)
			require.Equal(t, got, Hash(clique[0], false, true, true))
			require.NotEqual(t, got, Hash(hashClique(n - 1)[0], false, true, true))
		})
	}
}

func hashNamedEmpty(name string) *UserTypeExpr {
	return &UserTypeExpr{TypeName: name, AttributeExpr: &AttributeExpr{Type: &Object{}}}
}

func hashNamedObject(name, field string, att *AttributeExpr) *UserTypeExpr {
	return &UserTypeExpr{TypeName: name, AttributeExpr: &AttributeExpr{Type: &Object{{Name: field, Attribute: att}}}}
}

func hashSelfRecursive(name string) *UserTypeExpr {
	ut := hashNamedEmpty(name)
	ut.Type = &Object{{Name: "a", Attribute: &AttributeExpr{Type: ut}}}
	return ut
}

func hashMutual(first, second string) *UserTypeExpr {
	a := hashNamedEmpty(first)
	b := hashNamedObject(second, "a", &AttributeExpr{Type: a})
	a.Type = &Object{{Name: "b", Attribute: &AttributeExpr{Type: b}}}
	return a
}

func hashRecursiveUnion(untagged bool) *UserTypeExpr {
	ut := hashNamedEmpty("T")
	u := &Union{TypeName: "U", Untagged: untagged, Values: []*NamedAttributeExpr{
		{Name: "leaf", Attribute: &AttributeExpr{Type: String}},
		{Name: "node", Attribute: &AttributeExpr{Type: ut}},
	}}
	ut.Type = &Object{{Name: "v", Attribute: &AttributeExpr{Type: u}}}
	return ut
}

func hashClique(n int) []*UserTypeExpr {
	types := make([]*UserTypeExpr, n)
	for i := range types {
		types[i] = hashNamedEmpty(fmt.Sprintf("T%d", i))
	}
	for i, ut := range types {
		obj := &Object{}
		for j, other := range types {
			obj.Set(fmt.Sprintf("f%d_%d", i, j), &AttributeExpr{Type: other})
		}
		ut.Type = obj
	}
	return types
}
