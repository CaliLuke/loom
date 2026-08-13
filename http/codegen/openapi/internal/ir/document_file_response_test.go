package ir

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	dsl "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

func TestFileResponseProtocolResponses(t *testing.T) {
	root := codegen.RunDSL(t, fileResponseDocumentDSL)
	endpoint := root.API.HTTP.Services[0].HTTPEndpoints[0]
	bodyTypes := BuildBodyTypes(root.API, root.Types, root.ResultTypes)
	operation := buildOperation(
		transportir.BuildEndpoint(endpoint),
		bodyTypes.Services["files"]["download"],
		root.API.ExampleGenerator,
		false,
	)

	require.Equal(t, []string{"200", "206", "304", "412", "416"}, responseStatuses(operation.Responses))
	assertFileBinaryResponse(t, operation.Responses["200"].Value)
	assertFileBinaryResponse(t, operation.Responses["206"].Value)
	assertBinaryResponseMediaType(t, operation.Responses["206"].Value, "multipart/byteranges")
	require.Contains(t, operation.Responses["200"].Value.Headers, "Accept-Ranges")
	require.Contains(t, operation.Responses["200"].Value.Headers, "Content-Length")
	require.Contains(t, operation.Responses["206"].Value.Headers, "Content-Range")
	require.False(t, operation.Responses["206"].Value.Headers["Content-Range"].Value.Required)
	require.Contains(t, operation.Responses["304"].Value.Headers, "ETag")
	require.Contains(t, operation.Responses["304"].Value.Headers, "Last-Modified")
	require.Empty(t, operation.Responses["304"].Value.Content)
	require.Contains(t, operation.Responses["412"].Value.Headers, "ETag")
	require.Empty(t, operation.Responses["412"].Value.Content)
	require.Contains(t, operation.Responses["416"].Value.Headers, "Content-Range")
	require.False(t, operation.Responses["416"].Value.Headers["Content-Range"].Value.Required)
	require.Equal(t, "string", operation.Responses["416"].Value.Content["text/plain"].Schema.Type)
}

func TestFileResponseRequestHeadersAndHeadResponse(t *testing.T) {
	root := codegen.RunDSL(t, fileResponseDocumentDSL)
	endpoint := root.API.HTTP.Services[0].HTTPEndpoints[0]
	bodyTypes := BuildBodyTypes(root.API, root.Types, root.ResultTypes)
	head := buildRouteOperationFromIR(
		transportir.BuildEndpoint(endpoint),
		&transportir.Route{Method: "HEAD"},
		"/download",
		bodyTypes.Services["files"]["download"],
		root.API.ExampleGenerator,
		root.API.Meta,
		false,
	)

	wantHeaders := []string{"Range", "If-Range", "If-Match", "If-None-Match", "If-Modified-Since", "If-Unmodified-Since"}
	for _, name := range wantHeaders {
		require.Equal(t, 1, countHeaderParameters(head.Parameters, name), name)
	}
	for _, response := range head.Responses {
		require.Empty(t, response.Value.Content)
	}
}

func countHeaderParameters(params []*ParameterRef, name string) int {
	count := 0
	for _, param := range params {
		if param.Value != nil && param.Value.In == "header" && param.Value.Name == name {
			count++
		}
	}
	return count
}

func assertFileBinaryResponse(t *testing.T, response *Response) {
	t.Helper()
	require.NotNil(t, response)
	assertBinaryResponseMediaType(t, response, "*/*")
}

func assertBinaryResponseMediaType(t *testing.T, response *Response, contentType string) {
	t.Helper()
	media := response.Content[contentType]
	require.NotNil(t, media)
	require.Equal(t, "string", media.Schema.Type)
	require.Equal(t, "binary", media.Schema.Format)
}

func responseStatuses(responses map[string]*ResponseRef) []string {
	statuses := make([]string, 0, len(responses))
	for status := range responses {
		statuses = append(statuses, status)
	}
	sort.Strings(statuses)
	return statuses
}

func fileResponseDocumentDSL() {
	dsl.Service("files", func() {
		dsl.Method("download", func() {
			dsl.Payload(func() {
				dsl.Attribute("if_match", dsl.String)
			})
			dsl.Result(func() {
				dsl.Attribute("etag", dsl.String)
			})
			dsl.HTTP(func() {
				dsl.GET("/download")
				dsl.HEAD("/download")
				dsl.FileResponse()
				dsl.Header("if_match:If-Match")
				dsl.Response(func() {
					dsl.Header("etag:ETag")
				})
			})
		})
	})
}
