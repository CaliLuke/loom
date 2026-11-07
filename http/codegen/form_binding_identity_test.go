package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

func TestFormCaptureDistinctBindingIdentity(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		Service("identity", func() {
			Method("send", func() {
				Payload(func() {
					Attribute("first", String, func() {
						Meta("struct:tag:form", "one")
						Meta("struct:tag:json", "jsonOne")
					})
					Attribute("second", String, func() {
						Meta("struct:tag:form", "two")
						Meta("struct:tag:json", "jsonTwo")
					})
					Required("first", "second")
				})
				HTTP(func() {
					POST("/")
					FormRequest()
					Body(func() {
						Attribute("first:shared")
						Attribute("second:shared")
					})
					Response(StatusNoContent)
				})
			})
		})
	})
	services := CreateHTTPServices(root)
	fields := services.Get("identity").Endpoints[0].Payload.Request.OrdinaryFormFields
	names := make([]string, 0, len(fields))
	for _, field := range fields {
		names = append(names, field.HTTPName)
	}
	require.ElementsMatch(t, []string{"one", "two"}, names)
}
