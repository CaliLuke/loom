package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPreserveCanonicalOpenAPITypeName(t *testing.T) {
	attr := &AttributeExpr{
		Type: &UserTypeExpr{
			AttributeExpr: &AttributeExpr{
				Type: &Object{},
				Meta: MetaExpr{"openapi:typename": []string{"AuthSessionResponseBody"}},
			},
			TypeName: "BodyType",
		},
	}

	preserveCanonicalOpenAPITypeName(attr)

	require.Equal(t, "true", attr.Meta["openapi:typename:canonical"][0])
	require.Equal(t, "AuthSessionResponseBody", attr.Meta["openapi:typename"][0])
}

func TestCloneExplicitHTTPBodyRenamesAndSetsTypeUID(t *testing.T) {
	body := &AttributeExpr{
		Type: &UserTypeExpr{
			AttributeExpr: &AttributeExpr{
				Type: &Object{},
				Meta: MetaExpr{"openapi:typename": []string{"AuthSessionResponseBody"}},
			},
			TypeName: "BodyType",
			UID:      "original",
		},
	}

	cloned := cloneExplicitHTTPBody(body, "ShowResponseBody", "ResponseBody", "service#ShowResponseBody")

	require.NotSame(t, body, cloned)
	require.Equal(t, "true", cloned.Meta["openapi:typename:canonical"][0])
	require.Equal(t, "AuthSessionResponseBody", cloned.Meta["openapi:typename"][0])

	ut, ok := cloned.Type.(*UserTypeExpr)
	require.True(t, ok)
	require.Equal(t, "ShowResponseBody", ut.Name())
	require.Equal(t, "service#ShowResponseBody", ut.UID)

	original, ok := body.Type.(*UserTypeExpr)
	require.True(t, ok)
	require.Equal(t, "BodyType", original.Name())
	require.Equal(t, "original", original.UID)
}

func TestCloneExplicitHTTPBodySetsResultTypeUID(t *testing.T) {
	body := &AttributeExpr{
		Type: &ResultTypeExpr{
			UserTypeExpr: &UserTypeExpr{
				AttributeExpr: &AttributeExpr{
					Type: &Object{},
				},
				TypeName: "ResultBody",
				UID:      "result-original",
			},
			Identifier: "application/vnd.test+json",
		},
	}

	cloned := cloneExplicitHTTPBody(body, "ListResponseBody", "ResponseBody", "service#ListResponseBody")

	rt, ok := cloned.Type.(*ResultTypeExpr)
	require.True(t, ok)
	require.Equal(t, "ListResponseBody", rt.Name())
	require.Equal(t, "service#ListResponseBody", rt.UID)
}

func TestUnionToObjectUsesTaggedDiscriminators(t *testing.T) {
	Root = &RootExpr{
		API: &APIExpr{
			ExampleGenerator: &ExampleGenerator{
				Randomizer: NewFakerRandomizer("union-to-object"),
			},
		},
	}

	union := &Union{
		TypeKey:  "action",
		ValueKey: "value",
		Values: []*NamedAttributeExpr{
			{
				Name: "List",
				Attribute: &AttributeExpr{
					Type: &Object{
						&NamedAttributeExpr{Name: "limit", Attribute: &AttributeExpr{Type: Int}},
					},
					Meta: MetaExpr{"oneof:type:tag": []string{"list"}},
				},
			},
			{
				Name: "GetActive",
				Attribute: &AttributeExpr{
					Type: &Object{},
					Meta: MetaExpr{"oneof:type:tag": []string{"get_active"}},
				},
			},
		},
	}

	obj := UnionToObject(&AttributeExpr{Type: union})

	require.NotNil(t, obj)
	require.Equal(t, "object", obj.Type.Name())
	typedObj, ok := obj.Type.(*Object)
	require.True(t, ok)
	typeAttr := typedObj.Attribute("action")
	require.NotNil(t, typeAttr)
	require.NotNil(t, typeAttr.Validation)
	require.Equal(t, []any{"list", "get_active"}, typeAttr.Validation.Values)
	require.Contains(t, typeAttr.Description, `"list"`)
	require.Contains(t, typeAttr.Description, `"get_active"`)
	require.NotContains(t, typeAttr.Description, `"List"`)
	require.NotContains(t, typeAttr.Description, `"GetActive"`)
}
