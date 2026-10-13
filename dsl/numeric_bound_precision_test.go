package dsl_test

import (
	json "encoding/json/v2"
	"strconv"
	"testing"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
	"github.com/stretchr/testify/require"
)

func TestNumericBoundPreservesAuthoredInteger(t *testing.T) {
	values := []struct {
		name  string
		value any
		want  string
	}{
		{"above float exact range", int64(9007199254740993), "9007199254740993"},
		{"maximum signed", int64(9223372036854775807), "9223372036854775807"},
		{"maximum unsigned", uint64(18446744073709551615), "18446744073709551615"},
		{"native unsigned", uint(5), "5"},
		{"maximum native unsigned", ^uint(0), strconv.FormatUint(uint64(^uint(0)), 10)},
		{"negative exact integer", int64(-9007199254740993), "-9007199254740993"},
		{"decimal text", "9007199254740993", "9007199254740993"},
		{"small integer control", 42, "42"},
	}
	bounds := []struct {
		name string
		set  func(any)
		get  func(*expr.ValidationExpr) any
	}{
		{"minimum", Minimum, func(v *expr.ValidationExpr) any {
			return v.Minimum
		}},
		{"maximum", Maximum, func(v *expr.ValidationExpr) any {
			return v.Maximum
		}},
		{"exclusive minimum", ExclusiveMinimum, func(v *expr.ValidationExpr) any {
			return v.ExclusiveMinimum
		}},
		{"exclusive maximum", ExclusiveMaximum, func(v *expr.ValidationExpr) any {
			return v.ExclusiveMaximum
		}},
	}
	for _, bound := range bounds {
		for _, value := range values {
			t.Run(bound.name+"/"+value.name, func(t *testing.T) {
				eval.SetupTestContext(t)
				attribute := &expr.AttributeExpr{Type: Int64}
				switch value.value.(type) {
				case uint64:
					attribute.Type = UInt64
				case uint:
					attribute.Type = UInt
				}
				require.True(t, eval.Execute(func() {
					bound.set(value.value)
				}, attribute))
				require.NotNil(t, attribute.Validation)
				encoded, err := json.Marshal(bound.get(attribute.Validation))
				require.NoError(t, err)
				require.Equal(t, value.want, string(encoded))
			})
		}
	}
}
