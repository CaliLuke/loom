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

func TestSessionPayloadCandidateIdentity(t *testing.T) {
	for _, name := range []string{"nil", "nil type", "empty", "anonymous", "named", "unchanged named"} {
		t.Run(name, func(t *testing.T) {
			var payload *AttributeExpr
			switch name {
			case "nil type":
				payload = &AttributeExpr{}
			case "empty":
				payload = &AttributeExpr{Type: Empty}
			case "anonymous":
				payload = &AttributeExpr{Type: &Object{}}
			case "named", "unchanged named":
				obj := &Object{}
				if name == "unchanged named" {
					obj.Set("auth", &AttributeExpr{Type: String, Meta: MetaExpr{"security:token": {}}})
				}
				payload = &AttributeExpr{Type: &UserTypeExpr{TypeName: "Original", AttributeExpr: &AttributeExpr{Type: obj}}}
			}
			var original DataType
			if payload != nil {
				original = payload.Type
			}
			method := &MethodExpr{
				Name: "Read", Service: &ServiceExpr{Name: "Profiles"}, Payload: payload,
				SessionAuths: []*SessionAuthExpr{{Name: "session", Transports: []*SessionTransportExpr{{
					Kind: SessionBearerTransportKind, Scheme: &SchemeExpr{Kind: JWTKind, SchemeName: "jwt"}, FieldName: "auth",
				}}}},
			}
			require.NotPanics(t, func() {
				require.Empty(t, method.injectSessionAuthPayloadFields().Errors)
			})
			require.NotNil(t, method.Payload)
			require.NotNil(t, method.Payload.Find("auth"))
			require.Equal(t, String, method.Payload.Find("auth").Type)
			require.Contains(t, method.Payload.Find("auth").Meta, "security:token")
			switch name {
			case "named":
				require.NotSame(t, original, method.Payload.Type)
				require.Equal(t, "Original_Read_SessionPayload", method.Payload.Type.Name())
				require.Equal(t, "Original", original.Name())
				require.Nil(t, AsObject(original).Attribute("auth"))
			case "unchanged named":
				require.Same(t, original, method.Payload.Type)
				require.Equal(t, "Original", method.Payload.Type.Name())
			default:
				require.IsType(t, &Object{}, method.Payload.Type)
			}
			require.Equal(t, "Empty", Empty.Name())
			require.Empty(t, *AsObject(Empty))
		})
	}
}
