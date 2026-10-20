package codegen

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

func TestJSONRPCClientCLIFilesUseTransportConfiguration(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcClientCLITransportDSL)
	files := ClientCLIFiles("", CreateJSONRPCServices(root))

	require.NotEmpty(t, files)

	parseFile := requireFileWithSection(t, files, "parse-endpoint")
	assert.Contains(t, parseFile.Path, filepath.Join("jsonrpc", "cli", "single_host", "cli.go"))

	parseSource := sectionSourceByName(t, parseFile, "parse-endpoint")
	assert.Contains(t, parseSource, "loomhttp.WebSocketConfig")
	assert.NotContains(t, parseSource, "ConnConfigurer")
	require.IsType(t, &codegen.JenniferSection{}, sectionByName(t, parseFile, "parse-endpoint"))
}

func TestJSONRPCExampleCLIFilesUseTransportConfiguration(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcTransportGenerationDSL)
	files := ExampleCLIFiles("", CreateJSONRPCServices(root))

	require.Len(t, files, 1)
	assert.Equal(t, filepath.Join("cmd", "single_host-cli", "jsonrpc.go"), files[0].Path)

	code := renderCodegenFile(t, files[0])
	assert.Contains(t, code, "doJSONRPC")
	assert.Contains(t, code, "jsonrpcUsage")
	assert.NotContains(t, code, "doHTTP")
	assert.NotContains(t, code, "httpUsage")
	require.IsType(t, &codegen.JenniferSection{}, sectionByName(t, files[0], "cli-http-usage"))
}

func TestJSONRPCExampleCLIFilesSkipServicesWithoutTransportData(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcTransportAbsentServiceDSL)
	files := ExampleCLIFiles("", CreateJSONRPCServices(root))

	// The server hosting only the transport-absent service gets no JSON-RPC
	// example client.
	require.Len(t, files, 1)
	assert.Equal(t, filepath.Join("cmd", "single_host-cli", "jsonrpc.go"), files[0].Path)
	code := renderCodegenFile(t, files[0])
	assert.Contains(t, code, "doJSONRPC")
	assert.NotContains(t, code, "prompts")
}

func TestJSONRPCFilesSkipServicesWithoutEndpoints(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcTransportAbsentServiceDSL)
	require.Equal(t, []string{"calc"}, serviceNames(root.API.JSONRPC.ServicesWithEndpoints()))
	services := CreateJSONRPCServices(root)

	serverFiles := ServerFiles("example.com/gen", services)
	clientFiles := ClientFiles("example.com/gen", services)
	clientTypeFiles := ClientTypeFiles("example.com/gen", services)
	serverTypeFiles := ServerTypeFiles("example.com/gen", services)
	pathFiles := PathFiles(services)
	httpFiles := httpcodegen.ExampleServerFiles("example.com/gen", services)
	exampleServerFiles := ExampleServerFiles("example.com/gen", services, httpFiles)

	allFiles := append([]*codegen.File(nil), serverFiles...)
	allFiles = append(allFiles, clientFiles...)
	allFiles = append(allFiles, clientTypeFiles...)
	allFiles = append(allFiles, serverTypeFiles...)
	allFiles = append(allFiles, pathFiles...)
	allFiles = append(allFiles, exampleServerFiles...)
	for _, file := range allFiles {
		renderCodegenFile(t, file)
		assert.NotContains(t, file.Path, "prompts")
	}
	serverPaths := make([]string, 0, len(serverFiles))
	for _, file := range serverFiles {
		serverPaths = append(serverPaths, file.Path)
	}
	clientPaths := make([]string, 0, len(clientFiles))
	for _, file := range clientFiles {
		clientPaths = append(clientPaths, file.Path)
	}
	assert.Contains(t, serverPaths, filepath.Join(codegen.Gendir, "jsonrpc", "calc", "server", "server.go"))
	assert.Contains(t, clientPaths, filepath.Join(codegen.Gendir, "jsonrpc", "calc", "client", "client.go"))
}

func TestJSONRPCFilesSkipConfiguredServicesWithoutMethodEndpoints(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcTransportZeroEndpointDSL)
	services := CreateJSONRPCServices(root)

	require.Empty(t, root.API.JSONRPC.ServicesWithEndpoints())
	require.Empty(t, ServerFiles("example.com/gen", services))
	require.Empty(t, ClientFiles("example.com/gen", services))
	require.Empty(t, ClientTypeFiles("example.com/gen", services))
	require.Empty(t, ServerTypeFiles("example.com/gen", services))
	require.Empty(t, PathFiles(services))
	require.Empty(t, SSEServerFiles("example.com/gen", services))
	require.Empty(t, ExampleCLIFiles("example.com/gen", services))

	httpFiles := httpcodegen.ExampleServerFiles("example.com/gen", services)
	require.Empty(t, ExampleServerFiles("example.com/gen", services, httpFiles))
}

func serviceNames(services []*expr.HTTPServiceExpr) []string {
	names := make([]string, 0, len(services))
	for _, service := range services {
		names = append(names, service.Name())
	}
	return names
}

func TestJSONRPCExampleServerFilesUseTransportConfiguration(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcTransportGenerationDSL)
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

	return renderSectionSource(sectionByName(t, file, name))
}

func sectionByName(t *testing.T, file *codegen.File, name string) codegen.Section {
	t.Helper()

	for _, section := range file.AllSections() {
		if section.SectionName() == name {
			return section
		}
	}

	require.FailNowf(t, "missing section", "expected section %q in %s", name, file.Path)
	return nil
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

var jsonrpcTransportGenerationDSL = func() {
	dsl.API("jsonrpc-transport-generation", func() {
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

var jsonrpcClientCLITransportDSL = func() {
	dsl.API("jsonrpc-client-cli-transport", func() {
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

var jsonrpcTransportAbsentServiceDSL = func() {
	dsl.API("jsonrpc-transport-absent-service", func() {
		dsl.Server("SingleHost", func() {
			dsl.Services("prompts", "calc")
			dsl.Host("dev", func() {
				dsl.URI("http://example:8080")
			})
		})
		dsl.Server("PromptsOnly", func() {
			dsl.Services("prompts")
			dsl.Host("dev", func() {
				dsl.URI("http://example:8081")
			})
		})
		dsl.JSONRPC(func() {})
	})

	dsl.Service("calc", func() {
		dsl.JSONRPC(func() {
			dsl.POST("/rpc")
		})
		dsl.Method("add", func() {
			dsl.Payload(func() {
				dsl.Attribute("a", dsl.Int)
			})
			dsl.Result(dsl.Int)
			dsl.JSONRPC(func() {})
		})
	})

	dsl.Service("prompts", func() {
		dsl.JSONRPC(func() {
			dsl.POST("/rpc")
		})
		dsl.Method("dynamic", func() {
			dsl.Payload(func() {
				dsl.Attribute("name", dsl.String)
			})
			dsl.Result(dsl.String)
		})
	})
}

var jsonrpcTransportZeroEndpointDSL = func() {
	dsl.API("jsonrpc-zero-endpoint", func() {
		dsl.Server("PromptsOnly", func() {
			dsl.Services("prompts")
			dsl.Host("dev", func() {
				dsl.URI("http://example:8081")
			})
		})
		dsl.JSONRPC(func() {})
	})

	dsl.Service("prompts", func() {
		dsl.JSONRPC(func() {
			dsl.POST("/rpc")
		})
		dsl.Method("dynamic", func() {
			dsl.Payload(func() {
				dsl.Attribute("name", dsl.String)
			})
			dsl.Result(dsl.String)
		})
	})
}
