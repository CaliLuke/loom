package codegen

import (
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/example"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/naming"
)

// exampleServerServiceData retains the bindings allocated for one example file.
// The transport data is shared and remains unchanged by file-specific aliases.
type exampleServerServiceData struct {
	data           *ServiceData
	serviceImport  string
	serverImport   string
	protobufImport string
}

// ExampleServerFiles returns an example gRPC server implementation for each
// server expression that hosts a gRPC service.
func ExampleServerFiles(genpkg string, services *ServicesData) []*codegen.File {
	var fw []*codegen.File
	servers := example.NewServersData()
	for _, svr := range services.Root.API.Servers {
		if !servers.Get(svr, services.Root).HostsGRPC() {
			continue
		}
		if m := exampleServer(genpkg, services, svr, servers); m != nil {
			fw = append(fw, m)
		}
	}
	return fw
}

// exampleServer returns an example gRPC server implementation.
func exampleServer(genpkg string, services *ServicesData, svr *expr.ServerExpr, servers example.ServersData) *codegen.File {
	var (
		mainPath string

		svrdata = servers.Get(svr, services.Root)
	)
	mainPath = filepath.Join("cmd", svrdata.Dir, "grpc.go")
	if _, err := os.Stat(mainPath); !os.IsNotExist(err) {
		return nil // file already exists, skip it.
	}

	specs, exampleServices := exampleServerImports(genpkg, services)

	var (
		sections []codegen.Section
	)
	var svcdata []exampleServerServiceData
	for _, svc := range svr.Services {
		if data, ok := exampleServices[svc]; ok {
			svcdata = append(svcdata, data)
		}
	}
	sections = []codegen.Section{
		codegen.Header("", "main", specs),
		grpcExampleServerSection(svcdata),
	}
	return &codegen.File{Path: mainPath, Sections: sections, SkipExist: true}
}

// exampleServerImports allocates and retains each import role before rendering
// any references, keeping the transport data independent of this file's scope.
func exampleServerImports(genpkg string, services *ServicesData) ([]*codegen.ImportSpec, map[string]exampleServerServiceData) {
	baseSpecs := []*codegen.ImportSpec{
		{Path: "context"},
		{Path: "fmt"},
		{Path: "net"},
		{Path: "net/url"},
		{Path: "sync"},
		codegen.LoomNamedImport("grpc", "loomgrpc"),
		{Path: "github.com/CaliLuke/loom/clue/debug"},
		{Path: "github.com/CaliLuke/loom/clue/log"},
		{Path: "google.golang.org/grpc"},
		{Path: "google.golang.org/grpc/reflection"},
	}
	specs := make([]*codegen.ImportSpec, 0, len(baseSpecs)+3*len(services.Root.API.GRPC.Services)+1)
	specs = append(specs, baseSpecs...)
	scope := exampleServerImportScope(specs)
	exampleServices := make(map[string]exampleServerServiceData, len(services.Root.API.GRPC.Services))
	for _, svc := range services.Root.API.GRPC.Services {
		sd := services.Get(svc.Name())
		svcName := sd.Service.PathName
		data := exampleServerServiceData{
			data:           sd,
			serverImport:   scope.Unique(sd.Service.PkgName + "svr"),
			serviceImport:  scope.Unique(sd.Service.PkgName),
			protobufImport: scope.Unique(sd.PkgName),
		}
		exampleServices[svc.Name()] = data
		specs = append(specs,
			&codegen.ImportSpec{
				Path: path.Join(genpkg, "grpc", svcName, "server"),
				Name: data.serverImport,
			},
			&codegen.ImportSpec{
				Path: path.Join(genpkg, svcName),
				Name: data.serviceImport,
			},
			&codegen.ImportSpec{
				Path: path.Join(genpkg, "grpc", svcName, pbPkgName),
				Name: data.protobufImport,
			})
	}

	var (
		rootPath string
		apiPkg   string
	)
	// genpkg is created by path.Join so the separator is / regardless of operating system
	rootPath = "."
	if parent, _, ok := strings.CutLast(genpkg, "/"); ok && parent != "" {
		rootPath = parent
	}
	apiPkg = scope.Unique(service.PackageBaseName(services.Root.API.Name), "api")
	specs = append(specs, &codegen.ImportSpec{Path: rootPath, Name: apiPkg})

	return specs, exampleServices
}

func exampleServerImportScope(specs []*codegen.ImportSpec) *codegen.NameScope {
	scope := codegen.NewNameScope()
	for _, spec := range specs {
		if spec == nil || spec.Name == "_" || spec.Name == "." {
			continue
		}
		name := naming.ImportName(spec.Path, spec.Name)
		if name != "" {
			scope.Unique(name)
		}
	}
	return scope
}
