package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"goa.design/goa/v3/codegen/service"
	"goa.design/goa/v3/http/codegen/testdata"
)

func TestClientRequestBodyValidatorsForUnionBodies(t *testing.T) {
	t.Parallel()

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
				return
			}

			require.Empty(t, body.ValidateDef)
			require.Empty(t, body.ValidateRef)
		})
	}
}
