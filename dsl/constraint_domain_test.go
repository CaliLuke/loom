package dsl_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestConstraintDomainInvalidDesigns(t *testing.T) {
	for _, tc := range []struct {
		name       string
		prepare    func() expr.DataType
		rules      func()
		extend     bool
		diagnostic string
	}{
		{"equal exclusive", func() expr.DataType {
			return Float64
		}, func() {
			ExclusiveMinimum(5)
			ExclusiveMaximum(5)
		}, false, "minimum"},
		{"negative minimum", func() expr.DataType {
			return String
		}, func() {
			MinLength(-3)
		}, false, "length"},
		{"negative maximum", func() expr.DataType {
			return String
		}, func() {
			MaxLength(-1)
		}, false, "length"},
		{"named split numeric", func() expr.DataType {
			return Type("DomainNumber", Float64, func() {
				Minimum(5)
			})
		}, func() {
			ExclusiveMaximum(5)
		}, false, "bound"},
		{"named split length", func() expr.DataType {
			return Type("DomainString", String, func() {
				MinLength(3)
			})
		}, func() {
			MaxLength(2)
		}, false, "length"},
		{"masked negative length", func() expr.DataType {
			return Type("DomainString", String, func() {
				MinLength(-1)
			})
		}, func() {
			MinLength(0)
		}, false, "length"},
		{"extend split numeric", func() expr.DataType {
			return Type("DomainObject", func() {
				Attribute("a", Float64, func() {
					Minimum(5)
				})
			})
		}, func() {
			ExclusiveMaximum(5)
		}, true, "bound"},
		{"extend split length", func() expr.DataType {
			return Type("DomainObject", func() {
				Attribute("a", String, func() {
					MinLength(3)
				})
			})
		}, func() {
			MaxLength(2)
		}, true, "length"},
		{"negative array length", func() expr.DataType {
			return ArrayOf(String)
		}, func() {
			MinLength(-1)
		}, false, "length"},
		{"negative map length", func() expr.DataType {
			return MapOf(String, String)
		}, func() {
			MaxLength(-1)
		}, false, "length"},
		{"negative bytes length", func() expr.DataType {
			return Bytes
		}, func() {
			MaxLength(-1)
		}, false, "length"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, func() {
				typ := tc.prepare()
				Service("domain", func() {
					Method("check", func() {
						Payload(func() {
							if tc.extend {
								Extend(typ)
								typ = expr.AsObject(typ).Attribute("a").Type
							}
							Attribute("a", typ, tc.rules)
						})
						HTTP(func() {
							POST("/check")
						})
					})
				})
			})
			if err != nil {
				require.Contains(t, err.Error(), tc.diagnostic)
			}
		})
	}
}

func TestConstraintDomainValidDesigns(t *testing.T) {
	expr.RunDSL(t, func() {
		number := Type("DomainNumber", Float64, func() {
			Minimum("9007199254740993.125")
		})
		Service("domain", func() {
			Method("check", func() {
				Payload(func() {
					Attribute("equal", number, func() {
						Maximum("9007199254740993.125")
					})
					Attribute("narrow", Float64, func() {
						ExclusiveMinimum("9007199254740993.125")
						ExclusiveMaximum("9007199254740993.126")
					})
					Attribute("empty", String, func() {
						MinLength(0)
						MaxLength(0)
					})
					Attribute("items", ArrayOf(String), func() {
						MinLength(0)
						MaxLength(2)
					})
				})
				HTTP(func() {
					POST("/check")
				})
			})
		})
	})
}
