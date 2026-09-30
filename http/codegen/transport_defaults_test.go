package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestTransportMappingDefaults(t *testing.T) {
	t.Run("OpenAPI specimen", func(t *testing.T) {
		data := CreateHTTPServices(RunHTTPDSL(t, testdata.OpenAPI32FeaturesDSL))
		header := data.Get("catalog").Endpoint("parameters").Payload.Request.Headers[0]
		require.Equal(t, "all", header.DefaultValue)
		require.Equal(t, "string", header.TypeRef)
		require.True(t, header.FieldPointer)
	})
	for _, cookie := range []bool{false, true} {
		name := "header"
		if cookie {
			name = "cookie"
		}
		t.Run(name, func(t *testing.T) {
			root := RunHTTPDSL(t, func() {
				Service("defaults", func() {
					Method("read", func() {
						Payload(func() {
							Attribute("value", String, func() {
								Meta("struct:field:name", "CustomValue")
							})
						})
						HTTP(func() {
							GET("/")
							mapping := Header
							if cookie {
								mapping = Cookie
							}
							mapping("value", String, func() {
								Default("all")
								Meta("openapi:allowReserved", "true")
								MinLength(2)
							})
						})
					})
				})
			})
			request := CreateHTTPServices(root).Get("defaults").Endpoint("read").Payload.Request
			var data *AttributeData
			if cookie {
				data = request.Cookies[0].AttributeData
			} else {
				data = request.Headers[0].AttributeData
			}
			require.Equal(t, "CustomValue", data.FieldName)
			require.Equal(t, "all", data.DefaultValue)
			require.Contains(t, data.Validate, "InvalidLengthError")
			require.Equal(t, "string", data.TypeRef)
			require.True(t, data.FieldPointer)
		})
	}
}

func TestTransportLocationsUseEffectiveNamedDefaults(t *testing.T) {
	for _, location := range []string{"header", "query", "cookie"} {
		t.Run(location, func(t *testing.T) {
			root := RunHTTPDSL(t, func() {
				base := Type("TransportDefaultBase", String, func() {
					Default("base")
				})
				middle := Type("TransportDefaultMiddle", base)
				override := Type("TransportDefaultOverride", middle, func() {
					Default("override")
				})
				Service("defaults", func() {
					Method("read", func() {
						Payload(func() {
							Attribute("inherited", middle)
							Attribute("overridden", override)
						})
						HTTP(func() {
							GET("/")
							switch location {
							case "header":
								Header("inherited:X-Inherited")
								Header("overridden:X-Overridden")
							case "query":
								Param("inherited")
								Param("overridden")
							case "cookie":
								Cookie("inherited:inherited")
								Cookie("overridden:overridden")
							}
						})
					})
				})
			})
			request := CreateHTTPServices(root).Get("defaults").Endpoint("read").Payload.Request
			var attributes []*AttributeData
			switch location {
			case "header":
				for _, header := range request.Headers {
					attributes = append(attributes, header.AttributeData)
				}
			case "query":
				for _, param := range request.QueryParams {
					attributes = append(attributes, param.AttributeData)
				}
			case "cookie":
				for _, cookie := range request.Cookies {
					attributes = append(attributes, cookie.AttributeData)
				}
			}
			require.Len(t, attributes, 2)
			require.Equal(t, "base", attributes[0].DefaultValue)
			require.Equal(t, "override", attributes[1].DefaultValue)
		})
	}
}
