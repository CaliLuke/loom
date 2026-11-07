package transportir_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	testcodegen "github.com/CaliLuke/loom/codegen"
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
	"github.com/CaliLuke/loom/internal/httpencoding"
)

func TestMultipartContractTree(t *testing.T) {
	root := testcodegen.RunDSL(t, func() {
		list := Type("SourceList", ArrayOf(String), func() {
			Meta("http:encoding:sourceType", "absent")
		})
		record := Type("Record", func() {
			Attribute("value:shared", list, func() {
				Meta("struct:tag:form", "wire")
				Meta("struct:tag:json", "value")
			})
			Attribute("other:shared", String, func() {
				Meta("struct:tag:form", "other")
				Meta("struct:tag:json", "other")
			})
			Meta("openapi:encoding:shared:contentType", "application/json")
			Meta("openapi:additionalProperties", "false")
		})
		empty := Type("Blank", func() {})
		nested := Type("Nested", OneOf(record, empty), func() {
			Untagged()
		})
		choice := Type("Choice", OneOf(nested, record), func() {
			Untagged()
		})
		Service("Upload", func() {
			Method("Submit", func() {
				Payload(choice)
				HTTP(func() {
					POST("/")
					MultipartRequest()
					OptionalRequestBody()
				})
			})
			Method("Mapped", func() {
				Payload(OneOf(choice, String))
				HTTP(func() {
					POST("/mapped")
					MediaType("multipart/form-data", "Choice")
					MediaType("application/json", "String")
				})
			})
		})
	})
	service := transportir.BuildService(root.API.HTTP.Services[0])
	request := service.Endpoints[0].Request
	require.True(t, request.Multipart)
	require.False(t, request.MustHaveBody)
	require.True(t, request.OptionalBody)
	require.Len(t, request.MultipartEncoding.Branches, 2)
	leaf := request.MultipartEncoding.Branches[0].Branches[0]
	require.Equal(t, expr.ObjectClosed, leaf.ObjectOpenness)
	require.Len(t, leaf.Fields, 2)
	require.Equal(t, "wire", leaf.Fields["value:shared"].Name)
	require.Equal(t, "other", leaf.Fields["other:shared"].Name)
	require.Equal(t, httpencoding.Value, leaf.Fields["value:shared"].Codec.Application)
	require.Equal(t, httpencoding.JSONPart, leaf.Fields["value:shared"].Codec.Kind)
	require.Empty(t, request.MultipartEncoding.Branches[0].Branches[1].Fields)
	mapped := service.Endpoints[1].Request.MediaVariants[0].Multipart
	require.NotNil(t, mapped)
	require.Len(t, mapped.Branches, 2)
	require.Equal(t, "wire", mapped.Branches[0].Branches[0].Fields["value:shared"].Name)
}

func TestMultipartNormalizedSourceType(t *testing.T) {
	for _, form := range []bool{false, true} {
		root := testcodegen.RunDSL(t, func() {
			list := Type("SourceList", ArrayOf(String), func() {
				Meta("http:encoding:sourceType", "absent")
			})
			Service("Upload", func() {
				Method("Submit", func() {
					Payload(func() {
						Attribute("value", list)
						Meta("openapi:encoding:value:contentType", "application/json")
					})
					HTTP(func() {
						POST("/")
						if form {
							FormRequest()
						} else {
							MultipartRequest()
						}
					})
				})
			})
		})
		request := transportir.BuildService(root.API.HTTP.Services[0]).Endpoints[0].Request
		field := (*expr.AsObject(request.Body.Type))[0].Attribute
		require.Equal(t, "absent", expr.HTTPValueShape(field).SourceType)
		if form {
			require.Equal(t, httpencoding.Value, request.Form.Fields["value"].Application)
		} else {
			require.Equal(t, httpencoding.Value, request.MultipartEncoding.Fields["value"].Codec.Application)
		}
	}
}

func TestEmptyMediaBodyPresence(t *testing.T) {
	for _, form := range []bool{false, true} {
		for _, optional := range []bool{false, true} {
			root := testcodegen.RunDSL(t, func() {
				empty := Type("EmptyRecord", func() {})
				Service("Empty", func() {
					Method("Submit", func() {
						Payload(empty)
						HTTP(func() {
							POST("/")
							if form {
								FormRequest()
							} else {
								MultipartRequest()
							}
							if optional {
								OptionalRequestBody()
							}
						})
					})
				})
			})
			request := transportir.BuildService(root.API.HTTP.Services[0]).Endpoints[0].Request
			require.NotEqual(t, expr.Empty, request.Body.Type)
			require.Equal(t, !optional, request.MustHaveBody)
			if form {
				require.NotNil(t, request.Form)
				require.Empty(t, request.Form.Fields)
			} else {
				require.NotNil(t, request.MultipartEncoding)
				require.False(t, request.MultipartEncoding.Custom)
				require.Empty(t, request.MultipartEncoding.Fields)
			}
		}
	}
}
