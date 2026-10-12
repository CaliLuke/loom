package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// TestExtendCompositionRefinement checks the #755 overwrite witness and the
// authored overlay in both DSL orders, without multiplying unrelated shapes.
func TestExtendCompositionRefinement(t *testing.T) {
	for _, extendFirst := range []bool{false, true} {
		t.Run(map[bool]string{false: "attribute-first", true: "extend-first"}[extendFirst], func(t *testing.T) {
			var base, derived expr.UserType
			expr.RunDSL(t, func() {
				base = Type("Base", func() {
					Attribute("n", Int, "base", func() {
						Minimum(0)
						Default(2)
					})
					Required("n")
				})
				derived = Type("Derived", func() {
					if extendFirst {
						Extend(base)
					}
					Attribute("n", Int, "local", func() {
						Maximum(100)
						Default(5)
					})
					if !extendFirst {
						Extend(base)
					}
				})
				Service("numbers", func() {
					Method("derived", func() {
						Payload(derived)
					})
					Method("base", func() {
						Payload(base)
					})
				})
			})
			field := derived.Attribute().Find("n")
			constraints, err := expr.EffectiveConstraintsFor(field)
			require.NoError(t, err)
			if field.Description != "local" {
				t.Errorf("description = %q, want local authored description", field.Description)
			}
			if bound := constraints.UpperBound(); !bound.Present || bound.Value.String() != "100" {
				t.Errorf("effective upper bound = %#v, want 100", bound)
			}
			value, present := constraints.Default()
			require.True(t, present)
			require.Equal(t, 5, value)
			require.Equal(t, *boundForTest(0.0), constraints.LowerBound().Value)
			require.True(t, derived.Attribute().IsRequired("n"))
			require.Equal(t, 2, base.Attribute().Find("n").DefaultValue)
		})
	}
}

// TestExtendCompositionRejectsInvalidRefinement isolates the three admission
// obligations; none requires generator output to detect the bad contract.
func TestExtendCompositionRejectsInvalidRefinement(t *testing.T) {
	for _, test := range []struct {
		name       string
		typ        expr.DataType
		refine     func()
		extendLast bool
	}{
		{"incompatible type", String, func() {

		}, false},
		{"contradictory bounds", Int, func() {
			Maximum(-1)
		}, true},
		{"invalid local default", Int, func() {
			Default(-1)
		}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, func() {
				base := Type("Base", func() {
					Attribute("n", Int, func() {
						Minimum(0)
					})
				})
				derived := Type("Derived", func() {
					if !test.extendLast {
						Extend(base)
					}
					Attribute("n", test.typ, test.refine)
					if test.extendLast {
						Extend(base)
					}
				})
				Service("numbers", func() {
					Method("read", func() {
						Payload(derived)
					})
				})
			})
			require.Contains(t, err.Error(), "n")
			require.Regexp(t, `attribute_composition_test.go:[1-9][0-9]*`, err.Error())
		})
	}
}

// TestExtendCompositionOwnsInheritedFields mutates the completed first copy:
// its source and another consumer must retain their own mutable values.
func TestExtendCompositionOwnsInheritedFields(t *testing.T) {
	var base, first, second expr.UserType
	expr.RunDSL(t, func() {
		base = Type("Base", func() {
			Attribute("name", String, "base", func() {
				MinLength(1)
				Meta("documentation", "base")
			})
		})
		first = Type("First", func() {
			Extend(base)
		})
		second = Type("Second", func() {
			Extend(base)
		})
		Service("copies", func() {
			Method("first", func() {
				Payload(first)
			})
			Method("second", func() {
				Payload(second)
			})
		})
	})
	field := first.Attribute().Find("name")
	field.Description = "changed"
	field.Meta["documentation"][0] = "changed"
	*field.Validation.MinLength = 9
	for _, test := range []struct {
		name string
		typ  expr.UserType
	}{
		{"base", base},
		{"second", second},
	} {
		t.Run(test.name, func(t *testing.T) {
			untouched := test.typ.Attribute().Find("name")
			if untouched.Description != "base" {
				t.Errorf("description changed to %q through another occurrence", untouched.Description)
			}
			if untouched.Meta["documentation"][0] != "base" {
				t.Errorf("metadata changed through another occurrence")
			}
			require.Equal(t, 1, *untouched.Validation.MinLength)
		})
	}
}

// TestExtendCompositionPreservesImplicitNullableInheritance establishes that
// omission of Nullable is not an authored non-null override, in either order.
func TestExtendCompositionPreservesImplicitNullableInheritance(t *testing.T) {
	for _, extendFirst := range []bool{false, true} {
		t.Run(map[bool]string{false: "attribute-first", true: "extend-first"}[extendFirst], func(t *testing.T) {
			var derived expr.UserType
			expr.RunDSL(t, func() {
				base := Type("Base", func() {
					Attribute("name", String, func() {
						Nullable()
					})
				})
				derived = Type("Derived", func() {
					if extendFirst {
						Extend(base)
					}
					Attribute("name", String, func() {
						MinLength(1)
					})
					if !extendFirst {
						Extend(base)
					}
				})
				Service("nullable", func() {
					Method("read", func() {
						Payload(derived)
					})
				})
			})
			require.True(t, expr.AllowsNull(derived.Attribute().Find("name")))
		})
	}
}

// TestExtendCompositionPeerDefaults checks that copying an inherited default
// does not count as an explicit local disambiguation between two base fields.
func TestExtendCompositionPeerDefaults(t *testing.T) {
	for _, test := range []struct {
		name     string
		local    bool
		rejected bool
	}{
		{"inherited conflict", false, true},
		{"explicit local choice", true, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			var derived expr.UserType
			design := func() {
				left := Type("Left", func() {
					Attribute("n", Int, func() {
						Default(1)
					})
				})
				right := Type("Right", func() {
					Attribute("n", Int, func() {
						Default(2)
					})
				})
				derived = Type("Derived", func() {
					Extend(left)
					Attribute("n", Int, func() {
						Maximum(10)
						if test.local {
							Default(3)
						}
					})
					Extend(right)
				})
				Service("defaults", func() {
					Method("read", func() {
						Payload(derived)
					})
				})
			}
			if test.rejected {
				err := expr.RunInvalidDSL(t, design)
				require.Contains(t, err.Error(), "default")
				return
			}
			expr.RunDSL(t, design)
			constraints, err := expr.EffectiveConstraintsFor(derived.Attribute().Find("n"))
			require.NoError(t, err)
			value, present := constraints.Default()
			require.True(t, present)
			require.Equal(t, 3, value)
		})
	}
}

// TestExtendCompositionRejectsInheritedRPCNumberCollision checks #757 at the
// evaluation boundary; the named and inline roots exercise separate ownership.
func TestExtendCompositionRejectsInheritedRPCNumberCollision(t *testing.T) {
	for _, named := range []bool{false, true} {
		t.Run(map[bool]string{false: "inline", true: "named"}[named], func(t *testing.T) {
			err := expr.RunInvalidDSL(t, func() {
				base := Type("Base", func() {
					Field(1, "a", String)
				})
				definition := func() {
					Extend(base)
					Field(1, "c", String)
				}
				var payload any = definition
				if named {
					payload = Type("Derived", definition)
				}
				Service("numbers", func() {
					Method("read", func() {
						Payload(payload)
						GRPC(func() {

						})
					})
				})
			})
			require.Contains(t, err.Error(), "field number 1")
			require.Contains(t, err.Error(), `"a"`)
			require.Contains(t, err.Error(), `"c"`)
			require.Regexp(t, `attribute_composition_test.go:[1-9][0-9]*`, err.Error())
		})
	}
}

// Each admission case names a distinct composition obligation.
func TestExtendCompositionAdmissionBoundaries(t *testing.T) {
	for _, test := range []struct {
		name    string
		design  func()
		failure string
	}{
		{"named descendant", func() {
			a := Type("A", String, func() {
				MinLength(2)
			})
			b := Type("B", a, func() {
				MaxLength(5)
			})
			base := Type("Base", func() {
				Attribute("n", a)
			})
			Type("Derived", func() {
				Extend(base)
				Attribute("n", b)
			})
		}, ""},
		{"unrelated nominal substitution", func() {
			a := Type("A", String)
			b := Type("B", String)
			base := Type("Base", func() {
				Attribute("n", a)
			})
			Type("Derived", func() {
				Extend(base)
				Attribute("n", b)
			})
		}, "incompatible Extend types"},
		{"named nullable widening", func() {
			nullable := Type("NullableString", String, func() {
				Nullable()
			})
			left := Type("Left", func() {
				Attribute("n", String)
			})
			right := Type("Right", func() {
				Attribute("n", nullable)
			})
			Type("Derived", func() {
				Extend(left)
				Extend(right)
			})
		}, "incompatible nullable"},
		{"explicit nullable widening", func() {
			base := Type("Base", func() {
				Attribute("n", String)
			})
			Type("Derived", func() {
				Extend(base)
				Attribute("n", String, func() {
					Nullable()
				})
			})
		}, "incompatible nullable"},
		{"local enum widens", func() {
			base := Type("Base", func() {
				Attribute("n", Int, func() {
					Enum(1, 2)
				})
			})
			Type("Derived", func() {
				Extend(base)
				Attribute("n", Int, func() {
					Enum(1, 3)
				})
			})
		}, "enum member"},
		{"local field number changes identity", func() {
			base := Type("Base", func() {
				Field(1, "n", Int)
			})
			Type("Derived", func() {
				Extend(base)
				Field(2, "n", Int)
			})
		}, `metadata "rpc:tag"`},
		{"equal local default resolves peers", func() {
			a := Type("A", func() {
				Attribute("n", Int, func() {
					Default(1)
				})
			})
			b := Type("B", func() {
				Attribute("n", Int, func() {
					Default(2)
				})
			})
			Type("Derived", func() {
				Extend(a)
				Attribute("n", Int, func() {
					Default(1)
				})
				Extend(b)
			})
		}, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.failure != "" {
				err := expr.RunInvalidDSL(t, test.design)
				require.ErrorContains(t, err, test.failure)
				return
			}
			expr.RunDSL(t, test.design)
		})
	}
}

func TestExtendCompositionNestedConstraints(t *testing.T) {
	var derived expr.UserType
	expr.RunDSL(t, func() {
		base := Type("Base", func() {
			Attribute("ns", ArrayOf(Int, func() {
				Minimum(0)
			}))
		})
		derived = Type("Derived", func() {
			Extend(base)
			Attribute("ns", ArrayOf(Int, func() {
				Maximum(100)
			}))
		})
	})
	constraints, err := expr.EffectiveConstraintsFor(expr.AsArray(derived.Attribute().Find("ns").Type).ElemType)
	require.NoError(t, err)
	require.True(t, constraints.LowerBound().Present)
	require.Equal(t, *boundForTest(0.0), constraints.LowerBound().Value)
	require.Equal(t, *boundForTest(100.0), constraints.UpperBound().Value)
}

func TestExtendCompositionPeerEnumIntersection(t *testing.T) {
	var derived expr.UserType
	expr.RunDSL(t, func() {
		a := Type("A", func() {
			Attribute("n", Int, func() {
				Enum(1, 2)
			})
		})
		b := Type("B", func() {
			Attribute("n", Int, func() {
				Enum(2, 3)
			})
		})
		derived = Type("Derived", func() {
			Extend(a)
			Extend(b)
		})
	})
	constraints, err := expr.EffectiveConstraintsFor(derived.Attribute().Find("n"))
	require.NoError(t, err)
	values, present := constraints.EnumCandidates()
	require.True(t, present)
	require.Equal(t, []any{2}, values)
}

func TestExtendCompositionPreservesEmptySingleton(t *testing.T) {
	for _, emptyFirst := range []bool{false, true} {
		t.Run(map[bool]string{false: "extended-first", true: "empty-first"}[emptyFirst], func(t *testing.T) {
			root := expr.RunDSL(t, func() {
				base := Type("Base", func() {
					Attribute("n", Int)
				})
				Service("example", func() {
					if emptyFirst {
						Method("empty", func() {

						})
					}
					Method("extended", func() {
						Payload(func() {
							Extend(base)
						})
					})
					if !emptyFirst {
						Method("empty", func() {

						})
					}
				})
			})
			require.Empty(t, *expr.AsObject(expr.Empty))
			for _, method := range root.Services[0].Methods {
				if method.Name == "empty" {
					require.Same(t, expr.Empty, method.Payload.Type)
					require.Empty(t, *expr.AsObject(method.Payload.Type))
				} else {
					require.NotNil(t, method.Payload.Find("n"))
				}
			}
		})
	}
}

func TestExtendCompositionOwnsConstraintSources(t *testing.T) {
	var base, derived expr.UserType
	expr.RunDSL(t, func() {
		base = Type("Base", func() {
			Attribute("n", Int, func() {
				Minimum(0)
			})
		})
		derived = Type("Derived", func() {
			Extend(base)
			Attribute("n", Int, func() {
				Maximum(100)
			})
		})
	})
	*base.Attribute().Find("n").Validation.Minimum = *boundForTest(20)
	constraints, err := expr.EffectiveConstraintsFor(derived.Attribute().Find("n"))
	require.NoError(t, err)
	require.Equal(t, *boundForTest(0.0), constraints.LowerBound().Value)
}

func TestExtendCompositionCoalescesEquivalentDefaults(t *testing.T) {
	var derived expr.UserType
	expr.RunDSL(t, func() {
		a := Type("A", func() {
			Attribute("n", Int64, func() {
				Default(int(1))
			})
		})
		b := Type("B", func() {
			Attribute("n", Int64, func() {
				Default(int64(1))
			})
		})
		derived = Type("Derived", func() {
			Extend(a)
			Extend(b)
		})
	})
	constraints, err := expr.EffectiveConstraintsFor(derived.Attribute().Find("n"))
	require.NoError(t, err)
	value, present := constraints.Default()
	require.True(t, present)
	require.EqualValues(t, 1, value)
}
