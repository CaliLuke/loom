package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestClientRequestBodyValidatorsForUnionBodies(t *testing.T) {
	cases := []struct {
		name            string
		dsl             func()
		expectValidator bool
		validateRef     string
	}{
		{
			name:            "validated object body with primitive union field",
			dsl:             testdata.PayloadBodyUnionValidateDSL,
			expectValidator: true,
			validateRef:     "err = ValidateMethodBodyUnionValidateRequestBody(&body)",
		},
		{
			name:            "validated object body with user union field",
			dsl:             testdata.PayloadBodyUnionUserValidateDSL,
			expectValidator: true,
			validateRef:     "err = ValidateMethodBodyUnionUserValidateRequestBody(&body)",
		},
		{
			name:            "validated object body without union field",
			dsl:             testdata.PayloadBodyQueryUserValidateDSL,
			expectValidator: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := RunHTTPDSL(t, c.dsl)
			require.Len(t, root.API.HTTP.Services, 1)

			httpServices := NewServicesData(service.NewServicesData(root), root.API.HTTP)
			sd := httpServices.Get(root.API.HTTP.Services[0].Name())
			require.NotNil(t, sd)
			require.Len(t, sd.Endpoints, 1)

			body := sd.Endpoints[0].Payload.Request.ClientBody
			require.NotNil(t, body)

			if c.expectValidator {
				require.NotEmpty(t, body.ValidateDef)
				require.Equal(t, c.validateRef, body.ValidateRef)
			} else {
				require.Empty(t, body.ValidateDef)
				require.Empty(t, body.ValidateRef)
			}
		})
	}
}

func TestClientRequestBodyValidatorStubForUnionObjectBranchTypes(t *testing.T) {
	root := RunHTTPDSL(t, oauthFormRequestUnionDSL)
	require.Len(t, root.API.HTTP.Services, 1)

	httpServices := NewServicesData(service.NewServicesData(root), root.API.HTTP)
	sd := httpServices.Get(root.API.HTTP.Services[0].Name())
	require.NotNil(t, sd)

	stubs := make(map[string]*TypeData)
	for _, bodyType := range sd.ClientBodyAttributeTypes {
		stubs[bodyType.Name] = bodyType
	}

	authCode := stubs["AuthorizationCodeGrantRequestBody"]
	require.NotNil(t, authCode)
	require.Equal(t, "// no validations", authCode.ValidateDef)
	require.Equal(t, "err = ValidateAuthorizationCodeGrantRequestBody(v)", authCode.ValidateRef)

	refresh := stubs["RefreshTokenGrantRequestBody"]
	require.NotNil(t, refresh)
	require.Equal(t, "// no validations", refresh.ValidateDef)
	require.Equal(t, "err = ValidateRefreshTokenGrantRequestBody(v)", refresh.ValidateRef)
}

func TestAttributeTypeDataEmitsNoOpValidatorStubForTaggedClientRequestBodyType(t *testing.T) {
	tagged := &expr.UserTypeExpr{
		TypeName: "SingleAction",
		AttributeExpr: &expr.AttributeExpr{
			Type: &expr.Object{
				{
					Name: "value",
					Attribute: &expr.AttributeExpr{
						Type: expr.String,
					},
				},
			},
			Meta: expr.MetaExpr{"oneof:type:tag": []string{"single"}},
		},
	}
	sd := &ServiceData{
		Scope:           codegen.NewNameScope(),
		ServerTypeNames: make(map[string]bool),
		ClientTypeNames: make(map[string]bool),
	}
	sds := &ServicesData{
		ServicesData: &service.ServicesData{Root: expr.Root},
	}

	data := sds.attributeTypeData(tagged, true, false, false, true, sd)
	require.NotNil(t, data)
	require.Equal(t, "// no validations", data.ValidateDef)
	require.Equal(t, "err = ValidateSingleAction(v)", data.ValidateRef)
}

func TestAttributeTypeDataSkipsNoOpValidatorStubForUntaggedClientRequestBodyType(t *testing.T) {
	untagged := &expr.UserTypeExpr{
		TypeName: "SingleAction",
		AttributeExpr: &expr.AttributeExpr{
			Type: &expr.Object{
				{
					Name: "value",
					Attribute: &expr.AttributeExpr{
						Type: expr.String,
					},
				},
			},
		},
	}
	sd := &ServiceData{
		Scope:           codegen.NewNameScope(),
		ServerTypeNames: make(map[string]bool),
		ClientTypeNames: make(map[string]bool),
	}
	sds := &ServicesData{
		ServicesData: &service.ServicesData{Root: expr.Root},
	}

	data := sds.attributeTypeData(untagged, true, false, false, true, sd)
	require.NotNil(t, data)
	require.Empty(t, data.ValidateDef)
	require.Empty(t, data.ValidateRef)
}
