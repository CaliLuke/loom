package codegen

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestProblemResponseUsesNativeRepresentation(t *testing.T) {
	endpoint := firstEndpointData(t, testdata.DefaultErrorResponseDSL)
	response := endpoint.Errors[0].Errors[0].Response
	require.Equal(t, "problem", response.SourceVar)
	require.True(t, response.EncodePlan.NeedsProblemSource)
	require.Equal(t, "*loomhttp.ProblemResponse", response.ServerBody[0].Init.ServerArgs[0].TypeRef)
	require.NotContains(t, response.ServerBody[0].Init.ServerCode, "&*")
	code := response.ResultInit.ClientCode
	require.Contains(t, code, "loomhttp.ProblemResponse{")
	require.Contains(t, code, "body.RetryHint.Value()")
	require.Contains(t, code, "loomhttp.ProblemErrorFromBody(problem.Code,")
}

func TestClientResponseValidationEmitsOneRequiredNullablePresenceCheck(t *testing.T) {
	code := renderClientTypesCode(t, func() {
		dsl.Service("members", func() {
			dsl.Method("show", func() {
				dsl.Result(func() {
					dsl.Attribute("company_role", dsl.String, func() {
						dsl.Nullable()
						dsl.Pattern("^[a-z]+$")
					})
					dsl.Required("company_role")
				})
				dsl.HTTP(func() {
					dsl.GET("/members")
					dsl.Response(dsl.StatusOK)
				})
			})
		})
	})

	require.Equal(t, 1, strings.Count(code, `loom.MissingFieldError("company_role", "body")`), code)
	require.Contains(t, code, "if actual, ok := body.CompanyRole.Value(); ok")
	require.Contains(t, code, "ValidatePattern")
}
