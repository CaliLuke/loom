package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestCustomMapKeyAutomaticExample(t *testing.T) {
	for _, shape := range []string{"bytes", "named-bytes", "array", "nested-map", "nested-array"} {
		for _, length := range []int{-1, 0, 2} {
			t.Run(shape+"/"+map[int]string{-1: "automatic", 0: "empty", 2: "bounded"}[length], func(t *testing.T) {
				key := expr.DataType(expr.Bytes)
				if shape == "named-bytes" {
					key = &expr.UserTypeExpr{TypeName: "Key", AttributeExpr: &expr.AttributeExpr{Type: expr.Bytes}}
				} else if shape == "array" {
					key = arrayOf(expr.String)
				}
				var collection expr.DataType = &expr.Map{
					KeyType:  &expr.AttributeExpr{Type: key, Meta: expr.MetaExpr{"struct:field:type": {"[16]byte"}}},
					ElemType: &expr.AttributeExpr{Type: expr.String},
				}
				if shape == "nested-map" {
					collection = mapOf(collection)
				} else if shape == "nested-array" {
					collection = arrayOf(collection)
				}
				attribute := &expr.AttributeExpr{Type: collection}
				if length >= 0 {
					attribute.Validation = lengths(new(length), new(length))
				}
				require.NotPanics(t, func() {
					require.Nil(t, attribute.Example(expr.NewRandom("custom-key")))
				})
			})
		}
	}
}

func TestCustomMapKeyAuthoredExample(t *testing.T) {
	authored := map[[16]byte]string{{1}: "value"}
	attribute := &expr.AttributeExpr{
		Type: &expr.Map{
			KeyType:  &expr.AttributeExpr{Type: expr.Bytes, Meta: expr.MetaExpr{"struct:field:type": {"[16]byte"}}},
			ElemType: &expr.AttributeExpr{Type: expr.String},
		},
		UserExamples: []*expr.ExampleExpr{{Value: authored}},
	}
	require.Equal(t, authored, attribute.Example(expr.NewRandom("custom-key")))
}

func TestMapExampleOmitsNoncomparableDynamicKey(t *testing.T) {
	for _, length := range []int{-1, 2} {
		attribute := &expr.AttributeExpr{Type: &expr.Map{
			KeyType: &expr.AttributeExpr{
				Type:         expr.Any,
				UserExamples: []*expr.ExampleExpr{{Value: []byte("key")}},
			},
			ElemType: &expr.AttributeExpr{Type: expr.String},
		}}
		if length >= 0 {
			attribute.Validation = lengths(new(length), new(length))
		}
		require.NotPanics(t, func() {
			require.Nil(t, attribute.Example(expr.NewRandom("dynamic-key")))
		})
	}
}
