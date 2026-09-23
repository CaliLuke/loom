package expr

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEqual(t *testing.T) {
	var (
		ut1  = userType("ut1", object(String, Int))
		ut2  = userType("ut2", object(String, String))
		aut1 = userType("aut1", arrayOf(object(String, Int)))
		aut2 = userType("aut2", arrayOf(object(String, String)))
		rut  = userType("rut", object(String, Int))
		arut = userType("arut", arrayOf(rut))
	)
	nat := &NamedAttributeExpr{Name: "recursive", Attribute: &AttributeExpr{Type: rut}}
	*rut.Type.(*Object) = append(*rut.Type.(*Object), nat)
	var (
		selfRef  = selfReferencingType("SelfRef", "a")
		selfRef2 = selfReferencingType("SelfRef2", "a")
		empty    = userType("Empty", &Object{})
		wrapper  = userType("Wrapper", &Object{{Name: "a", Attribute: &AttributeExpr{Type: empty}}})
		shared   = userType("Shared", object(String))
		diamond  = userType("Diamond", &Object{
			{Name: "left", Attribute: &AttributeExpr{Type: shared}},
			{Name: "right", Attribute: &AttributeExpr{Type: shared}},
		})
		mutualA  = userType("MutualA", &Object{})
		mutualB  = userType("MutualB", &Object{{Name: "a", Attribute: &AttributeExpr{Type: mutualA}}})
		chain    = userType("Chain", &Object{{Name: "b", Attribute: &AttributeExpr{Type: wrapper}}})
		diamond2 = userType("Diamond2", &Object{
			{Name: "left", Attribute: &AttributeExpr{Type: shared}},
			{Name: "right", Attribute: &AttributeExpr{Type: userType("Other", object(String))}},
		})
	)
	*mutualA.Type.(*Object) = Object{{Name: "b", Attribute: &AttributeExpr{Type: mutualB}}}
	cases := []struct {
		Name     string
		dt, dt2  DataType
		Expected bool
	}{
		{"primitive-true", String, String, true},
		{"primitive-false", String, Int, false},
		{"array-primitive-true", arrayOf(String), arrayOf(String), true},
		{"array-primitive-false", arrayOf(String), arrayOf(Int), false},
		{"map-primitive-true", mapOf(String, String), mapOf(String, String), true},
		{"map-primitive-false", mapOf(String, Int), mapOf(Int, Int), false},
		{"map-primitive-false-2", mapOf(Int, String), mapOf(Int, Int), false},
		{"object-true", object(String, Int), object(String, Int), true},
		{"object-false", object(String, Int), object(String, String), false},
		{"array-object-true", arrayOf(object(String, Float32)), arrayOf(object(String, Float32)), true},
		{"array-object-false", arrayOf(object(String, Float32)), arrayOf(object(Int)), false},
		{"map-object-true", mapOf(object(String, Float32), object(String, Float32)), mapOf(object(String, Float32), object(String, Float32)), true},
		{"map-object-false", mapOf(object(String, Float32), object(Int)), mapOf(object(Int), object(Int)), false},
		{"map-object-false-2", mapOf(object(Int), object(String, Float32)), mapOf(object(Int), object(Int)), false},
		{"user-true", ut1, ut1, true},
		{"user-false", ut1, ut2, false},
		{"user-recursive-true", rut, rut, true},
		{"user-recursive-false", rut, ut2, false},
		{"user-recursive-false-2", ut1, rut, false},
		{"array-user-true", aut1, aut1, true},
		{"array-user-false", aut1, aut2, false},
		{"array-user-recursive-true", arut, arut, true},
		{"array-user-recursive-false", arut, aut2, false},
		{"array-user-recursive-false-2", aut1, arut, false},
		{"user-self-reference-vs-empty-false", selfRef, wrapper, false},
		{"user-empty-vs-self-reference-false", wrapper, selfRef, false},
		{"user-self-reference-isomorphic-true", selfRef, selfRef2, true},
		{"user-shared-reference-true", diamond, diamond2, true},
		{"user-mutual-recursion-vs-empty-false", mutualA, chain, false},
		{"user-mutual-recursion-true", mutualA, mutualA, true},
	}
	for _, k := range cases {
		t.Run(k.Name, func(t *testing.T) {
			res := Equal(k.dt, k.dt2)
			if res != k.Expected {
				t.Errorf("Equal(%q, %q) returned %v but expected %v",
					k.dt.Name(), k.dt2.Name(), res, k.Expected)
			}
		})
	}
}

func arrayOf(dt DataType) *Array {
	return &Array{ElemType: &AttributeExpr{Type: dt}}
}

func mapOf(dt, kt DataType) *Map {
	return &Map{ElemType: &AttributeExpr{Type: dt}, KeyType: &AttributeExpr{Type: kt}}
}

func object(dts ...DataType) *Object {
	var obj Object = make([]*NamedAttributeExpr, len(dts))
	for i, dt := range dts {
		att := &AttributeExpr{Type: dt}
		obj[i] = &NamedAttributeExpr{
			Attribute: att,
			Name:      fmt.Sprintf("att%d", i),
		}
	}
	return &obj
}

func TestEqualCliqueComparesEachPairOnce(t *testing.T) {
	cases := []struct {
		Name     string
		N        int
		Differ   bool
		Expected bool
	}{
		{"clique-10-equal", 10, false, true},
		{"clique-12-equal", 12, false, true},
		{"clique-12-different", 12, true, false},
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			left := cliqueTypes("Left", tc.N)
			right := cliqueTypes("Right", tc.N)
			if tc.Differ {
				last := right[tc.N-1].Type.(*Object)
				*last = append(*last, &NamedAttributeExpr{Name: "extra", Attribute: &AttributeExpr{Type: String}})
			}
			c := &typeComparer{seen: make(map[[2]DataType]struct{})}
			require.Equal(t, tc.Expected, c.equal(left[0], right[0]))
			// Each pair of user types and each pair of objects is expanded
			// once; later references to a pair only hit the seen set.
			require.LessOrEqual(t, c.visits, 2*tc.N*tc.N)
			if tc.Expected {
				// One root lookup, one object lookup per matched type and
				// one lookup per field reference.
				require.Equal(t, tc.N*tc.N+1, c.visits)
			}
		})
	}
}

// cliqueTypes returns n user types where each type has a field referencing
// every other type.
func cliqueTypes(prefix string, n int) []*UserTypeExpr {
	types := make([]*UserTypeExpr, n)
	for i := range types {
		types[i] = userType(fmt.Sprintf("%s%d", prefix, i), &Object{})
	}
	for i, ut := range types {
		obj := ut.Type.(*Object)
		for j, other := range types {
			if i != j {
				*obj = append(*obj, &NamedAttributeExpr{Name: fmt.Sprintf("f%d", j), Attribute: &AttributeExpr{Type: other}})
			}
		}
	}
	return types
}

// selfReferencingType returns a user type T{field: T}.
func selfReferencingType(name, field string) *UserTypeExpr {
	ut := userType(name, &Object{})
	*ut.Type.(*Object) = Object{{Name: field, Attribute: &AttributeExpr{Type: ut}}}
	return ut
}

func userType(name string, dt DataType) *UserTypeExpr {
	return &UserTypeExpr{TypeName: name, AttributeExpr: &AttributeExpr{Type: dt}}
}
