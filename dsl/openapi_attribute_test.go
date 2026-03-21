package dsl_test

import (
	"testing"

	. "goa.design/goa/v3/dsl"
	"goa.design/goa/v3/expr"
)

func TestReadOnlyAndWriteOnlyHelpers(t *testing.T) {
	root := expr.RunDSL(t, func() {
		var Credential = Type("Credential", func() {
			Attribute("password", String, func() {
				WriteOnly()
			})
			Attribute("token", String, func() {
				ReadOnly()
			})
		})

		Service("svc", func() {
			Method("show", func() {
				Result(Credential)
				HTTP(func() {
					GET("/")
				})
			})
		})
	})

	cred := root.API.HTTP.Services[0].Endpoint("show").MethodExpr.Result
	password := cred.Find("password")
	token := cred.Find("token")
	if password == nil || token == nil {
		t.Fatalf("expected credential attributes to be present")
	}
	if value, ok := password.Meta.Last("openapi:writeOnly"); !ok || value != "true" {
		t.Fatalf("expected password to be write-only, got %q", value)
	}
	if value, ok := token.Meta.Last("openapi:readOnly"); !ok || value != "true" {
		t.Fatalf("expected token to be read-only, got %q", value)
	}
}
