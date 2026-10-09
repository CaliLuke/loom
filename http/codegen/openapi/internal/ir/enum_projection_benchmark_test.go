package ir

import (
	"fmt"
	"testing"

	"github.com/CaliLuke/loom/expr"
)

func BenchmarkProjectOpenAPIEnumValues(b *testing.B) {
	for _, size := range []int{64, 433} {
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			values := make([]any, size)
			for i := range values {
				values[i] = fmt.Sprintf("Zone/Region%d", i)
			}
			attribute := &expr.AttributeExpr{Type: expr.String, Validation: &expr.ValidationExpr{Values: values}}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				projectOpenAPIValues(attribute, values)
			}
		})
	}
}
