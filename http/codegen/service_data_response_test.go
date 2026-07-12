package codegen

import (
	"strings"
	"testing"

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
		"v := loomhttp.ProblemErrorFromBody(code, 401, detail, instance, body.RetryHint)",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected generated transform to contain %q, got:\n%s", want, code)
		}
	}
}
