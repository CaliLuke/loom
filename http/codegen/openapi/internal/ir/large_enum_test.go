package ir

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestNestedEnumFingerprintScaling(t *testing.T) {
	allocations := func(size int) float64 {
		attribute := nestedEnumAttribute(size)
		return testing.AllocsPerRun(1, func() {
			fingerprintAttribute(attribute, false)
		})
	}
	small, large := allocations(32), allocations(128)
	require.Less(t, large, small*6, "quadrupling enum size must not rebuild the enum graph for every value")
}

func BenchmarkNestedEnumFingerprint(b *testing.B) {
	for _, size := range []int{32, 128, 433, 866} {
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			attribute := nestedEnumAttribute(size)
			b.ReportAllocs()
			for b.Loop() {
				fingerprintAttribute(attribute, false)
			}
		})
	}
}

func nestedEnumAttribute(size int) *expr.AttributeExpr {
	values := make([]any, size)
	for index := range values {
		values[index] = fmt.Sprintf("category-%04d", index)
	}
	category := &expr.AttributeExpr{Type: expr.String, Validation: &expr.ValidationExpr{Values: values}}
	shared := &expr.UserTypeExpr{TypeName: "Shared", AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{
		{Name: "category", Attribute: category},
	}}}
	object := &expr.Object{}
	for index := range 6 {
		*object = append(*object, &expr.NamedAttributeExpr{Name: fmt.Sprintf("response%d", index), Attribute: &expr.AttributeExpr{Type: shared}})
	}
	return &expr.AttributeExpr{Type: object}
}
