package codegen

import (
	"path"
	"path/filepath"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/example"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

type jsonrpcExampleServiceData struct {
	Data                *httpcodegen.ServiceData
	ServiceImport       string
	HTTPServerImport    string
	JSONRPCServerImport string
}

// ExampleServerFiles returns an example JSON-RPC server implementation for
// each server expression that hosts a JSON-RPC service. The generated
// handleHTTPServer function also serves the plain HTTP services the server
// hosts, so files may contain the HTTP example server to extend.
func ExampleServerFiles(genpkg string, data *httpcodegen.ServicesData, files []*codegen.File) []*codegen.File {
	var fw []*codegen.File
	servers := example.NewServersData()
	httpData := httpcodegen.NewServicesData(data.ServicesData, data.Root.API.HTTP)
	for _, svr := range data.Root.API.Servers {
		if !servers.Get(svr, data.Root).HostsJSONRPC() {
			continue
		}
		if m := exampleServer(genpkg, data, httpData, svr, files, servers); m != nil {
			fw = append(fw, m)
		}
	}
	return fw
}

func exampleServer(genpkg string, data, httpData *httpcodegen.ServicesData, svr *expr.ServerExpr, files []*codegen.File, servers example.ServersData) *codegen.File {
	svrdata := servers.Get(svr, data.Root)
	httppath := filepath.Join("cmd", svrdata.Dir, "http.go")
	file := findOrBuildExampleHTTPServer(genpkg, httpData, svr, files, httppath)
	file.Path = filepath.Join(filepath.Dir(file.Path), "jsonrpc.go")

	sections := file.AllSections()
	header := file.HeaderSection()
	if header == nil {
		return file
	}
	addJSONRPCExampleImports(header, genpkg, data)

	svcdata := buildJSONRPCExampleServiceData(svr, data, header, genpkg)
	updatedSections := make([]codegen.Section, 0, len(sections)+2)
	httpServices := buildJSONRPCExampleServiceData(svr, httpData, header, genpkg)
	serviceImports := jsonrpcExampleServiceImports(httpServices, svcdata)
	apiPkg := jsonrpcExampleAPIPkg(genpkg, header, data)
	for _, section := range sections {
		switch section.SectionName() {
		case "server-http-start":
			updatedSections = append(updatedSections, codegen.NewRenderSection("server-http-start", func() string {
				return jsonrpcExampleServerStartSource(svrdata.HTTPHandlerArgs, serviceImports)
			}))
			continue
		case "server-http-end":
			updatedSections = append(updatedSections, codegen.NewRenderSection("server-http-end", func() string {
				return jsonrpcExampleServerEndSource(httpServices, svcdata)
			}))
			continue
		case "server-http-init":
			updatedSections = append(updatedSections, codegen.NewRenderSection("server-http-init", func() string {
				return jsonrpcExampleServerConfigureSource(httpServices, svcdata, apiPkg)
			}))
			continue
		}
		updatedSections = append(updatedSections, section)
	}
	file.SetSections(updatedSections)
	return file
}

// jsonrpcExampleServiceImports indexes the service package import names of
// the given services by service design name.
func jsonrpcExampleServiceImports(serviceGroups ...[]jsonrpcExampleServiceData) map[string]string {
	imports := make(map[string]string)
	for _, services := range serviceGroups {
		for _, svc := range services {
			imports[svc.Data.Service.Name] = svc.ServiceImport
		}
	}
	return imports
}

// findOrBuildExampleHTTPServer returns the example HTTP server of svr found
// at httpPath in files, or builds it from the HTTP services data httpData when
// the server hosts no HTTP service. Both paths yield the same file, which the
// caller extends with the JSON-RPC services.
func findOrBuildExampleHTTPServer(genpkg string, httpData *httpcodegen.ServicesData, svr *expr.ServerExpr, files []*codegen.File, httpPath string) *codegen.File {
	for _, f := range files {
		if f.Path == httpPath {
			return f
		}
	}
	return httpcodegen.ExampleServer(genpkg, httpData.Root, svr, httpData)
}

func addJSONRPCExampleImports(header codegen.Section, genpkg string, data *httpcodegen.ServicesData) {
	scope := codegen.NewNameScope()
	headerData := codegen.HeaderDataForSection(header)
	importsByPath := make(map[string]string)
	if headerData != nil {
		for _, spec := range headerData.Imports {
			if spec == nil || spec.Name == "_" || spec.Name == "." {
				continue
			}
			name := spec.Name
			if name == "" {
				name = path.Base(spec.Path)
			}
			if name != "" {
				scope.Unique(name)
				importsByPath[spec.Path] = name
			}
		}
	}
	for _, svc := range data.Root.API.JSONRPC.Services {
		sd := data.Get(svc.Name())
		svcName := sd.Service.PathName
		servicePath := path.Join(genpkg, svcName)
		serviceImport, ok := importsByPath[servicePath]
		if !ok {
			serviceImport = scope.Unique(sd.Service.PkgName, "svc")
			codegen.AddSectionImport(header, &codegen.ImportSpec{Path: servicePath, Name: serviceImport})
		}
		codegen.AddSectionImport(header, &codegen.ImportSpec{
			Path: path.Join(genpkg, "jsonrpc", svcName, "server"),
			Name: scope.Unique(serviceImport+"jssvr", "jssvr"),
		})
	}
}

func buildJSONRPCExampleServiceData(svr *expr.ServerExpr, data *httpcodegen.ServicesData, header codegen.Section, genpkg string) []jsonrpcExampleServiceData {
	svcdata := make([]jsonrpcExampleServiceData, 0, len(svr.Services))
	headerData := codegen.HeaderDataForSection(header)
	importNames := make(map[string]string)
	if headerData != nil {
		for _, spec := range headerData.Imports {
			if spec != nil && spec.Name != "" {
				importNames[spec.Path] = spec.Name
			}
		}
	}
	for _, svc := range svr.Services {
		d := data.Get(svc)
		if d == nil {
			continue
		}
		svcName := d.Service.PathName
		svcdata = append(svcdata, jsonrpcExampleServiceData{
			Data:                d,
			ServiceImport:       importNames[path.Join(genpkg, svcName)],
			HTTPServerImport:    importNames[path.Join(genpkg, "http", svcName, "server")],
			JSONRPCServerImport: importNames[path.Join(genpkg, "jsonrpc", svcName, "server")],
		})
	}
	return svcdata
}

func jsonrpcExampleAPIPkg(genpkg string, header codegen.Section, data *httpcodegen.ServicesData) string {
	headerData := codegen.HeaderDataForSection(header)
	if headerData != nil {
		rootPath := "."
		if parent, _, ok := strings.CutLast(genpkg, "/"); ok && parent != "" {
			rootPath = parent
		}
		for _, imp := range headerData.Imports {
			if imp.Path == rootPath {
				if imp.Name != "" {
					return imp.Name
				}
				parts := strings.Split(rootPath, "/")
				return parts[len(parts)-1]
			}
		}
	}
	return service.PackageBaseName(data.Root.API.Name)
}
