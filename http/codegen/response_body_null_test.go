package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	. "github.com/CaliLuke/loom/dsl"
)

// TestResponseBodyNullability checks that each client response body occurrence
// carries its own root null policy into the generated decoder call. Parent
// nullability and optionality of a selected field do not admit null.
func TestResponseBodyNullability(t *testing.T) {
	const (
		strict     = "err = decoder(loomhttp.WithNonNullableResponseBody(resp)).Decode(&body)"
		permissive = "err = decoder(resp).Decode(&body)"
	)
	for _, tc := range []struct {
		name   string
		method func(nullableValue, nullableParent any)
		allows bool
	}{
		{"string", func(nullableValue, nullableParent any) {
			Result(String)
			HTTP(func() {
				GET("/")
			})
		}, false},
		{"object", func(nullableValue, nullableParent any) {
			Result(func() {
				Attribute("name", String)
			})
			HTTP(func() {
				GET("/")
			})
		}, false},
		{"nullable", func(nullableValue, nullableParent any) {
			Result(String, func() {
				Nullable()
			})
			HTTP(func() {
				GET("/")
			})
		}, true},
		{"any", func(nullableValue, nullableParent any) {
			Result(Any)
			HTTP(func() {
				GET("/")
			})
		}, true},
		{"nullable alias", func(nullableValue, nullableParent any) {
			Result(nullableValue)
			HTTP(func() {
				GET("/")
			})
		}, true},
		{"required child of nullable parent", func(nullableValue, nullableParent any) {
			Result(nullableParent, func() {
				Nullable()
			})
			HTTP(func() {
				GET("/")
				Response(StatusOK, func() {
					Body("value")
				})
			})
		}, false},
		{"nullable child", func(nullableValue, nullableParent any) {
			Result(func() {
				Attribute("value", String, func() {
					Nullable()
				})
				Required("value")
			})
			HTTP(func() {
				GET("/")
				Response(StatusOK, func() {
					Body("value")
				})
			})
		}, true},
		{"optional child", func(nullableValue, nullableParent any) {
			Result(func() {
				Attribute("value", String)
			})
			HTTP(func() {
				GET("/")
				Response(StatusOK, func() {
					Body("value")
				})
			})
		}, false},
		{"optional nullable child", func(nullableValue, nullableParent any) {
			Result(func() {
				Attribute("value", String, func() {
					Nullable()
				})
			})
			HTTP(func() {
				GET("/")
				Response(StatusOK, func() {
					Body("value")
				})
			})
		}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := RunHTTPDSL(t, func() {
				nullableValue := Type("NullableValue", String, func() {
					Nullable()
				})
				parent := Type("Parent", func() {
					Attribute("value", String)
					Required("value")
				})
				nullableParent := Type("NullableParent", parent, func() {
					Nullable()
				})
				Service("receiver", func() {
					Method("receive", func() {
						tc.method(nullableValue, nullableParent)
					})
				})
			})
			services := CreateHTTPServices(root)
			body := services.Get("receiver").Endpoints[0].Result.Responses[0].ClientBody
			require.NotNil(t, body)
			require.Nil(t, body.Media)
			require.Equal(t, tc.allows, body.AllowsNull)
			file := findFileWithSection(t, ClientFiles("gen", services), "response-decoder")
			source := codegen.SectionCode(t, file.Section("response-decoder")[0])
			if tc.allows {
				require.Contains(t, source, permissive)
				require.NotContains(t, source, "WithNonNullableResponseBody")
			} else {
				require.Contains(t, source, strict)
			}
		})
	}
}

// TestErrorResponseBodyNullability checks that error response bodies share the
// response body null policy with success bodies.
func TestErrorResponseBodyNullability(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		Service("receiver", func() {
			Method("receive", func() {
				Error("strict", String)
				Error("nullable", String, func() {
					Nullable()
				})
				HTTP(func() {
					GET("/")
					Response("strict", StatusBadRequest)
					Response("nullable", StatusConflict)
				})
			})
		})
	})
	services := CreateHTTPServices(root)
	endpoint := services.Get("receiver").Endpoints[0]
	allows := make(map[string]bool)
	for _, group := range endpoint.Errors {
		for _, e := range group.Errors {
			require.NotNil(t, e.Response.ClientBody)
			allows[e.Name] = e.Response.ClientBody.AllowsNull
		}
	}
	require.Equal(t, map[string]bool{"strict": false, "nullable": true}, allows)
}
