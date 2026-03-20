package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPreserveCanonicalOpenAPITypeName(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

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
