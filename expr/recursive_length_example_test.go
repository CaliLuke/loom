package expr_test

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

// TestRecursiveLengthValidatedExample checks that the example of a recursive
// type whose cycle goes through an array or a map with length validations
// stops at the type that is still being generated, that it is the same for
// the same seed, and that every array and map in it satisfies its length
// validations.
func TestRecursiveLengthValidatedExample(t *testing.T) {
	cases := map[string]struct {
		// Build returns the attribute to generate the example of and the
		// name of the member that holds the recursive collection.
		Build func() (*expr.AttributeExpr, string)
	}{
		"self-array-max-length": {
			Build: func() (*expr.AttributeExpr, string) {
				node := objectType("Node")
				setMembers(node, member("children", arrayOf(node), lengths(nil, new(3))))
				return &expr.AttributeExpr{Type: node}, "children"
			},
		},
		"self-array-min-length": {
			Build: func() (*expr.AttributeExpr, string) {
				node := objectType("Node")
				setMembers(node, member("children", arrayOf(node), lengths(new(2), nil)))
				return &expr.AttributeExpr{Type: node}, "children"
			},
		},
		"self-array-exact-length": {
			Build: func() (*expr.AttributeExpr, string) {
				node := objectType("Node")
				setMembers(node, member("children", arrayOf(node), lengths(new(2), new(2))))
				return &expr.AttributeExpr{Type: node}, "children"
			},
		},
		"mutual-array-cycle": {
			Build: func() (*expr.AttributeExpr, string) {
				a := objectType("A")
				b := objectType("B")
				setMembers(a, member("bs", arrayOf(b), lengths(nil, new(3))))
				setMembers(b, member("as", arrayOf(a), lengths(new(1), new(2))))
				return &expr.AttributeExpr{Type: a}, "bs"
			},
		},
		"self-map-max-length": {
			Build: func() (*expr.AttributeExpr, string) {
				node := objectType("Node")
				setMembers(node, member("index", mapOf(node), lengths(nil, new(2))))
				return &expr.AttributeExpr{Type: node}, "index"
			},
		},
		"self-map-min-length": {
			Build: func() (*expr.AttributeExpr, string) {
				node := objectType("Node")
				setMembers(node, member("index", mapOf(node), lengths(new(1), nil)))
				return &expr.AttributeExpr{Type: node}, "index"
			},
		},
		"named-array-type-max-length": {
			Build: func() (*expr.AttributeExpr, string) {
				node := objectType("Node")
				nodes := &expr.UserTypeExpr{
					TypeName: "Nodes",
					AttributeExpr: &expr.AttributeExpr{
						Type:       arrayOf(node),
						Validation: lengths(new(1), new(3)),
					},
				}
				setMembers(node, member("children", nodes, nil))
				return &expr.AttributeExpr{Type: node}, "children"
			},
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			for _, seed := range []string{"a", "b", "c", "d", "e", "f", "g", "h"} {
				att, recursive := c.Build()
				var first, second any
				require.NotPanics(t, func() {
					first = att.Example(expr.NewRandom(seed))
					second = att.Example(expr.NewRandom(seed))
				}, "seed %q", seed)
				require.Equal(t, first, second, "seed %q", seed)
				object, ok := first.(map[string]any)
				require.True(t, ok, "seed %q: example has type %T", seed, first)
				require.Contains(t, object, recursive, "seed %q", seed)
				requireLengths(t, att, first)
			}
		})
	}
}

// requireLengths checks that every array and map in example satisfies the
// length validations of the attribute that describes it.
func requireLengths(t *testing.T, att *expr.AttributeExpr, example any) {
	t.Helper()
	value := reflect.ValueOf(example)
	if att.Validation != nil && (att.Validation.MinLength != nil || att.Validation.MaxLength != nil) {
		require.Contains(t, []reflect.Kind{reflect.Slice, reflect.Map}, value.Kind(), "example has type %T", example)
		if att.Validation.MinLength != nil {
			require.GreaterOrEqual(t, value.Len(), *att.Validation.MinLength)
		}
		if att.Validation.MaxLength != nil {
			require.LessOrEqual(t, value.Len(), *att.Validation.MaxLength)
		}
	}
	switch dt := att.Type.(type) {
	case expr.UserType:
		requireLengths(t, dt.Attribute(), example)
	case *expr.Object:
		for _, nat := range *dt {
			if member := value.MapIndex(reflect.ValueOf(nat.Name)); member.IsValid() {
				requireLengths(t, nat.Attribute, member.Interface())
			}
		}
	case *expr.Array:
		for i := range value.Len() {
			requireLengths(t, dt.ElemType, value.Index(i).Interface())
		}
	case *expr.Map:
		for _, key := range value.MapKeys() {
			requireLengths(t, dt.ElemType, value.MapIndex(key).Interface())
		}
	}
}

// objectType returns an object user type named name without members.
func objectType(name string) *expr.UserTypeExpr {
	return &expr.UserTypeExpr{TypeName: name, AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{}}}
}

// setMembers sets the members of the object user type ut.
func setMembers(ut *expr.UserTypeExpr, members ...*expr.NamedAttributeExpr) {
	obj := expr.Object(members)
	ut.AttributeExpr.Type = &obj
}

// member returns an object member named name of type dt with validation v.
func member(name string, dt expr.DataType, v *expr.ValidationExpr) *expr.NamedAttributeExpr {
	return &expr.NamedAttributeExpr{Name: name, Attribute: &expr.AttributeExpr{Type: dt, Validation: v}}
}

// arrayOf returns an array of elements of type dt.
func arrayOf(dt expr.DataType) *expr.Array {
	return &expr.Array{ElemType: &expr.AttributeExpr{Type: dt}}
}

// mapOf returns a map of string keys to values of type dt.
func mapOf(dt expr.DataType) *expr.Map {
	return &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: &expr.AttributeExpr{Type: dt}}
}

// lengths returns a validation with the given length bounds.
func lengths(minLength, maxLength *int) *expr.ValidationExpr {
	return &expr.ValidationExpr{MinLength: minLength, MaxLength: maxLength}
}
