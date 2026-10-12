package dsl_test

import (
	"testing"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/stretchr/testify/require"
)

func TestNumericValueDesignAdmission(t *testing.T) {
	for _, tc := range []struct {
		name string
		kind expr.DataType
		raw  any
	}{
		{"int32 overflow", Int32, uint32(1 << 31)},
		{"uint negative", UInt, -5},
		{"float32 overflow", Float32, 1e39},
	} {
		for _, role := range []struct {
			name string
			set  func(any)
		}{
			{"default", Default},
			{"enum", func(value any) {
				Enum(value, 2)
			}},
			{"example", func(value any) {
				Example(value)
			}},
		} {
			t.Run(tc.name+"/"+role.name, func(t *testing.T) {
				expr.RunInvalidDSL(t, func() {
					Service("numbers", func() {
						Method("read", func() {
							Payload(func() {
								Attribute("n", tc.kind, func() {
									role.set(tc.raw)
								})
							})
							HTTP(func() {
								POST("/numbers")
							})
						})
					})
				})
			})
		}
	}
}

func TestNumericValueRedundantBoundControl(t *testing.T) {
	expr.RunDSL(t, func() {
		Type("Redundant", Int32, func() {
			Minimum(-1e12)
			Maximum(1e12)
			Default(1)
			Enum(1, 2)
			Example(2)
		})
	})
}

// Optional object examples retain their partial-documentation contract. This
// correction must not replace their existing shape check with full membership.
func TestNumericValuePartialExampleControls(t *testing.T) {
	for _, raw := range []any{map[string]any{}, map[string]any{"n": uint32(1 << 31)}} {
		expr.RunDSL(t, func() {
			Type("Partial", func() {
				Attribute("n", Int32)
				Required("n")
				Example(raw)
			})
		})
	}
}

func TestNumericValueNestedDeclarations(t *testing.T) {
	for _, role := range []string{"default", "enum"} {
		t.Run(role, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, func() {
				narrow := Type("Narrow", Int32)
				Type("DeclaredValues", func() {
					Attribute("nested", ArrayOf(narrow))
					value := map[string]any{"nested": []uint32{1 << 31}}
					if role == "default" {
						Default(value)
					} else {
						Enum(value)
					}
				})
			})
			require.ErrorContains(t, err, role)
			require.ErrorContains(t, err, "DeclaredValues")
		})
	}
}

func TestNumericValueInheritedPredicateControl(t *testing.T) {
	expr.RunDSL(t, func() {
		base := Type("BaseCarrier", Int32, func() {
			Enum(1, 2)
			Default(1)
		})
		Type("RefinedCarrier", base, func() {
			Minimum(2)
			Default(2)
		})
	})
}
