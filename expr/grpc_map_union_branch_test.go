package expr

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/eval"
)

func TestGRPCMapUnionBranchShapes(t *testing.T) {
	keyAlias := &UserTypeExpr{TypeName: "Key", AttributeExpr: &AttributeExpr{Type: Int32}}
	choice := &Union{TypeName: "Choice", Values: []*NamedAttributeExpr{
		{Name: "text", Attribute: &AttributeExpr{Type: String}},
		{Name: "number", Attribute: &AttributeExpr{Type: Int}},
	}}
	for _, test := range []struct {
		name  string
		key   DataType
		value DataType
		named bool
		error string
	}{
		{"string", String, Int, true, ""},
		{"boolean", Boolean, String, true, ""},
		{"integer", Int64, String, true, ""},
		{"unsigned", UInt64, String, true, ""},
		{"key alias", keyAlias, String, true, ""},
		{"any value", String, Any, true, ""},
		{"unnamed", String, Int, false, "declare the map with Type"},
		{"float key", Float64, Int, true, "gRPC map keys must be"},
		{"any key", Any, Int, true, "gRPC map keys must be"},
		{"union value", String, choice, true, "is a map value, not supported"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var branch DataType = &Map{KeyType: &AttributeExpr{Type: test.key}, ElemType: &AttributeExpr{Type: test.value}}
			if test.named {
				branch = &UserTypeExpr{TypeName: "Index", AttributeExpr: &AttributeExpr{Type: branch}}
			}
			attribute := &AttributeExpr{Type: &Union{TypeName: "Outer", Values: []*NamedAttributeExpr{
				{Name: "index", Attribute: &AttributeExpr{Type: branch}},
				{Name: "text", Attribute: &AttributeExpr{Type: String}},
			}}}
			var errors eval.ValidationErrors
			validateGRPCMessageShapes(attribute, grpcEndpointForTagValidationTest(), &errors, make(map[*Union]struct{}), make(map[*AttributeExpr]struct{}))
			if test.error == "" {
				require.Empty(t, errors.Errors)
			} else {
				require.Contains(t, errors.Error(), test.error)
			}
		})
	}
}
