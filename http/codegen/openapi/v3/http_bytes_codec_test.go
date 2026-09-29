package openapiv3

import (
	"bytes"
	"context"
	"encoding/gob"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/CaliLuke/loom/http/codegen/openapi"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/stretchr/testify/require"
)

type httpByteCodecCase struct {
	media string
	codec string
}

// These rows enumerate ResponseEncoder's explicit Content-Type switch,
// including every suffix family and the fallback. They do not infer a codec
// from a generic "non-JSON" media classification.
func httpByteResponseCodecs() []httpByteCodecCase {
	return []httpByteCodecCase{
		{"", "json"}, {"application/json", "json"}, {"application/json; charset=utf-8", "json"},
		{"application/vnd.bytes+json", "json"},
		{"application/xml", "xml"}, {"application/vnd.bytes+xml", "xml"},
		{"application/gob", "gob"}, {"application/vnd.bytes+gob", "gob"},
		{"text/html", "text"}, {"text/plain", "text"}, {"text/plain; charset=utf-8", "text"},
		{"application/vnd.bytes+html", "text"}, {"application/vnd.bytes+txt", "text"},
		{"application/octet-stream", "json"}, {"application/unknown", "json"},
	}
}

func TestHTTPByteResponseCodecSchemas(t *testing.T) {
	for _, version := range []string{"3.1", "3.2"} {
		for _, test := range httpByteResponseCodecs() {
			t.Run(version+"/"+test.media, func(t *testing.T) {
				checkHTTPByteEncoder(t, test.media, "", test.codec)
				root := codegen.RunDSL(t, func() {
					dsl.API("byte-codecs", func() {
						dsl.Meta("openapi:version", version)
					})
					dsl.Service("byte-codecs", func() {
						dsl.Method("bytes", func() {
							dsl.Payload(dsl.Bytes, byteCodecBounds)
							dsl.Result(dsl.Bytes, byteCodecBounds)
							dsl.HTTP(func() {
								dsl.POST("/bytes")
								dsl.Response(dsl.StatusOK, func() {
									if test.media != "" {
										dsl.ContentType(test.media)
									}
								})
							})
						})
					})
				})
				spec := New(root)
				require.NotNil(t, spec)
				media := test.media
				if media == "" {
					media = "application/json"
				}
				content := byteCodecResponseContent(t, spec)
				checkByteCodecSchema(t, spec, content[media].Schema, test.codec == "json")
				request := spec.Paths["/bytes"].Post.RequestBody
				if request.Ref != "" {
					request = spec.Components.RequestBodies[strings.TrimPrefix(request.Ref, "#/components/requestBodies/")]
				}
				checkByteCodecSchema(t, spec, request.Value.Content["application/json"].Schema, true)
			})
		}
	}
}

func TestHTTPByteResponseHeaderDoesNotSelectEncoder(t *testing.T) {
	for _, declared := range []string{"", "application/json", "text/plain"} {
		t.Run(declared, func(t *testing.T) {
			root := codegen.RunDSL(t, byteHeaderCodecDesign(declared))
			spec := New(root)
			require.NotNil(t, spec)
			content := byteCodecResponseContent(t, spec)
			require.Len(t, content, 3)
			for _, media := range []string{"application/json", "text/plain", "application/gob"} {
				for _, accept := range []string{"application/json", "text/plain"} {
					// The generated encoder selects its codec before mapped headers are
					// written. The wire label is not a second encoder selection.
					ctx := context.WithValue(context.Background(), loomhttp.ContentTypeKey, declared)
					ctx = context.WithValue(ctx, loomhttp.AcceptTypeKey, accept)
					out := httptest.NewRecorder()
					encoder := loomhttp.ResponseEncoder(ctx, out)
					out.Header().Set("Content-Type", media)
					require.NoError(t, encoder.Encode([]byte("hi")))
					actual := declared
					if actual == "" {
						actual = accept
					}
					expected := "hi"
					if actual == "application/json" {
						expected = "\"aGk=\"\n"
					}
					require.Equal(t, expected, out.Body.String())
				}
				checkByteCodecSchema(t, spec, content[media].Schema, declared == "application/json")
			}
		})
	}
}

func byteHeaderCodecDesign(declared string) func() {
	return func() {
		dsl.Service("byte-media-header", func() {
			dsl.Method("bytes", func() {
				dsl.Result(func() {
					dsl.Attribute("data", dsl.Bytes, byteCodecBounds)
					dsl.Attribute("media", dsl.String, func() {
						dsl.Enum("application/json", "text/plain", "application/gob")
					})
					dsl.Required("data", "media")
				})
				dsl.HTTP(func() {
					dsl.POST("/bytes")
					dsl.Response(dsl.StatusOK, func() {
						dsl.Header("media:Content-Type")
						dsl.Body("data")
						if declared != "" {
							dsl.ContentType(declared)
						}
					})
				})
			})
		})
	}
}

func TestHTTPByteNegotiatedEncoderDispatch(t *testing.T) {
	for _, test := range []httpByteCodecCase{
		{"", "json"}, {"application/json", "json"}, {"application/xml", "xml"}, {"application/gob", "gob"},
		{"text/html", "text"}, {"text/plain", "text"}, {"text/plain; charset=utf-8", "text"},
		{"application/vnd.bytes+xml", "json"}, {"application/vnd.bytes+gob", "json"},
		{"application/vnd.bytes+txt", "json"}, {"application/unknown", "json"},
	} {
		t.Run(test.media, func(t *testing.T) {
			checkHTTPByteEncoder(t, "", test.media, test.codec)
		})
	}
}

func TestHTTPByteRequestDecoderDispatch(t *testing.T) {
	for _, test := range []httpByteCodecCase{
		{"", "json"}, {"application/json", "json"}, {"application/json; charset=utf-8", "json"},
		{"application/xml", "xml"}, {"application/gob", "gob"}, {"text/html", "text"}, {"text/plain", "text"},
		{"text/plain; charset=utf-8", "text"}, {"application/vnd.bytes+json", "unsupported"},
		{"application/vnd.bytes+xml", "unsupported"}, {"application/vnd.bytes+gob", "unsupported"},
		{"application/vnd.bytes+html", "unsupported"}, {"application/vnd.bytes+txt", "unsupported"},
		{"application/octet-stream", "unsupported"}, {"application/unknown", "unsupported"},
	} {
		t.Run(test.media, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(byteCodecWire(t, test.codec)))
			req.Header.Set("Content-Type", test.media)
			var value []byte
			err := loomhttp.RequestDecoder(req).Decode(&value)
			if test.codec == "unsupported" {
				var problem *loom.ServiceError
				require.ErrorAs(t, err, &problem)
				require.Equal(t, "unsupported_media_type", problem.Name)
				require.Nil(t, value)
			} else {
				require.NoError(t, err)
				require.Equal(t, []byte("hi"), value)
			}
		})
	}
}

func TestHTTPByteRequestEncoderAlwaysJSON(t *testing.T) {
	for _, media := range []string{"", "application/json", "text/plain", "application/xml", "application/gob", "application/vnd.bytes+json"} {
		t.Run(media, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			req.Header.Set("Content-Type", media)
			require.NoError(t, loomhttp.RequestEncoder(req).Encode([]byte("hi")))
			wire, err := io.ReadAll(req.Body)
			require.NoError(t, err)
			require.NoError(t, req.Body.Close())
			require.Equal(t, `"aGk="`, string(wire))
		})
	}
}

func checkHTTPByteEncoder(t *testing.T, media, accept, codec string) {
	t.Helper()
	ctx := context.WithValue(context.Background(), loomhttp.AcceptTypeKey, accept)
	ctx = context.WithValue(ctx, loomhttp.ContentTypeKey, media)
	out := httptest.NewRecorder()
	err := loomhttp.ResponseEncoder(ctx, out).Encode([]byte("hi"))
	if codec == "xml" {
		require.EqualError(t, err, "xml: unsupported type: []uint8")
		require.Empty(t, out.Body.Bytes())
		return
	}
	require.NoError(t, err)
	wire := byteCodecWire(t, codec)
	if codec == "json" {
		wire = append(wire, '\n')
	}
	require.Equal(t, wire, out.Body.Bytes())
	response := out.Result()
	defer func() {
		require.NoError(t, response.Body.Close())
	}()
	var decoded []byte
	require.NoError(t, loomhttp.ResponseDecoder(response).Decode(&decoded))
	require.Equal(t, []byte("hi"), decoded)
}

func TestHTTPByteResponseDecoderDispatch(t *testing.T) {
	for _, test := range append(httpByteResponseCodecs(), httpByteCodecCase{"not a media type", "json"}) {
		t.Run(test.media, func(t *testing.T) {
			response := &http.Response{Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(byteCodecWire(t, test.codec)))}
			response.Header.Set("Content-Type", test.media)
			defer func() {
				require.NoError(t, response.Body.Close())
			}()
			var decoded []byte
			require.NoError(t, loomhttp.ResponseDecoder(response).Decode(&decoded))
			require.Equal(t, []byte("hi"), decoded)
		})
	}
}

func TestHTTPByteMalformedResponseContentType(t *testing.T) {
	ctx := context.WithValue(context.Background(), loomhttp.ContentTypeKey, "not a media type")
	require.Nil(t, loomhttp.ResponseEncoder(ctx, httptest.NewRecorder()))
}

func byteCodecWire(t *testing.T, codec string) []byte {
	t.Helper()
	switch codec {
	case "json":
		return []byte(`"aGk="`)
	case "xml":
		return []byte("<data>hi</data>")
	case "gob":
		var buffer bytes.Buffer
		require.NoError(t, gob.NewEncoder(&buffer).Encode([]byte("hi")))
		return buffer.Bytes()
	default:
		return []byte("hi")
	}
}

func byteCodecBounds() {
	dsl.MinLength(2)
	dsl.MaxLength(2)
}

func byteCodecResponseContent(t *testing.T, spec *OpenAPI) map[string]*MediaType {
	t.Helper()
	response := spec.Paths["/bytes"].Post.Responses["200"]
	if response.Ref != "" {
		response = spec.Components.Responses[strings.TrimPrefix(response.Ref, "#/components/responses/")]
	}
	return response.Value.Content
}

func checkByteCodecSchema(t *testing.T, spec *OpenAPI, schema *openapi.Schema, builtinJSON bool) {
	t.Helper()
	for schema.Ref != "" {
		schema = spec.Components.Schemas[strings.TrimPrefix(schema.Ref, "#/components/schemas/")]
		require.NotNil(t, schema)
	}
	if builtinJSON {
		require.Equal(t, "base64", schema.ContentEncoding)
		require.Empty(t, schema.Format)
		require.Nil(t, schema.MinLength)
		require.Nil(t, schema.MaxLength)
	} else {
		require.Empty(t, schema.ContentEncoding)
		require.Equal(t, "binary", schema.Format)
		require.Equal(t, 2, *schema.MinLength)
		require.Equal(t, 2, *schema.MaxLength)
	}
}
