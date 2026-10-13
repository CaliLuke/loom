package service

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service/testdata"
)

func TestMultipleServicesSecurityFixture(t *testing.T) {
	root := codegen.RunDSL(t, testdata.MultipleServicesDSL)
	require.Len(t, root.Services, 3)
	endpoint := root.API.HTTP.Service("ServiceWithJWTAndAPIKey").Endpoint("Method")
	require.Len(t, endpoint.Requirements, 1, "both credentials must remain in one AND requirement")
	require.Len(t, endpoint.Requirements[0].Schemes, 2)
	require.Equal(t, "api_key", endpoint.Requirements[0].Schemes[0].SchemeName)
	require.Equal(t, "jwt", endpoint.Requirements[0].Schemes[1].SchemeName)
	for _, scheme := range endpoint.Requirements[0].Schemes {
		require.Equal(t, "header", scheme.In)
		switch scheme.SchemeName {
		case "api_key":
			require.Equal(t, "X-API-Key", scheme.Name)
		case "jwt":
			require.Equal(t, "Authorization", scheme.Name)
		default:
			t.Errorf("unexpected security scheme %q", scheme.SchemeName)
		}
	}
	apiKeyOnly := root.API.HTTP.Service("ServiceWithAPIKeyAuth").Endpoint("Method")
	require.Len(t, apiKeyOnly.Requirements, 1)
	require.Len(t, apiKeyOnly.Requirements[0].Schemes, 1)
	require.Equal(t, "api_key", apiKeyOnly.Requirements[0].Schemes[0].SchemeName)
	require.Equal(t, "Authorization", apiKeyOnly.Requirements[0].Schemes[0].Name)
	require.Empty(t, root.API.HTTP.Service("ServiceWithNoSecurity").Endpoint("Method").Requirements)
}
