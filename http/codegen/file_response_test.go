package codegen

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/codegentest"
	. "github.com/CaliLuke/loom/dsl"
)

func TestFileResponseHTTPSections(t *testing.T) {
	root := RunHTTPDSL(t, fileResponseHTTPDSL)
	services := CreateHTTPServices(root)
	serverFiles := ServerFiles("gen", services)

	handlers := codegentest.Sections(serverFiles, "server.go", "server-handler-init")
	require.Len(t, handlers, 1)
	handler := codegen.SectionCode(t, handlers[0])
	require.Contains(t, handler, "o := res.(*"+services.Get("files").Service.PkgName+".DownloadFileResponseData)")
	require.Contains(t, handler, "if o.File == nil || o.File.Content == nil")
	require.Contains(t, handler, "if closer, ok := o.File.Content.(io.Closer); ok")
	require.Contains(t, handler, "if err := closer.Close(); err != nil")
	require.Contains(t, handler, "o.File.ServeHTTP(w, r)")
	require.NotContains(t, handler, "io.Copy(w, o.File")
	metadata := strings.Index(handler, "encodeResponse(ctx, w, o.Result)")
	serve := strings.Index(handler, "o.File.ServeHTTP(w, r)")
	require.NotEqual(t, -1, metadata)
	require.Greater(t, serve, metadata)
	require.Contains(t, handler, `w.Header().Set("Content-Type", "application/pdf")`)

	encoders := codegentest.Sections(serverFiles, "encode_decode.go", "response-encoder")
	require.Len(t, encoders, 1)
	encoder := codegen.SectionCode(t, encoders[0])
	require.Contains(t, encoder, `w.Header().Set("Etag", *res.Etag)`)
	require.NotContains(t, encoder, "NewDownloadResponseBody")
	require.NotContains(t, encoder, "enc.Encode(body)")
	require.NotContains(t, encoder, "w.WriteHeader")
	errorEncoders := codegentest.Sections(serverFiles, "encode_decode.go", "error-encoder")
	require.Len(t, errorEncoders, 1)
	errorEncoder := codegen.SectionCode(t, errorEncoders[0])
	require.Contains(t, errorEncoder, "w.WriteHeader(http.StatusNotFound)")

	endpoint := services.Get("files").Endpoints[0]
	client := codegen.SectionCode(t, clientEndpointSection(endpoint))
	require.Contains(t, client, "Body: resp.Body")
	require.Contains(t, client, "Result: res.(*"+services.Get("files").Service.PkgName+".DownloadResult)")

	decoders := codegentest.Sections(ClientFiles("gen", services), "encode_decode.go", "response-decoder")
	require.Len(t, decoders, 1)
	decoder := codegen.SectionCode(t, decoders[0])
	require.Contains(t, decoder, "case http.StatusOK, http.StatusPartialContent, http.StatusNotModified:")
	require.NotContains(t, decoder, "case http.StatusPreconditionFailed:")
	require.NotContains(t, decoder, "case http.StatusRequestedRangeNotSatisfiable:")
}

func fileResponseHTTPDSL() {
	Service("files", func() {
		Method("download", func() {
			Error("not_found")
			Result(func() {
				Attribute("etag", String)
			})
			HTTP(func() {
				GET("/download")
				FileResponse()
				Response(func() {
					ContentType("application/pdf")
					Header("etag:ETag")
				})
				Response("not_found", StatusNotFound)
			})
		})
	})
}
