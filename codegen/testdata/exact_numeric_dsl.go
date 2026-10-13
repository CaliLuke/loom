package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// ExactNumericPayload composes exact bounds through a named scalar and Extend.
// The same contract exercises generated HTTP, gRPC and JSON-RPC validators.
func ExactNumericPayload() expr.UserType {
	alias := Type("ExactSigned", Int64, func() {
		Minimum("9007199254740993")
	})
	parent := Type("ExactNumericParent", func() {
		Field(1, "signed", alias)
	})
	return Type("ExactNumericPayload", func() {
		Extend(parent)
		Field(1, "signed", alias, func() {
			Maximum("9223372036854775807")
		})
		Field(2, "unsigned", UInt64, func() {
			Maximum("18446744073709551615")
			ExclusiveMinimum("18446744073709551614")
		})
		Field(3, "negative", Int64, func() {
			ExclusiveMinimum("-9223372036854775808")
		})
		Field(4, "upper", Int64, func() {
			ExclusiveMaximum("9007199254740993")
		})
		Field(5, "decimal", Float64, func() {
			Maximum("0.1")
		})
		Field(6, "decimal32", Float32, func() {
			Maximum("0.1")
		})
		Field(7, "tiny", Float64, func() {
			Minimum("1e-1000001")
		})
		Field(8, "control", Int32, func() {
			Minimum(uint(2))
			Maximum(uint(17))
		})
		Field(9, "annotation", Float64, func() {
			Minimum("0.1")
			Default(0.1)
			Enum(0.1, 0.2)
		})
	})
}
