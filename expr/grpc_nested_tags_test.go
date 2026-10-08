package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNestedRPCFieldTags(t *testing.T) {
	tagged := func(name string, typ DataType) *NamedAttributeExpr {
		return &NamedAttributeExpr{Name: name, Attribute: &AttributeExpr{Type: typ, Meta: MetaExpr{"rpc:tag": {"1"}}}}
	}
	for _, c := range []struct {
		name string
		wrap func(DataType) DataType
	}{
		{"object", func(dt DataType) DataType { return dt }},
		{"array", func(dt DataType) DataType { return &Array{ElemType: &AttributeExpr{Type: dt}} }},
		{"map", func(dt DataType) DataType {
			return &Map{KeyType: &AttributeExpr{Type: String}, ElemType: &AttributeExpr{Type: dt}}
		}},
		{"union", func(dt DataType) DataType {
			return &Union{TypeName: "Choice", Values: []*NamedAttributeExpr{tagged("branch", dt)}}
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			endpoint := &GRPCEndpointExpr{MethodExpr: &MethodExpr{Name: "Method"}}
			nested := &UserTypeExpr{TypeName: "Nested", AttributeExpr: &AttributeExpr{Type: &Object{{Name: "value", Attribute: &AttributeExpr{Type: Any}}}}}
			fields := &Object{tagged("nested", c.wrap(nested))}
			err := validateRPCTags(fields, endpoint)
			require.ErrorContains(t, err, `attribute "value" does not have "rpc:tag"`)
			nested.AttributeExpr.Type = &Object{tagged("value", Any)}
			require.Empty(t, validateRPCTags(fields, endpoint).Errors)
			nested.AttributeExpr.Type = &Object{tagged("first", String), tagged("second", String)}
			require.ErrorContains(t, validateRPCTags(fields, endpoint), `field number 1 in attribute "second" already exists`)
			nested.AttributeExpr.Meta = MetaExpr{"struct:field:proto": {"External", "external.proto", "external"}}
			require.Empty(t, validateRPCTags(fields, endpoint).Errors)
		})
	}
}

func TestRecursiveRPCFieldTags(t *testing.T) {
	endpoint := &GRPCEndpointExpr{MethodExpr: &MethodExpr{Name: "Method"}}
	node := &UserTypeExpr{TypeName: "Node", AttributeExpr: &AttributeExpr{}}
	node.Type = &Object{
		{Name: "next", Attribute: &AttributeExpr{Type: node, Meta: MetaExpr{"rpc:tag": {"1"}}}},
		{Name: "value", Attribute: &AttributeExpr{Type: Any}},
	}
	err := validateRPCMessageTags(&AttributeExpr{Type: node}, endpoint)
	require.Len(t, err.Errors, 1)
	require.ErrorContains(t, err, `attribute "value"`)
}

func TestCustomRPCMessageRootIsOpaque(t *testing.T) {
	root := &AttributeExpr{
		Type: &Object{{Name: "external", Attribute: &AttributeExpr{Type: String}}},
		Meta: MetaExpr{"struct:field:proto": {"External", "external.proto", "external"}},
	}
	endpoint := &GRPCEndpointExpr{MethodExpr: &MethodExpr{Name: "Method"}}
	named := &UserTypeExpr{TypeName: "External", AttributeExpr: root}
	for _, att := range []*AttributeExpr{
		root,
		{Type: named},
		{Type: &ResultTypeExpr{UserTypeExpr: named}},
		{Type: &UserTypeExpr{TypeName: "Alias", AttributeExpr: &AttributeExpr{Type: named}}},
	} {
		selected := rpcMessageAttribute(att, nil, NewEmptyMappedAttributeExpr())
		require.Same(t, att, selected)
		require.Empty(t, validateRPCMessageTags(selected, endpoint).Errors)
	}
}
