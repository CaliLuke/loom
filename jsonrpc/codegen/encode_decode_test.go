package codegen

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/testutil"
	"github.com/CaliLuke/loom/dsl"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

func TestSharedClientGeneratorEmitsJSONRPCRequestEnvelope(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcEncodeDecodeDSL)
	services := CreateJSONRPCServices(root)
	file := httpcodegen.ClientEncodeDecodeFile("", root.API.JSONRPC.Services[0], services)

	require.NotNil(t, file)
	code := sectionSourceByName(t, file, "request-encoder")
	assert.Contains(t, code, `body := &jsonrpc.Request{`)
	assert.Contains(t, code, `JSONRPC: "2.0"`)
}

func TestSharedServerGeneratorEmitsJSONRPCRequestDecoder(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcEncodeDecodeDSL)
	services := CreateJSONRPCServices(root)
	file := httpcodegen.ServerEncodeDecodeFile("", root.API.JSONRPC.Services[0], services)

	require.NotNil(t, file)
	code := sectionSourceByName(t, file, "request-decoder")
	assert.Contains(t, code, `func(r *http.Request, req *jsonrpc.RawRequest)`)
	assert.Contains(t, code, `r.Body = io.NopCloser(bytes.NewReader(params))`)
}

func TestJSONRPCClientEncodeDecodeFile(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcEncodeDecodeDSL)
	files := ClientFiles("", CreateJSONRPCServices(root))

	file := requireEncodeDecodeFile(t, files, "client")
	sectionNames := sectionNames(file)
	assert.Contains(t, sectionNames, "source-header")
	assert.Contains(t, sectionNames, "jsonrpc-request-encoder")
	assert.Contains(t, sectionNames, "jsonrpc-response-decoder")
	assert.NotContains(t, sectionNames, "request-encoder")
	assert.NotContains(t, sectionNames, "response-decoder")

	code := renderCodegenFile(t, file)
	assert.Contains(t, code, `body := &jsonrpc.Request{`)
	testutil.AssertGo(t, filepath.Join("testdata", "golden", "jsonrpc-client-encode-decode.golden"), code)
}

func TestJSONRPCServerEncodeDecodeFile(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcEncodeDecodeDSL)
	files := ServerFiles("", CreateJSONRPCServices(root))

	file := requireEncodeDecodeFile(t, files, "server")
	sectionNames := sectionNames(file)
	assert.Contains(t, sectionNames, "source-header")
	assert.Contains(t, sectionNames, "jsonrpc-request-decoder")
	assert.NotContains(t, sectionNames, "request-decoder")
	assert.NotContains(t, sectionNames, "error-encoder")

	code := renderCodegenFile(t, file)
	assert.Contains(t, code, `r.Body = io.NopCloser(bytes.NewReader(params))`)
	assert.NotContains(t, code, `bytes.NewReader(req.Params)`)
	testutil.AssertGo(t, filepath.Join("testdata", "golden", "jsonrpc-server-encode-decode.golden"), code)
}

func TestJSONRPCClientDecoderReturnsDecodedUnmappedError(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcEncodeDecodeDSL)
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

	assert.Contains(t, code, `return nil, loomhttp.ErrValidationError("ServiceBodyMultipleView", "MethodBodyMultipleView", err)`)
	assert.NotContains(t, code, `res := servicebodymultipleview.NewResulttypemultipleviews(vres)`)
	testutil.AssertGo(t, filepath.Join("testdata", "golden", "jsonrpc-client-encode-decode-viewed.golden"), code)
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

var jsonrpcEncodeDecodeDSL = func() {
	dsl.API("jsonrpc-encode-decode", func() {
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
