package dsl_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestAuthorizationDSL(t *testing.T) {
	cases := []struct {
		name   string
		method func(*expr.AuthorizationExpr)
		want   string
	}{
		{"missing classification", func(_ *expr.AuthorizationExpr) {
		}, "authorization classification"},
		{"exempt", func(_ *expr.AuthorizationExpr) {
			NoAccessCheck("Caller profile")
		}, ""},
		{"bound", func(a *expr.AuthorizationExpr) {
			Authorize(a, func() {
				Bind("id", "id")
			})
		}, ""},
		{"missing binding", func(a *expr.AuthorizationExpr) {
			Authorize(a)
		}, "binding"},
		{"missing field", func(a *expr.AuthorizationExpr) {
			Authorize(a, func() {
				Bind("id", "missing")
			})
		}, "missing"},
		{"wrong type", func(a *expr.AuthorizationExpr) {
			Authorize(a, func() {
				Bind("id", "count")
			})
		}, "type"},
		{"conflicting exemption", func(a *expr.AuthorizationExpr) {
			Authorize(a, func() {
				Bind("id", "id")
			})
			NoAccessCheck("Exempt")
		}, "exemption"},
		{"empty exemption", func(_ *expr.AuthorizationExpr) {
			NoAccessCheck("")
		}, "reason"},
		{"missing case", func(a *expr.AuthorizationExpr) {
			AuthorizeBy("action", func() {
				AuthorizationCase("read", func() {
					Authorize(a, func() {
						Bind("id", "id")
					})
				})
			})
		}, "write"},
		{"exhaustive cases", func(a *expr.AuthorizationExpr) {
			AuthorizeBy("action", func() {
				AuthorizationCase("read", func() {
					NoAccessCheck("Own profile")
				})
				AuthorizationCase("write", func() {
					Authorize(a, func() {
						Bind("id", "id")
					})
				})
			})
		}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := runDSL(t, func() {
				input := Type("DocumentRef", func() {
					Attribute("id", String)
					Required("id")
				})
				a := Authorization("document.edit", input)
				Service("documents", func() {
					StrictAuthorization()
					Method("update", func() {
						Payload(func() {
							Attribute("id", String)
							Attribute("count", Int)
							Attribute("action", String, func() {
								Enum("read", "write")
							})
							Required("id", "count", "action")
						})
						tc.method(a)
					})
				})
			})
			if tc.want == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.want)
			}
		})
	}
}
