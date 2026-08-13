package openapiv3_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	dsl "github.com/CaliLuke/loom/dsl"
	openapiv3 "github.com/CaliLuke/loom/http/codegen/openapi/v3"
)

func TestRenderedFileResponseProtocolContract(t *testing.T) {
	tests := []struct {
		name        string
		target      string
		wantVersion string
	}{
		{name: "OpenAPI 3.2", wantVersion: openapiv3.OpenAPIVersion},
		{name: "OpenAPI 3.1", target: "3.1", wantVersion: openapiv3.OpenAPICompatibilityVersion},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			artifacts := renderOpenAPIArtifactsForVersion(t, fileResponseOpenAPIDSL, test.target, test.wantVersion)
			spec := decodeOpenAPIJSON(t, artifacts.JSON)
			operation := requireOperation(t, spec, "/download", "get")
			for _, name := range []string{"Range", "If-Range", "If-Match", "If-None-Match", "If-Modified-Since", "If-Unmodified-Since"} {
				requireOperationParameter(t, spec, operation, name, "header")
			}
			responses := requireMap(t, operation["responses"], "file responses")
			require.ElementsMatch(t, []string{"200", "206", "304", "412", "416"}, mapKeys(responses))

			ok := resolveFileResponse(t, spec, responses["200"])
			partial := resolveFileResponse(t, spec, responses["206"])
			notModified := resolveFileResponse(t, spec, responses["304"])
			preconditionFailed := resolveFileResponse(t, spec, responses["412"])
			rangeNotSatisfiable := resolveFileResponse(t, spec, responses["416"])

			assertBinaryMediaType(t, ok, "*/*")
			assertBinaryMediaType(t, partial, "*/*")
			assertBinaryMediaType(t, partial, "multipart/byteranges")
			require.Contains(t, requireMap(t, ok["headers"], "200 headers"), "Accept-Ranges")
			require.Contains(t, requireMap(t, ok["headers"], "200 headers"), "Content-Length")
			require.Contains(t, requireMap(t, partial["headers"], "206 headers"), "Content-Range")
			require.Contains(t, requireMap(t, notModified["headers"], "304 headers"), "ETag")
			require.NotContains(t, notModified, "content")
			require.Contains(t, requireMap(t, preconditionFailed["headers"], "412 headers"), "Last-Modified")
			require.NotContains(t, preconditionFailed, "content")
			require.Contains(t, requireMap(t, rangeNotSatisfiable["headers"], "416 headers"), "Content-Range")
			require.Contains(t, requireMap(t, rangeNotSatisfiable["content"], "416 content"), "text/plain")

			head := requireOperation(t, spec, "/download", "head")
			for _, raw := range requireMap(t, head["responses"], "HEAD responses") {
				response := resolveFileResponse(t, spec, raw)
				require.NotContains(t, response, "content")
			}
		})
	}
}

func TestRenderedFileResponseExplicitContentType(t *testing.T) {
	artifacts := renderOpenAPIArtifacts(t, fileResponseExplicitContentTypeOpenAPIDSL)
	spec := decodeOpenAPIJSON(t, artifacts.JSON)
	operation := requireOperation(t, spec, "/download", "get")
	responses := requireMap(t, operation["responses"], "file responses")
	ok := resolveFileResponse(t, spec, responses["200"])
	partial := resolveFileResponse(t, spec, responses["206"])
	assertBinaryMediaType(t, ok, "application/pdf")
	require.NotContains(t, requireMap(t, ok["content"], "response content"), "*/*")
	assertBinaryMediaType(t, partial, "application/pdf")
	assertBinaryMediaType(t, partial, "multipart/byteranges")
}

func assertBinaryMediaType(t *testing.T, response map[string]any, contentType string) {
	t.Helper()
	content := requireMap(t, response["content"], "response content")
	media := requireMap(t, content[contentType], "response media type")
	schema := requireMap(t, media["schema"], "response schema")
	require.Equal(t, "string", schema["type"])
	require.Equal(t, "binary", schema["format"])
}

func resolveFileResponse(t *testing.T, spec map[string]any, raw any) map[string]any {
	t.Helper()
	response := requireMap(t, raw, "response")
	ref, ok := response["$ref"].(string)
	if !ok {
		return response
	}
	name := strings.TrimPrefix(ref, "#/components/responses/")
	require.NotEqual(t, ref, name, "unexpected response reference %q", ref)
	return requireComponentResponse(t, spec, name)
}

func mapKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}

func fileResponseOpenAPIDSL() {
	fileResponseOpenAPIDSLWithContentType("")
}

func fileResponseExplicitContentTypeOpenAPIDSL() {
	fileResponseOpenAPIDSLWithContentType("application/pdf")
}

func fileResponseOpenAPIDSLWithContentType(contentType string) {
	dsl.API("file-response", func() {
		dsl.Title("File Response API")
		dsl.Description("Serves seekable downloads with HTTP range support.")
		dsl.Version("1.0.0")
		dsl.License(func() {
			dsl.Name("Apache-2.0")
			dsl.URL("https://www.apache.org/licenses/LICENSE-2.0.html")
		})
		dsl.Server("files", func() {
			dsl.Host("production", func() {
				dsl.URI("https://files.example.test")
			})
		})
	})
	dsl.Service("files", func() {
		dsl.Description("Downloads stored files.")
		dsl.Method("download", func() {
			dsl.NoSecurity()
			dsl.Result(func() {
				dsl.Attribute("etag", dsl.String)
			})
			dsl.HTTP(func() {
				dsl.GET("/download")
				dsl.HEAD("/download")
				dsl.FileResponse()
				dsl.Response(func() {
					if contentType != "" {
						dsl.ContentType(contentType)
					}
					dsl.Header("etag:ETag")
				})
			})
		})
	})
}
