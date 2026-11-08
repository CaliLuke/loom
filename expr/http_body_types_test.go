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

func TestHTTPResponseBodyNameSuffixFromSelector(t *testing.T) {
	t.Run("captured validation and final body agree", func(t *testing.T) {
		source := &AttributeExpr{Type: &Object{
			&NamedAttributeExpr{Name: "status:code", Attribute: &AttributeExpr{Type: Int}},
			&NamedAttributeExpr{Name: "message", Attribute: &AttributeExpr{Type: String}},
		}, Validation: &ValidationExpr{Required: []string{"status:code", "message"}}}
		r := &HTTPResponseExpr{Status: HTTPStatusSelector{Code: 200}, StatusField: "status", Headers: NewEmptyMappedAttributeExpr()}
		e := &HTTPEndpointExpr{MethodExpr: &MethodExpr{Name: "read", Result: source}, Service: &HTTPServiceExpr{ServiceExpr: &ServiceExpr{Name: "capture"}}, Responses: []*HTTPResponseExpr{r}}
		e.captureResponseBindings()
		require.Len(t, r.bindings.bindings, 2)
		require.Equal(t, "status", r.bindings.bindings[0].location)
		require.Contains(t, responseReconstructionBindings(r), "status")
		before, _ := responseBodySource(source, r)
		require.Nil(t, before.Find("status"))
		r.Finalize(e, source)
		after := httpResponseBody(e, r)
		require.Nil(t, after.Find("status"))
		require.NotNil(t, after.Find("message"))
		require.ElementsMatch(t, before.AllRequired(), after.AllRequired())
		require.ElementsMatch(t, []string{"status", "message"}, r.RequiredFields(""))
	})

	result := &AttributeExpr{Type: &Object{&NamedAttributeExpr{Name: "value", Attribute: &AttributeExpr{Type: String}}}}
	endpoint := &HTTPEndpointExpr{
		MethodExpr: &MethodExpr{Name: "read", Result: result},
		Service:    &HTTPServiceExpr{ServiceExpr: &ServiceExpr{Name: "store"}},
	}
	for _, tc := range []struct {
		key    HTTPStatusSelector
		suffix string
	}{
		{HTTPStatusSelector{Code: 299}, "Status299"},
		{HTTPStatusSelector{Class: Status2XX}, "Status2XX"},
		{HTTPStatusSelector{Class: StatusFallback}, "StatusDefault"},
		{HTTPStatusSelector{Code: 200}, "OK"},
		{HTTPStatusSelector{Code: 201}, "Created"},
	} {
		t.Run(tc.key.OpenAPIKey(), func(t *testing.T) {
			r := &HTTPResponseExpr{Status: tc.key, Headers: NewEmptyMappedAttributeExpr()}
			endpoint.Responses = []*HTTPResponseExpr{r, {Status: HTTPStatusSelector{Code: 200}}}
			body := httpResponseBody(endpoint, r)
			require.Equal(t, "Read"+tc.suffix+"ResponseBody", body.Type.Name())
		})
	}
}
