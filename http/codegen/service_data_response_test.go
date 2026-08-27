package codegen

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

func TestBuildProblemServerResponseBodyCodeTrimsPointerRefForLiteral(t *testing.T) {
	code := buildProblemServerResponseBodyCode("*MethodUnauthorizedResponseBody", &transportir.ResponseStatus{
		StatusCode: 401,
		Error: &transportir.Error{
			Attribute: &expr.AttributeExpr{},
		},
	})

	if !strings.Contains(code, "body := &MethodUnauthorizedResponseBody{") {
		t.Fatalf("expected pointer type ref to be trimmed in composite literal, got:\n%s", code)
	}
	if strings.Contains(code, "body := &*MethodUnauthorizedResponseBody{") {
		t.Fatalf("unexpected invalid pointer composite literal, got:\n%s", code)
	}
}

func TestBuildProblemClientResultTransformCodeDereferencesOptionalBodyFields(t *testing.T) {
	code := buildProblemClientResultTransformCode(&transportir.ResponseStatus{StatusCode: 401}, true, nil)

	for _, want := range []string{
		"if body.Code != nil {",
		"code = *body.Code",
		"if body.Detail != nil {",
		"detail = *body.Detail",
		"if body.Instance != nil {",
		"instance = *body.Instance",
		"var retryHint *string",
		"if actual, ok := body.RetryHint.Value(); ok {",
		"retryHint = &actual",
		"v := loomhttp.ProblemErrorFromBody(code, 401, detail, instance, retryHint)",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected generated transform to contain %q, got:\n%s", want, code)
		}
	}
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
