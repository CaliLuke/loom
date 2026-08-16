package openapiv3

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

type exampleTarget struct {
	example  any
	examples map[string]*ExampleRef
}

func (target *exampleTarget) setExample(value any) {
	target.example = value
}

func (target *exampleTarget) setExamples(values map[string]*ExampleRef) {
	target.examples = values
}

func TestShouldSuppressOpenAPIExamplesHandlesRecursiveUserTypes(t *testing.T) {
	node := &expr.UserTypeExpr{
		TypeName:      "Node",
		AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{}},
	}
	obj := node.AttributeExpr.Type.(*expr.Object)
	*obj = append(*obj, &expr.NamedAttributeExpr{
		Name:      "child",
		Attribute: &expr.AttributeExpr{Type: node},
	})

	attr := &expr.AttributeExpr{Type: node}
	require.False(t, shouldSuppressOpenAPIExamples(attr, false))
}

func TestExplicitNullExampleMarshalsAsPresentNull(t *testing.T) {
	attribute := &expr.AttributeExpr{
		Type:         expr.String,
		Nullable:     true,
		UserExamples: []*expr.ExampleExpr{{ExplicitNull: true}},
	}
	target := new(exampleTarget)
	initExamples(target, attribute, nil, false)

	encoded, err := json.Marshal(struct {
		Example any `json:"example,omitempty"`
	}{Example: target.example})
	require.NoError(t, err)
	require.JSONEq(t, `{"example":null}`, string(encoded))
}
