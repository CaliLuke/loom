package dsl_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestHTTPMediaTypeDSL(t *testing.T) {
	root := expr.RunDSL(t, func() {
		API("media", func() {
		})
		Service("media", func() {
			Method("exchange", func() {
				Payload(func() {
					OneOf("body", func() {
						Attribute("json", Int)
						Attribute("text", String)
					})
					Required("body")
				})
				Result(func() {
					OneOf("body", func() {
						Attribute("json", Int)
						Attribute("text", String)
					})
					Required("body")
				})
				HTTP(func() {
					POST("/")
					Body("body")
					MediaType("application/json", "json")
					MediaType("text/plain", "text")
					Response(StatusOK, func() {
						Body("body")
						MediaType("application/json", "json")
						MediaType("text/plain", "text")
					})
				})
			})
		})
	})
	endpoint := root.API.HTTP.Services[0].HTTPEndpoints[0]
	require.Len(t, endpoint.MediaTypes, 2)
	require.Equal(t, "json", endpoint.MediaTypes[0].Branch)
	require.Len(t, endpoint.Responses[0].MediaTypes, 2)
}

func TestHTTPMediaTypeAdmission(t *testing.T) {
	for _, tc := range []struct {
		name    string
		mapping func()
		want    string
	}{
		{"unknown branch", func() {
			MediaType("application/json", "missing")
			MediaType("text/plain", "text")
		}, "unknown body branch"},
		{"unmapped branch", func() {
			MediaType("application/json", "json")
		}, "no mapping"},
		{"duplicate branch", func() {
			MediaType("application/json", "json")
			MediaType("application/xml", "json")
			MediaType("text/plain", "text")
		}, "more than once"},
		{"overlapping parameters", func() {
			MediaType("application/json; a=1", "json")
			MediaType("application/json; b=2", "text")
		}, "ambiguous media"},
		{"range without header", func() {
			MediaType("application/*", "json")
			MediaType("text/plain", "text")
		}, "mapped Content-Type"},
		{"text number", func() {
			MediaType("text/plain", "json")
			MediaType("application/json", "text")
		}, "String or Bytes"},
		{"static type", func() {
			MediaType("application/json", "json")
			MediaType("text/plain", "text")
			ContentType("application/json")
		}, "without ContentType"},
		{"form scalar", func() {
			MediaType("application/x-www-form-urlencoded", "json")
			MediaType("text/plain", "text")
		}, "require non-nullable objects"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, func() {
				Service("media", func() {
					Method("exchange", func() {
						Payload(func() {
							OneOf("body", func() {
								Attribute("json", Int)
								Attribute("text", String)
							})
						})
						HTTP(func() {
							POST("/")
							Body("body")
							tc.mapping()
						})
					})
				})
			})
			require.ErrorContains(t, err, tc.want)
		})
	}
}

func TestHTTPMediaContentTypeHeaderAdmission(t *testing.T) {
	for _, response := range []string{"request", "response", "error"} {
		for _, media := range []string{"application/json", "application/*"} {
			for _, tc := range []struct {
				name     string
				typ      expr.DataType
				nullable bool
			}{{"integer", Int, false}, {"array", ArrayOf(String), false}, {"nullable", String, true}} {
				t.Run(tc.name+media+response, func(t *testing.T) {
					err := expr.RunInvalidDSL(t, func() {
						value := Type("WithMedia", func() {
							Attribute("content_type", tc.typ, func() {
								if tc.nullable {
									Nullable()
								}
							})
							OneOf("body", func() {
								Attribute("json", Int)
							})
						})
						Service("media", func() {
							Method("exchange", func() {
								Payload(value)
								Result(value)
								Error("bad", value)
								mapping := func() {
									Header("content_type:Content-Type")
									Body("body")
									MediaType(media, "json")
								}
								HTTP(func() {
									POST("/")
									switch response {
									case "error":
										Response("bad", StatusBadRequest, mapping)
									case "response":
										Response(StatusOK, mapping)
									default:
										mapping()
									}
								})
							})
						})
					})
					require.ErrorContains(t, err, "Content-Type")
				})
			}
		}
	}
}

func TestHTTPMediaParameterWildcardIsExact(t *testing.T) {
	expr.RunDSL(t, func() {
		Service("media", func() {
			Method("exchange", func() {
				Payload(func() {
					OneOf("body", func() {
						Attribute("json", Int)
					})
				})
				HTTP(func() {
					POST("/")
					Body("body")
					MediaType(`application/json; profile="*"`, "json")
				})
			})
		})
	})
}

func TestHTTPMediaMultipartPartRequiresConcreteType(t *testing.T) {
	for _, media := range []string{"image/*", "*/*", "image"} {
		t.Run(media, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, func() {
				Service("media", func() {
					Method("exchange", func() {
						Payload(func() {
							OneOf("body", func() {
								Attribute("file", func() {
									Attribute("data", Bytes)
									Meta("openapi:encoding:data:contentType", media)
								})
							})
						})
						HTTP(func() {
							POST("/")
							Body("body")
							MediaType("multipart/form-data", "file")
						})
					})
				})
			})
			require.ErrorContains(t, err, "concrete")
		})
	}
}
