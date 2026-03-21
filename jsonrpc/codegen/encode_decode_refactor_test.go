package codegen

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"goa.design/goa/v3/codegen"
	"goa.design/goa/v3/dsl"
)

func TestJSONRPCClientEncodeDecodeFileRewrite(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcEncodeDecodeRewriteDSL)
	files := ClientFiles("", CreateJSONRPCServices(root))

	file := requireCodegenFile(t, files, "client", "encode_decode.go")
	sectionNames := sectionNames(file)
	assert.Contains(t, sectionNames, "source-header")
	assert.Contains(t, sectionNames, "jsonrpc-request-encoder")
	assert.Contains(t, sectionNames, "jsonrpc-response-decoder")
	assert.NotContains(t, sectionNames, "request-encoder")
	assert.NotContains(t, sectionNames, "response-decoder")

	code := renderCodegenFile(t, file)
	assert.Contains(t, code, `"github.com/google/uuid"`)
	assert.Contains(t, code, `goa.design/goa/v3/jsonrpc`)
	assert.Contains(t, code, `body := &jsonrpc.Request{`)
	assert.Contains(t, code, `Method:  "sum",`)
}

func TestJSONRPCServerEncodeDecodeFileRewrite(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcEncodeDecodeRewriteDSL)
	files := ServerFiles("", CreateJSONRPCServices(root))

	file := requireCodegenFile(t, files, "server", "encode_decode.go")
	sectionNames := sectionNames(file)
	assert.Contains(t, sectionNames, "source-header")
	assert.Contains(t, sectionNames, "jsonrpc-request-decoder")
	assert.NotContains(t, sectionNames, "request-decoder")
	assert.NotContains(t, sectionNames, "error-encoder")

	code := renderCodegenFile(t, file)
	assert.Contains(t, code, `goa.design/goa/v3/jsonrpc`)
	assert.Contains(t, code, `func(r *http.Request, req *jsonrpc.RawRequest)`)
	assert.Contains(t, code, `r.Body = io.NopCloser(bytes.NewReader(req.Params))`)
}

func requireCodegenFile(t *testing.T, files []*codegen.File, dir, base string) *codegen.File {
	t.Helper()

	for _, file := range files {
		if filepath.Base(file.Path) == base && filepath.Base(filepath.Dir(file.Path)) == dir {
			return file
		}
	}
	require.FailNowf(t, "missing codegen file", "expected %s/%s", dir, base)
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
