package expr_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// TestExplicitResponseBodyOpenAPITypename checks which OpenAPI type name an
// explicit response body carries. A body declared with a user type claims the
// name of the result type, and a name declared on the body itself wins. An
// inline body that lists result attributes declares a shape of its own, so it
// inherits the result type name only as a hint, like the implicit body.
func TestExplicitResponseBodyOpenAPITypename(t *testing.T) {
	cases := []struct {
		name          string
		resultType    bool
		body          func()
		wantName      string
		wantCanonical bool
	}{
		{
			name:     "inline subset of a type",
			body:     func() { Body(func() { Attribute("a") }) },
			wantName: "RT",
		},
		{
			name:       "inline subset of a result type",
			resultType: true,
			body:       func() { Body(func() { Attribute("a") }) },
			wantName:   "RT",
		},
		{
			name: "inline copy of a type",
			body: func() {
				Body(func() {
					Attribute("a")
					Attribute("b")
				})
			},
			wantName: "RT",
		},
		{
			name: "inline subset with its own name",
			body: func() {
				Body(func() {
					Meta("openapi:typename", "Selected")
					Attribute("a")
				})
			},
			wantName:      "Selected",
			wantCanonical: true,
		},
		{
			name:          "declared type",
			body:          func() { Body(rtForTypenameTest) },
			wantName:      "RT",
			wantCanonical: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := expr.RunDSL(t, func() {
				if tc.resultType {
					rtForTypenameTest = ResultType("application/vnd.rt", "RT", func() {
						Attribute("a", String)
						Attribute("b", String)
					})
				} else {
					rtForTypenameTest = Type("RT", func() {
						Attribute("a", String)
						Attribute("b", String)
					})
				}
				Service("Svc", func() {
					Method("do", func() {
						Result(rtForTypenameTest)
						HTTP(func() {
							POST("/do")
							Response(StatusOK, tc.body)
						})
					})
				})
			})
			body := root.API.HTTP.Services[0].HTTPEndpoints[0].Responses[0].Body
			require.NotNil(t, body)
			ut, ok := body.Type.(expr.UserType)
			require.True(t, ok, "response body must be a user type")

			name, ok := body.Meta.Last("openapi:typename")
			if !ok {
				name, ok = ut.Attribute().Meta.Last("openapi:typename")
			}
			assert.True(t, ok, "response body must carry an OpenAPI type name")
			assert.Equal(t, tc.wantName, name)
			_, canonical := body.Meta["openapi:typename:canonical"]
			assert.Equal(t, tc.wantCanonical, canonical)
		})
	}
}

// rtForTypenameTest holds the result type of the current
// TestExplicitResponseBodyOpenAPITypename case so that a Body DSL can refer
// to it.
var rtForTypenameTest expr.UserType
