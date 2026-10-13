package codegen

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/example"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
)

type exampleCLIServiceData struct {
	Data              *ServiceData
	ServiceImport     string
	DiagnosticsImport string
}

// ExampleCLITransport describes the transport-specific names and paths used by
// a generated example client.
type ExampleCLITransport struct {
	// PathName is the transport directory below the generated package.
	PathName string
	// Filename is the transport example source filename.
	Filename string
	// FunctionName is the transport entry point called by the example main.
	FunctionName string
	// UsagePrefix prefixes the generated usage helper functions.
	UsagePrefix string
	// StreamingConfigDefault is the Go expression for default connection configuration.
	StreamingConfigDefault string
}

// ExampleCLIFiles returns an example client tool HTTP implementation for each
// server expression that hosts an HTTP service.
func ExampleCLIFiles(genpkg string, services *ServicesData) []*codegen.File {
	var files []*codegen.File
	servers := example.NewServersData()
	for _, svr := range services.Root.API.Servers {
		if !hostsTransportService(svr, services.Expressions) {
			continue
		}
		if f := exampleCLIWithCache(genpkg, svr, services, servers, httpExampleCLITransport()); f != nil {
			files = append(files, f)
		}
	}
	return files
}

// ExampleCLI returns an example client tool HTTP implementation for the given
// server expression. It returns nil when the server hosts no HTTP service.
func ExampleCLI(genpkg string, svr *expr.ServerExpr, services *ServicesData) *codegen.File {
	if !hostsTransportService(svr, services.Expressions) {
		return nil
	}
	return exampleCLIWithCache(genpkg, svr, services, example.NewServersData(), httpExampleCLITransport())
}

// ExampleCLIForTransport returns an example client configured for transport.
// It is used by transports that share the HTTP client runtime. It returns nil
// when the server hosts no service of services.
func ExampleCLIForTransport(
	genpkg string,
	svr *expr.ServerExpr,
	services *ServicesData,
	transport ExampleCLITransport,
) *codegen.File {
	if !hostsTransportService(svr, services.Expressions) {
		return nil
	}
	return exampleCLIWithCache(genpkg, svr, services, example.NewServersData(), transport)
}

func exampleCLIWithCache(
	genpkg string,
	svr *expr.ServerExpr,
	services *ServicesData,
	servers example.ServersData,
	transport ExampleCLITransport,
) *codegen.File {
	transport = normalizeExampleCLITransport(transport)
	svrdata := servers.Get(svr, services.Root)
	path := filepath.Join("cmd", svrdata.Dir+"-cli", transport.Filename)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return nil // file already exists, skip it.
	}
	rootPath := "."
	if parent, _, ok := strings.CutLast(genpkg, "/"); ok && parent != "" {
		rootPath = parent
	}
	specs := make([]*codegen.ImportSpec, 0, 15+2*len(services.Expressions.Services))
	specs = append(specs, []*codegen.ImportSpec{
		{Path: "context"},
		{Path: "encoding/json/v2", Name: "json"},
		{Path: "flag"},
		{Path: "fmt"},
		{Path: "net/http"},
		{Path: "net/url"},
		{Path: "os"},
		{Path: "strings"},
		{Path: "time"},
		{Path: "github.com/gorilla/websocket"},
		codegen.LoomImport(""),
		codegen.LoomNamedImport("http", "loomhttp"),
		{Path: genpkg + "/" + transport.PathName + "/cli/" + svrdata.Dir, Name: "cli"},
	}...)
	importScope := codegen.NewNameScope()
	reserveExampleImportNames(importScope, specs)
	exampleServices, serviceImports := exampleCLIServiceImports(genpkg, transport.PathName, services, importScope)
	specs = append(specs, serviceImports...)
	interceptorsPkg := importScope.Unique("interceptors", "ex")
	specs = append(specs, &codegen.ImportSpec{Path: rootPath + "/interceptors", Name: interceptorsPkg})
	apiPkg := importScope.Unique(service.PackageBaseName(services.Root.API.Name), "api")
	specs = append(specs, &codegen.ImportSpec{Path: rootPath, Name: apiPkg})

	plan := buildCLIArgumentPlan(svr, services)
	svcData := make([]exampleCLIServiceData, 0, len(plan.services))
	for _, svc := range plan.services {
		if data, ok := exampleServices[svc.Service.Name]; ok {
			svcData = append(svcData, data)
		}
	}
	sections := []codegen.Section{
		codegen.Header("", "main", specs),
		exampleCLIStartSection(svcData, interceptorsPkg, transport.FunctionName),
		exampleCLIStreamingSection(plan),
		exampleCLIEndSection(plan, apiPkg, transport.StreamingConfigDefault),
		exampleCLIUsageSection(transport.UsagePrefix),
	}
	return &codegen.File{
		Path:      path,
		Sections:  sections,
		SkipExist: true,
	}
}

func exampleCLIServiceImports(
	genpkg, transportPath string, services *ServicesData, importScope *codegen.NameScope,
) (map[string]exampleCLIServiceData, []*codegen.ImportSpec) {
	var specs []*codegen.ImportSpec
	exampleServices := make(map[string]exampleCLIServiceData, len(services.Expressions.Services))
	for _, svc := range services.Expressions.Services {
		data := services.Get(svc.Name())
		serviceImport := importScope.Unique(data.Service.PkgName, "svc")
		diagnosticsImport := ""
		if !emptyDiagnosticFields(serviceDiagnosticFields(data)) {
			diagnosticsImport = importScope.Unique(data.Service.PkgName + "diagnostics")
			specs = append(specs, &codegen.ImportSpec{
				Path: genpkg + "/" + transportPath + "/" + data.Service.PathName,
				Name: diagnosticsImport,
			})
		}
		exampleServices[svc.Name()] = exampleCLIServiceData{Data: data, ServiceImport: serviceImport, DiagnosticsImport: diagnosticsImport}
		specs = append(specs, &codegen.ImportSpec{
			Path: genpkg + "/" + data.Service.PathName,
			Name: serviceImport,
		})
	}
	return exampleServices, specs
}

func httpExampleCLITransport() ExampleCLITransport {
	return ExampleCLITransport{
		PathName:               "http",
		Filename:               "http.go",
		FunctionName:           "doHTTP",
		UsagePrefix:            "http",
		StreamingConfigDefault: "nil",
	}
}

func normalizeExampleCLITransport(transport ExampleCLITransport) ExampleCLITransport {
	defaults := httpExampleCLITransport()
	if transport.PathName == "" {
		transport.PathName = defaults.PathName
	}
	if transport.Filename == "" {
		transport.Filename = defaults.Filename
	}
	if transport.FunctionName == "" {
		transport.FunctionName = defaults.FunctionName
	}
	if transport.UsagePrefix == "" {
		transport.UsagePrefix = defaults.UsagePrefix
	}
	if transport.StreamingConfigDefault == "" {
		transport.StreamingConfigDefault = defaults.StreamingConfigDefault
	}
	return transport
}
