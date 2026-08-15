package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestOpenAPIRequestBodyValidation(t *testing.T) {
	cases := []struct {
		name string
		dsl  func()
		want string
	}{
		{
			name: "requires raw request mode",
			dsl: func() {
				rawRequestBodyEndpoint(func() {
					OpenAPIRequestBody(Bytes, "application/octet-stream", true)
				})
			},
			want: "OpenAPIRequestBody requires SkipRequestBodyEncodeDecode.",
		},
		{
			name: "rejects invalid content type",
			dsl: func() {
				rawRequestBodyEndpoint(func() {
					SkipRequestBodyEncodeDecode()
					OpenAPIRequestBody(Bytes, "not a media type", true)
				})
			},
			want: `OpenAPIRequestBody content type "not a media type" is invalid.`,
		},
		{
			name: "rejects duplicate content types",
			dsl: func() {
				rawRequestBodyEndpoint(func() {
					SkipRequestBodyEncodeDecode()
					OpenAPIRequestBodyTypes(Bytes, []string{"application/json", "application/json"}, true)
				})
			},
			want: `OpenAPIRequestBody content type "application/json" is duplicated.`,
		},
		{
			name: "requires a content type",
			dsl: func() {
				rawRequestBodyEndpoint(func() {
					SkipRequestBodyEncodeDecode()
					OpenAPIRequestBodyTypes(Bytes, nil, true)
				})
			},
			want: "OpenAPIRequestBody requires at least one content type.",
		},
		{
			name: "rejects form mode",
			dsl: func() {
				rawRequestBodyEndpoint(func() {
					FormRequest()
					SkipRequestBodyEncodeDecode()
					OpenAPIRequestBody(Bytes, "application/octet-stream", true)
				})
			},
			want: "HTTP endpoint cannot use OpenAPIRequestBody with FormRequest.",
		},
		{
			name: "rejects multipart mode",
			dsl: func() {
				rawRequestBodyEndpoint(func() {
					MultipartRequest()
					SkipRequestBodyEncodeDecode()
					OpenAPIRequestBody(Bytes, "application/octet-stream", true)
				})
			},
			want: "HTTP endpoint cannot use OpenAPIRequestBody with MultipartRequest.",
		},
		{
			name: "rejects optional typed body mode",
			dsl: func() {
				rawRequestBodyEndpoint(func() {
					Body(func() {
						Attribute("id")
					})
					OptionalRequestBody()
					SkipRequestBodyEncodeDecode()
					OpenAPIRequestBody(Bytes, "application/octet-stream", true)
				})
			},
			want: "HTTP endpoint cannot use OpenAPIRequestBody with OptionalRequestBody.",
		},
		{
			name: "rejects typed body mapping",
			dsl: func() {
				rawRequestBodyEndpoint(func() {
					Body(func() {
						Attribute("id")
					})
					SkipRequestBodyEncodeDecode()
					OpenAPIRequestBody(Bytes, "application/octet-stream", true)
				})
			},
			want: "Cannot define a request body when using SkipRequestBodyEncodeDecode.",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, tc.dsl)
			require.Contains(t, stripValidationLocations(err.Error()), tc.want)
		})
	}
}

func TestOpenAPIRequestBodyTypesRecordsMediaTypes(t *testing.T) {
	root := expr.RunDSL(t, func() {
		rawRequestBodyEndpoint(func() {
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(Bytes, []string{"application/json", "multipart/form-data"}, true)
		})
	})

	endpoint := root.API.HTTP.Services[0].HTTPEndpoints[0]
	require.Equal(t, []string{"application/json", "multipart/form-data"}, endpoint.OpenAPIRequestBodyContentTypes)
}

func rawRequestBodyEndpoint(endpointDSL func()) {
	Service("Service", func() {
		Method("Method", func() {
			Payload(func() {
				Field(1, "id", String)
			})
			HTTP(func() {
				POST("/")
				Header("id:X-ID")
				endpointDSL()
			})
		})
	})
}
