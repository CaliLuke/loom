package codegen

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/example"
	"github.com/CaliLuke/loom/expr"
)

type exampleCLIServiceData struct {
	Data          *ServiceData
	ServiceImport string
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
}

// ExampleCLIFiles returns an example client tool HTTP implementation for each
// server expression.
func ExampleCLIFiles(genpkg string, services *ServicesData) []*codegen.File {
	var files []*codegen.File
	servers := example.NewServersData()
	for _, svr := range services.Root.API.Servers {
		if f := exampleCLIWithCache(genpkg, svr, services, servers, httpExampleCLITransport()); f != nil {
			files = append(files, f)
		}
	}
	return files
}

// ExampleCLI returns an example client tool HTTP implementation for the given
// server expression.
func ExampleCLI(genpkg string, svr *expr.ServerExpr, services *ServicesData) *codegen.File {
	return exampleCLIWithCache(genpkg, svr, services, example.NewServersData(), httpExampleCLITransport())
}

// ExampleCLIForTransport returns an example client configured for transport.
// It is used by transports that share the HTTP client runtime.
func ExampleCLIForTransport(
	genpkg string,
	svr *expr.ServerExpr,
	services *ServicesData,
	transport ExampleCLITransport,
) *codegen.File {
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
	specs := []*codegen.ImportSpec{
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
	}
	importScope := codegen.NewNameScope()
	reserveExampleImportNames(importScope, specs)
	exampleServices := make(map[string]exampleCLIServiceData, len(services.Expressions.Services))
	for _, svc := range services.Expressions.Services {
		data := services.Get(svc.Name())
		serviceImport := importScope.Unique(data.Service.PkgName, "svc")
		exampleServices[svc.Name()] = exampleCLIServiceData{Data: data, ServiceImport: serviceImport}
		specs = append(specs, &codegen.ImportSpec{
			Path: genpkg + "/" + data.Service.PathName,
			Name: serviceImport,
		})
	}
	interceptorsPkg := importScope.Unique("interceptors", "ex")
	specs = append(specs, &codegen.ImportSpec{Path: rootPath + "/interceptors", Name: interceptorsPkg})
	apiPkg := importScope.Unique(strings.ToLower(codegen.Goify(services.Root.API.Name, false)), "api")
	specs = append(specs, &codegen.ImportSpec{Path: rootPath, Name: apiPkg})

	svcData := make([]exampleCLIServiceData, 0, len(svr.Services))
	for _, svc := range svr.Services {
		if data, ok := exampleServices[svc]; ok {
			svcData = append(svcData, data)
		}
	}
	sections := []codegen.Section{
		codegen.Header("", "main", specs),
		exampleCLIStartSection(svcData, interceptorsPkg, transport.FunctionName),
		exampleCLIStreamingSection(svcData),
		exampleCLIEndSection(svcData, apiPkg),
		exampleCLIUsageSection(transport.UsagePrefix),
	}
	return &codegen.File{
		Path:      path,
		Sections:  sections,
		SkipExist: true,
	}
}

func httpExampleCLITransport() ExampleCLITransport {
	return ExampleCLITransport{
		PathName:     "http",
		Filename:     "http.go",
		FunctionName: "doHTTP",
		UsagePrefix:  "http",
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
	return transport
}
