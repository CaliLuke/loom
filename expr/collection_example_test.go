package expr_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

type collectionRandomizer struct {
	expr.Randomizer
	length int
}

func TestArrayExampleLengthIndependentOfElement(t *testing.T) {
	cases := []struct {
		name   string
		typeOf expr.DataType
	}{
		{"string", expr.String},
		{"bytes", expr.Bytes},
		{"array", arrayOf(expr.String)},
		{"map", mapOf(expr.String)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			for _, elementLength := range []int{0, 1, 2, 5} {
				for _, collectionLength := range []int{0, 1, 4} {
					t.Run(fmt.Sprintf("element-%d/collection-%d", elementLength, collectionLength), func(t *testing.T) {
						array := &expr.Array{ElemType: &expr.AttributeExpr{
							Type:       c.typeOf,
							Validation: lengths(new(elementLength), new(elementLength)),
						}}
						for _, constrained := range []bool{false, true} {
							attribute := &expr.AttributeExpr{Type: array}
							if constrained {
								attribute.Validation = lengths(new(collectionLength), new(collectionLength))
							}
							generator := &expr.ExampleGenerator{Randomizer: collectionRandomizer{
								Randomizer: expr.NewFakerRandomizer("collections"),
								length:     collectionLength,
							}}
							example := reflect.ValueOf(attribute.Example(generator))
							require.True(t, example.IsValid())
							require.Equal(t, collectionLength, example.Len(), "constrained=%t", constrained)
							for i := range example.Len() {
								require.Equal(t, elementLength, example.Index(i).Len(), "constrained=%t", constrained)
							}
						}
					})
				}
			}
		})
	}
}

func TestMapExampleRetainsRecursiveValues(t *testing.T) {
	cases := []struct {
		name  string
		build func() *expr.AttributeExpr
	}{
		{"self", func() *expr.AttributeExpr {
			node := objectType("Node")
			setMembers(node, member("index", mapOf(node), nil))
			return &expr.AttributeExpr{Type: node}
		}},
		{"mutual", func() *expr.AttributeExpr {
			a, b := objectType("A"), objectType("B")
			setMembers(a, member("index", mapOf(b), nil))
			setMembers(b, member("index", mapOf(a), nil))
			return &expr.AttributeExpr{Type: a}
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			for i := range 32 {
				seed := fmt.Sprintf("recursive-map-%d", i)
				attribute := c.build()
				example := attribute.Example(expr.NewRandom(seed))
				require.Equal(t, example, attribute.Example(expr.NewRandom(seed)))
				object := example.(map[string]any)
				index := object["index"].(map[string]map[string]any)
				require.NotEmpty(t, index, "seed %s", seed)
				for _, value := range index {
					if c.name == "mutual" {
						nested := value["index"].(map[string]map[string]any)
						require.NotEmpty(t, nested, "seed %s", seed)
						for _, placeholder := range nested {
							require.NotNil(t, placeholder)
							require.Empty(t, placeholder)
						}
					} else {
						require.NotNil(t, value)
						require.Empty(t, value)
					}
				}
			}
		})
	}
}

func TestArrayExampleRetainsAllocationLimit(t *testing.T) {
	array := arrayOf(expr.String)
	generator := &expr.ExampleGenerator{Randomizer: collectionRandomizer{
		Randomizer: expr.DeterministicRandomizer{},
		length:     1025,
	}}
	require.Nil(t, array.Example(generator))
}

func (r collectionRandomizer) ArrayLength() int {
	return r.length
}
