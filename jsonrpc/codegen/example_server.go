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
	for _, svr := range data.Root.API.Servers {
		if m := exampleServer(genpkg, data, svr, files); m != nil {
			fw = append(fw, m)
		}
	}
	return fw
}

func exampleServer(genpkg string, data *httpcodegen.ServicesData, svr *expr.ServerExpr, files []*codegen.File) *codegen.File {
	svrdata := example.Servers.Get(svr, data.Root)
	httppath := filepath.Join("cmd", svrdata.Dir, "http.go")

	// Retrieve existing HTTP server file or create a new one
	var file *codegen.File
	var hasHTTP bool
	for _, f := range files {
		if f.Path == httppath {
			file = f
			hasHTTP = true
			break
		}
	}
	if file == nil {
		file = httpcodegen.ExampleServer(genpkg, data.Root, svr, data)
		updateHeader(file)
	}
	file.Path = rewriteJSONRPCExampleServerPath(file.Path)

	// Add JSON-RPC imports to the HTTP server file
	sections := file.AllSections()
	header, ok := sections[0].(*codegen.SectionTemplate)
	if !ok {
		return file
	}
	scope := codegen.NewNameScope()
	for _, svc := range data.Root.API.JSONRPC.Services {
		sd := data.Get(svc.Name())
		svcName := sd.Service.PathName
		codegen.AddImport(header, &codegen.ImportSpec{
			Path: path.Join(genpkg, svcName),
			Name: scope.Unique(sd.Service.PkgName),
		})
		codegen.AddImport(header, &codegen.ImportSpec{
			Path: path.Join(genpkg, "jsonrpc", svcName, "server"),
			Name: scope.Unique(sd.Service.PkgName + "jssvr"),
		})
	}

	// Add JSON-RPC to the HTTP server file
	var svcdata []*httpcodegen.ServiceData
	for _, svc := range svr.Services {
		if d := data.Get(svc); d != nil {
			svcdata = append(svcdata, d)
		}
	}
	updatedSections := make([]codegen.Section, 0, len(sections)+2)
	httpServices := []*httpcodegen.ServiceData(nil)
	if hasHTTP {
		httpServices = svcdata
	}
	apiPkg := jsonrpcExampleAPIPkg(genpkg, header, data)
	for _, section := range sections {
		switch section.SectionName() {
		case "server-http-start":
			updatedSections = append(updatedSections, codegen.MustRenderSection("server-http-start", func() string {
				return jsonrpcExampleServerStartSource(httpServices, svcdata)
			}))
			continue
		case "server-http-end":
			updatedSections = append(updatedSections, codegen.MustRenderSection("server-http-end", func() string {
				return jsonrpcExampleServerEndSource(httpServices, svcdata)
			}))
			continue
		case "server-http-init":
			updatedSections = append(updatedSections, codegen.MustRenderSection("server-http-init", func() string {
				return jsonrpcExampleServerConfigureSource(httpServices, svcdata, apiPkg)
			}))
			continue
		}
		updatedSections = append(updatedSections, section)
	}
	file.SetSections(updatedSections)
	return file
}

func jsonrpcExampleAPIPkg(genpkg string, header *codegen.SectionTemplate, data *httpcodegen.ServicesData) string {
	headerData := codegen.HeaderSectionData(header)
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
