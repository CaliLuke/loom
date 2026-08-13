package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSessionSecurityConflictDoesNotMutateSharedPayload(t *testing.T) {
	obj := &Object{}
	obj.Set("auth", &AttributeExpr{Type: Int})
	shared := &UserTypeExpr{
		AttributeExpr: &AttributeExpr{Type: obj},
		TypeName:      "SharedPayload",
	}
	method := &MethodExpr{
		Name:    "Secure",
		Service: &ServiceExpr{Name: "ConflictService"},
		Payload: &AttributeExpr{Type: shared},
		SessionAuths: []*SessionAuthExpr{{
			Name: "app_session",
			Transports: []*SessionTransportExpr{
				{
					Kind:      SessionCookieTransportKind,
					Scheme:    &SchemeExpr{Kind: APIKeyKind, SchemeName: "cookie"},
					FieldName: "browser_session",
				},
				{
					Kind:      SessionBearerTransportKind,
					Scheme:    &SchemeExpr{Kind: JWTKind, SchemeName: "jwt"},
					FieldName: "auth",
				},
			},
		}},
	}

	verr := method.injectSessionAuthPayloadFields()
	require.NotEmpty(t, verr.Errors)
	require.Same(t, shared, method.Payload.Type)
	require.Nil(t, method.Payload.Find("browser_session"))
	require.Equal(t, Int, method.Payload.Find("auth").Type)
}
