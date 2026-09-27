package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestAttributeExprInheritedRequiredness checks requiredness before and after
// finalization, including mapped names and shared or cyclic reference graphs.
func TestAttributeExprInheritedRequiredness(t *testing.T) {
	typ := func(name string, fields, required []string) *UserTypeExpr {
		object := &Object{}
		for _, field := range fields {
			object.Set(field, &AttributeExpr{Type: String})
		}
		return &UserTypeExpr{TypeName: name, AttributeExpr: &AttributeExpr{Type: object, Validation: &ValidationExpr{Required: required}}}
	}
	for _, tc := range []struct {
		name  string
		build func() *AttributeExpr
		want  []string
	}{
		{"selected reference", func() *AttributeExpr {
			source := typ("Source", []string{"a", "b"}, []string{"a", "b"})
			target := typ("Target", []string{"a:wire"}, nil)
			target.References = []DataType{source}
			return target.Attribute()
		}, []string{"a:wire"}},
		{"extension", func() *AttributeExpr {
			source := typ("Source", []string{"a", "b"}, []string{"a", "b"})
			target := typ("Target", []string{"c"}, []string{"c"})
			target.Bases = []DataType{source}
			return target.Attribute()
		}, []string{"c", "a", "b"}},
		{"named reference", func() *AttributeExpr {
			source := typ("Source", []string{"a", "b"}, []string{"a", "b"})
			target := typ("Target", []string{"a"}, nil)
			target.References = []DataType{source}
			return &AttributeExpr{Type: target}
		}, []string{"a"}},
		{"shared reference", func() *AttributeExpr {
			common := typ("Common", []string{"a", "b"}, []string{"a", "b"})
			left := typ("Left", []string{"a"}, nil)
			left.References = []DataType{common}
			right := typ("Right", []string{"b"}, nil)
			right.References = []DataType{common}
			target := typ("Target", []string{"a", "b"}, nil)
			target.References = []DataType{left, right}
			return target.Attribute()
		}, []string{"a", "b"}},
		{"cycle", func() *AttributeExpr {
			left := typ("Left", []string{"a"}, []string{"a"})
			right := typ("Right", []string{"a"}, nil)
			left.References = []DataType{right}
			right.References = []DataType{left}
			return &AttributeExpr{Type: left}
		}, []string{"a"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			att := tc.build()
			require.Equal(t, tc.want, att.AllRequired())
			att.Finalize()
			require.Equal(t, tc.want, att.AllRequired())
		})
	}
}
