package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestUntaggedTextLocationAdmission(t *testing.T) {
	for _, location := range []string{"path", "query", "header", "cookie", "response header"} {
		t.Run(location, func(t *testing.T) {
			expr.RunDSL(t, func() {
				choice := Type("Choice", OneOf(func() {
					Untagged()
					Attribute("Number", Int)
					Attribute("Items", ArrayOf(Int))
				}))
				Service("choices", func() {
					Method("echo", func() {
						Payload(func() {
							Attribute("value", choice)
						})
						Result(func() {
							Attribute("value", choice)
						})
						HTTP(func() {
							if location == "path" {
								GET("/{value}")
							} else {
								GET("/")
							}
							switch location {
							case "path":
								Param("value", func() {
									Meta("openapi:style", "label")
									Meta("openapi:explode", "true")
								})
							case "query":
								Param("value", func() {
									Meta("openapi:explode", "false")
								})
							case "header":
								Header("value")
							case "cookie":
								Cookie("value", func() {
									Meta("openapi:explode", "false")
								})
							case "response header":
								Response(StatusOK, func() {
									Header("value")
								})
							}
						})
					})
				})
			})
		})
	}
	for _, tc := range []struct {
		name, style string
		second      bool
		failure     string
	}{
		{"deep scalar", "deepObject", false, "branch 0"},
		{"two residual bindings", "form", true, "overlapping parameter namespaces"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, func() {
				dictionary := Type("Dictionary", MapOf(String, Int))
				choice := Type("Choice", OneOf(Int, dictionary), Untagged)
				Service("choices", func() {
					Method("read", func() {
						Payload(func() {
							Attribute("value", choice)
							Attribute("second", dictionary)
						})
						HTTP(func() {
							GET("/")
							Param("value", func() {
								Meta("openapi:style", tc.style)
							})
							if tc.second {
								Param("second", func() {
									Meta("openapi:style", "form")
								})
							}
						})
					})
				})
			})
			require.ErrorContains(t, err, tc.failure)
		})
	}
}
