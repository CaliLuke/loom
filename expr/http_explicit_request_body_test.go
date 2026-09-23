package expr_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// TestExplicitInlineRequestBodyIsNamedUserType checks that an explicit inline
// object request body becomes a named user type, like an implicit request
// body and an explicit inline response body, while other explicit bodies keep
// their type. Documentation generators unwrap the inline object.
func TestExplicitInlineRequestBodyIsNamedUserType(t *testing.T) {
	cases := []struct {
		name     string
		body     func()
		wantName string
		wantUser bool
		inline   bool
		required []string
	}{
		{
			name: "inline selection",
			body: func() {
				Body(func() {
					Attribute("item")
					Attribute("name")
					Required("name")
				})
			},
			wantName: "DoRequestBody",
			wantUser: true,
			inline:   true,
			required: []string{"name"},
		},
		{
			name: "inline object attribute",
			body: func() {
				Body("inline")
			},
			wantName: "DoRequestBody",
			wantUser: true,
			inline:   true,
		},
		{
			name: "user type attribute",
			body: func() {
				Body("item")
			},
			wantName: "DoRequestBody",
			wantUser: true,
		},
		{
			name: "primitive attribute",
			body: func() {
				Body("name")
			},
			wantName: expr.String.Name(),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := expr.RunDSL(t, func() {
				item := Type("Item", func() {
					Attribute("x", String)
				})
				Service("Svc", func() {
					Method("do", func() {
						Payload(func() {
							Attribute("item", item)
							Attribute("inline", func() {
								Attribute("y", String)
							})
							Attribute("name", String)
							Attribute("q", String)
							Required("name")
						})
						HTTP(func() {
							POST("/do")
							Param("q")
							tc.body()
						})
					})
				})
			})
			body := root.API.HTTP.Services[0].HTTPEndpoints[0].Body
			require.NotNil(t, body)
			_, isUser := body.Type.(expr.UserType)
			assert.Equal(t, tc.wantUser, isUser)
			assert.Equal(t, tc.wantName, body.Type.Name())
			if isUser {
				assert.Equal(t, "Svc#DoRequestBody", body.Type.(expr.UserType).ID())
			}
			for _, name := range tc.required {
				assert.True(t, body.IsRequired(name), "required %s", name)
			}
			_, ok := body.Meta["http:body"]
			assert.True(t, ok, "http:body meta must stay on the body attribute")

			unwrapped := expr.UnwrapInlineHTTPBody(body)
			if !tc.inline {
				assert.Same(t, body, unwrapped)
				return
			}
			assert.IsType(t, &expr.Object{}, unwrapped.Type)
			assert.Equal(t, body.Meta, unwrapped.Meta)
			assert.Same(t, body.Validation, unwrapped.Validation)
			wrapper := body.Type.(expr.UserType).Attribute()
			for _, key := range []string{"openapi:typename", "openapi:typename:canonical"} {
				_, ok := wrapper.Meta[key]
				assert.False(t, ok, "wrapper must not carry %s", key)
			}
		})
	}
}
