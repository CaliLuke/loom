package codegen

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
)

func TestJSONRPCClientEncodeDecodeFileRewrite(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcEncodeDecodeRewriteDSL)
	files := ClientFiles("", CreateJSONRPCServices(root))

	file := requireEncodeDecodeFile(t, files, "client")
	sectionNames := sectionNames(file)
	assert.Contains(t, sectionNames, "source-header")
	assert.Contains(t, sectionNames, "jsonrpc-request-encoder")
	assert.Contains(t, sectionNames, "jsonrpc-response-decoder")
	assert.NotContains(t, sectionNames, "request-encoder")
	assert.NotContains(t, sectionNames, "response-decoder")

	code := renderCodegenFile(t, file)
	assert.Contains(t, code, `"github.com/google/uuid"`)
	assert.Contains(t, code, `github.com/CaliLuke/loom/jsonrpc`)
	assert.Contains(t, code, `sync/atomic`)
	assert.Contains(t, code, `body := &jsonrpc.Request{`)
	assert.Contains(t, code, `Method:  "sum",`)
}

func TestJSONRPCServerEncodeDecodeFileRewrite(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcEncodeDecodeRewriteDSL)
	files := ServerFiles("", CreateJSONRPCServices(root))

	file := requireEncodeDecodeFile(t, files, "server")
	sectionNames := sectionNames(file)
	assert.Contains(t, sectionNames, "source-header")
	assert.Contains(t, sectionNames, "jsonrpc-request-decoder")
	assert.NotContains(t, sectionNames, "request-decoder")
	assert.NotContains(t, sectionNames, "error-encoder")

	code := renderCodegenFile(t, file)
	assert.Contains(t, code, `github.com/CaliLuke/loom/jsonrpc`)
	assert.Contains(t, code, `"bytes"`)
	assert.Contains(t, code, `"io"`)
	assert.Contains(t, code, `func(r *http.Request, req *jsonrpc.RawRequest)`)
	assert.Contains(t, code, `r.Body = io.NopCloser(bytes.NewReader(req.Params))`)
}

func TestJSONRPCClientDecoderReturnsDecodedUnmappedError(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcEncodeDecodeRewriteDSL)
	files := ClientFiles("", CreateJSONRPCServices(root))

	file := requireEncodeDecodeFile(t, files, "client")
	code := renderCodegenFile(t, file)

	assert.Contains(t, code, `return nil, jresp.Error`)
	assert.NotContains(t, code, `default:
				body, _ := io.ReadAll(resp.Body)`)
}

func TestJSONRPCClientDecoderPropagatesViewedResultConstructorError(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcViewedResultDSL)
	files := ClientFiles("", CreateJSONRPCServices(root))

	file := requireEncodeDecodeFile(t, files, "client")
	code := renderCodegenFile(t, file)

	assert.Contains(t, code, `res, err := servicebodymultipleview.NewResulttypemultipleviews(vres)`)
	assert.Contains(t, code, `return nil, loomhttp.ErrValidationError("ServiceBodyMultipleView", "MethodBodyMultipleView", err)`)
	assert.NotContains(t, code, `res := servicebodymultipleview.NewResulttypemultipleviews(vres)`)
}

func requireEncodeDecodeFile(t *testing.T, files []*codegen.File, dir string) *codegen.File {
	t.Helper()

	for _, file := range files {
		if filepath.Base(file.Path) == "encode_decode.go" && filepath.Base(filepath.Dir(file.Path)) == dir {
			return file
		}
	}
	require.FailNowf(t, "missing codegen file", "expected %s/encode_decode.go", dir)
	return nil
}

func renderCodegenFile(t *testing.T, file *codegen.File) string {
	t.Helper()

	return jsonrpcGeneratedCode(t, []*codegen.File{file})
}

func sectionNames(file *codegen.File) []string {
	names := make([]string, 0, len(file.AllSections()))
	for _, section := range file.AllSections() {
		names = append(names, section.SectionName())
	}
	return names
}

var jsonrpcEncodeDecodeRewriteDSL = func() {
	dsl.API("jsonrpc-encode-decode-rewrite", func() {
		dsl.JSONRPC(func() {})
	})

	dsl.Service("calc", func() {
		dsl.JSONRPC(func() {
			dsl.POST("/rpc")
		})

		dsl.Method("sum", func() {
			dsl.Payload(func() {
				dsl.ID("id")
				dsl.Attribute("a", dsl.Int)
				dsl.Attribute("b", dsl.Int)
				dsl.Required("a", "b")
			})
			dsl.Result(func() {
				dsl.ID("id")
				dsl.Attribute("total", dsl.Int)
			})
			dsl.JSONRPC(func() {})
		})
	})
}

var jsonrpcViewedResultDSL = func() {
	dsl.API("jsonrpc-viewed-result", func() {
		dsl.JSONRPC(func() {})
	})

	resultType := dsl.ResultType("application/vnd.result.multiple.views", func() {
		dsl.TypeName("Resulttypemultipleviews")
		dsl.Attributes(func() {
			dsl.Attribute("a", dsl.String)
			dsl.Attribute("b", dsl.String)
		})
		dsl.View("default", func() {
			dsl.Attribute("a")
			dsl.Attribute("b")
		})
		dsl.View("tiny", func() {
			dsl.Attribute("a")
		})
	})

	dsl.Service("ServiceBodyMultipleView", func() {
		dsl.JSONRPC(func() {
			dsl.POST("/rpc")
		})

		dsl.Method("MethodBodyMultipleView", func() {
			dsl.Result(resultType)
			dsl.JSONRPC(func() {})
		})
	})
}
