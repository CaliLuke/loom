package codegen

import (
	"path"
	"path/filepath"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/example"
	"github.com/CaliLuke/loom/expr"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

// ExampleServerFiles returns example JSON-RPC server implementation.
func ExampleServerFiles(genpkg string, data *httpcodegen.ServicesData, files []*codegen.File) []*codegen.File {
	var fw []*codegen.File
	servers := example.NewServersData()
	for _, svr := range data.Root.API.Servers {
		if m := exampleServer(genpkg, data, svr, files, servers); m != nil {
			fw = append(fw, m)
		}
	}
	return fw
}

func exampleServer(genpkg string, data *httpcodegen.ServicesData, svr *expr.ServerExpr, files []*codegen.File, servers example.ServersData) *codegen.File {
	svrdata := servers.Get(svr, data.Root)
	httppath := filepath.Join("cmd", svrdata.Dir, "http.go")
	file, hasHTTP := findOrBuildExampleHTTPServer(genpkg, data, svr, files, httppath)
	file.Path = rewriteJSONRPCExampleServerPath(file.Path)

	sections := file.AllSections()
	header := file.HeaderSection()
	if header == nil {
		return file
	}
	addJSONRPCExampleImports(header, genpkg, data)

	svcdata := jsonrpcExampleServiceData(svr, data)
	updatedSections := make([]codegen.Section, 0, len(sections)+2)
	httpServices := []*httpcodegen.ServiceData(nil)
	if hasHTTP {
		httpServices = svcdata
	}
	apiPkg := jsonrpcExampleAPIPkg(genpkg, header, data)
	for _, section := range sections {
		switch section.SectionName() {
		case "server-http-start":
			updatedSections = append(updatedSections, codegen.NewRenderSection("server-http-start", func() string {
				return jsonrpcExampleServerStartSource(httpServices, svcdata)
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

func findOrBuildExampleHTTPServer(genpkg string, data *httpcodegen.ServicesData, svr *expr.ServerExpr, files []*codegen.File, httpPath string) (*codegen.File, bool) {
	for _, f := range files {
		if f.Path == httpPath {
			return f, true
		}
	}
	file := httpcodegen.ExampleServer(genpkg, data.Root, svr, data)
	updateHeader(file)
	return file, false
}

func addJSONRPCExampleImports(header codegen.Section, genpkg string, data *httpcodegen.ServicesData) {
	scope := codegen.NewNameScope()
	for _, svc := range data.Root.API.JSONRPC.Services {
		sd := data.Get(svc.Name())
		svcName := sd.Service.PathName
		codegen.AddSectionImport(header, &codegen.ImportSpec{
			Path: path.Join(genpkg, svcName),
			Name: scope.Unique(sd.Service.PkgName),
		})
		codegen.AddSectionImport(header, &codegen.ImportSpec{
			Path: path.Join(genpkg, "jsonrpc", svcName, "server"),
			Name: scope.Unique(sd.Service.PkgName + "jssvr"),
		})
	}
}

func jsonrpcExampleServiceData(svr *expr.ServerExpr, data *httpcodegen.ServicesData) []*httpcodegen.ServiceData {
	svcdata := make([]*httpcodegen.ServiceData, 0, len(svr.Services))
	for _, svc := range svr.Services {
		if d := data.Get(svc); d != nil {
			svcdata = append(svcdata, d)
		}
	}
	return svcdata
}

func jsonrpcExampleAPIPkg(genpkg string, header codegen.Section, data *httpcodegen.ServicesData) string {
	headerData := codegen.HeaderDataForSection(header)
	if headerData != nil {
		rootPath := "."
		if idx := strings.LastIndex(genpkg, "/"); idx > 0 {
			rootPath = genpkg[:idx]
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
	return strings.ToLower(codegen.Goify(data.Root.API.Name, false))
}
