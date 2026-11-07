package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
)

func TestTextUnionMatcherCapturesClosure(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		record := dsl.Type("Record", func() {
			dsl.Attribute("value", dsl.String)
			dsl.Required("value")
		})
		inner := dsl.Type("Inner", dsl.OneOf(record, dsl.Boolean), func() {
			dsl.Untagged()
			dsl.Meta("openapi:unevaluatedProperties", "false")
		})
		outer := dsl.Type("Outer", dsl.OneOf(inner, dsl.Int), dsl.Untagged)
		dsl.Service("closure", func() {
			dsl.Method("read", func() {
				dsl.Payload(func() {
					dsl.Attribute("value", outer, func() {
						dsl.Meta("openapi:unevaluatedProperties", "false")
					})
					dsl.Required("value")
				})
				dsl.HTTP(func() {
					dsl.GET("/")
					dsl.Param("value")
				})
			})
		})
	})
	services := CreateHTTPServices(root)
	element := services.Get("closure").Endpoints[0].Payload.Request.QueryParams[0].Element
	got := renderTextUnionMatcher(element, "query")
	require.Contains(t, got, `MatchUntaggedText("value", branches, true)`)
	require.Contains(t, got, `ComposeTextUnion("value", branches0_, true,`)
	require.Contains(t, got, `Annotations: leaf0_0.Annotations`)
}

func TestTextUnionMatcherUsesScalarBranchStorage(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		first := dsl.Type("First", dsl.String, func() {
			dsl.Enum("first")
		})
		second := dsl.Type("Second", dsl.String, func() {
			dsl.Pattern("^second$")
		})
		choice := dsl.Type("Choice", dsl.OneOf(first, second), dsl.Untagged)
		dsl.Service("carriers", func() {
			dsl.Method("read", func() {
				dsl.Payload(func() {
					dsl.Attribute("value", choice)
					dsl.Required("value")
				})
				dsl.HTTP(func() {
					dsl.GET("/")
					dsl.Param("value")
				})
			})
		})
	})
	services := CreateHTTPServices(root)
	element := services.Get("carriers").Endpoints[0].Payload.Request.QueryParams[0].Element
	for _, pointer := range []bool{false, true} {
		element.context.Pointer = pointer
		got := renderTextUnionMatcher(element, "query")
		require.NotContains(t, got, "if v != nil", "branch candidates are values regardless of the enclosing field mode")
		require.NotContains(t, got, "string(*v)")
		require.Contains(t, got, `string(v) == "first"`)
		require.Contains(t, got, `ValidatePattern("value", string(v), "^second$")`)
	}
}
