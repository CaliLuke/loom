package codegen

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

func TestJSONRPCClientCLIFilesUseAdaptationPipeline(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcClientCLIAdaptationDSL)
	files := ClientCLIFiles("", CreateJSONRPCServices(root))

	require.NotEmpty(t, files)

	parseFile := requireFileWithSection(t, files, "parse-endpoint")
	assert.Contains(t, parseFile.Path, filepath.Join("jsonrpc", "cli", "single_host", "cli.go"))

	parseSource := sectionSourceByName(t, parseFile, "parse-endpoint")
	assert.Contains(t, parseSource, "loomhttp.ConnConfigureFunc")
	assert.NotContains(t, parseSource, "ConnConfigurer")
}

func TestJSONRPCExampleCLIFilesUseAdaptationPipeline(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcAdaptationPipelineDSL)
	files := ExampleCLIFiles("", CreateJSONRPCServices(root))

	require.Len(t, files, 1)
	assert.Equal(t, filepath.Join("cmd", "single_host-cli", "jsonrpc.go"), files[0].Path)

	code := renderCodegenFile(t, files[0])
	assert.Contains(t, code, "doJSONRPC")
	assert.Contains(t, code, "jsonrpcUsage")
	assert.NotContains(t, code, "doHTTP")
	assert.NotContains(t, code, "httpUsage")
}

func TestJSONRPCExampleServerFilesUseAdaptationPipeline(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcAdaptationPipelineDSL)
	services := CreateJSONRPCServices(root)
	httpFiles := httpcodegen.ExampleServerFiles("", services)
	files := ExampleServerFiles("", services, httpFiles)

	require.Len(t, files, 1)
	assert.Equal(t, filepath.Join("cmd", "single_host", "jsonrpc.go"), files[0].Path)

	code := renderCodegenFile(t, files[0])
	assert.Contains(t, code, "calcJSONRPCServer")
	assert.Contains(t, code, "calcjssvr.Mount(mux, calcJSONRPCServer)")
	assert.Contains(t, code, "calcJSONRPCServer = calcjssvr.New(")
	assert.Contains(t, code, "context.WithoutCancel(ctx)")
	assert.NotContains(t, code, "context.WithTimeout(context.Background()")
}

func sectionSourceByName(t *testing.T, file *codegen.File, name string) string {
	t.Helper()

	for _, section := range file.AllSections() {
		if section.SectionName() == name {
			return renderSectionSource(section)
		}
	}

	require.FailNowf(t, "missing section", "expected section %q in %s", name, file.Path)
	return ""
}

func requireFileWithSection(t *testing.T, files []*codegen.File, name string) *codegen.File {
	t.Helper()

	for _, file := range files {
		for _, section := range file.AllSections() {
			if section.SectionName() == name {
				return file
			}
		}
	}

	require.FailNowf(t, "missing codegen file", "expected file with section %q", name)
	return nil
}

var jsonrpcAdaptationPipelineDSL = func() {
	dsl.API("jsonrpc-adaptation-pipeline", func() {
		dsl.Server("SingleHost", func() {
			dsl.Services("calc")
			dsl.Host("dev", func() {
				dsl.URI("http://example:8080")
			})
		})
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

var jsonrpcClientCLIAdaptationDSL = func() {
	dsl.API("jsonrpc-client-cli-adaptation-pipeline", func() {
		dsl.Server("SingleHost", func() {
			dsl.Services("calc")
			dsl.Host("dev", func() {
				dsl.URI("http://example:8080")
			})
		})
		dsl.JSONRPC(func() {})
	})

	dsl.Service("calc", func() {
		dsl.JSONRPC(func() {})

		dsl.Method("events", func() {
			dsl.Payload(func() {
				dsl.ID("id")
				dsl.Attribute("name", dsl.String)
				dsl.Required("name")
			})
			dsl.StreamingResult(func() {
				dsl.ID("id")
				dsl.Attribute("message", dsl.String)
			})
			dsl.JSONRPC(func() {})
		})
	})
}
