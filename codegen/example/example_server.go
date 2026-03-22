package example

import (
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
)

// ServerFiles returns an example server main implementation for every server
// expression in the service design.
func ServerFiles(genpkg string, root *expr.RootExpr, services *service.ServicesData) []*codegen.File {
	var fw []*codegen.File
	for _, svr := range root.API.Servers {
		if m := exampleSvrMain(genpkg, root, svr, services); m != nil {
			fw = append(fw, m)
		}
	}
	return fw
}

// exampleSvrMain returns the default main function for the given server
// expression.
func exampleSvrMain(genpkg string, root *expr.RootExpr, svr *expr.ServerExpr, services *service.ServicesData) *codegen.File {
	svrdata := Servers.Get(svr, root)
	mainPath := filepath.Join("cmd", svrdata.Dir, "main.go")
	if _, err := os.Stat(mainPath); !os.IsNotExist(err) {
		return nil // file already exists, skip it.
	}
	specs := []*codegen.ImportSpec{
		{Path: "context"},
		{Path: "flag"},
		{Path: "fmt"},
		{Path: "net"},
		{Path: "net/url"},
		{Path: "os"},
		{Path: "os/signal"},
		{Path: "strings"},
		{Path: "sync"},
		{Path: "syscall"},
		{Path: "time"},
		{Path: "goa.design/clue/log"},
	}

	// Iterate through services listed in the server expression.
	svcData := make([]*service.Data, len(svr.Services))
	scope := codegen.NewNameScope()
	hasInterceptors := false
	for i, svc := range svr.Services {
		sd := services.Get(svc)
		svcData[i] = sd
		specs = append(specs, &codegen.ImportSpec{
			Path: path.Join(genpkg, sd.PathName),
			Name: scope.Unique(sd.PkgName, "svc"),
		})
		hasInterceptors = hasInterceptors || len(sd.ServerInterceptors) > 0
	}
	interPkg := scope.Unique("interceptors", "ex")

	var (
		rootPath string
		apiPkg   string
	)
	{
		// genpkg is created by path.Join so the separator is / regardless of operating system
		idx := strings.LastIndex(genpkg, string("/"))
		rootPath = "."
		if idx > 0 {
			rootPath = genpkg[:idx]
		}
		apiPkg = scope.Unique(strings.ToLower(codegen.Goify(root.API.Name, false)), "api")
	}
	specs = append(specs, &codegen.ImportSpec{Path: rootPath, Name: apiPkg})
	if hasInterceptors {
		specs = append(specs, &codegen.ImportSpec{Path: path.Join(rootPath, "interceptors"), Name: interPkg})
	}

	sections := []codegen.Section{
		codegen.Header("", "main", specs),
		newRenderSection("server-main", func() string {
			return renderServerMain(svrdata, svcData, apiPkg, interPkg, hasInterceptors, root)
		}),
	}

	return &codegen.File{Path: mainPath, Sections: sections, SkipExist: true}
}

// mustInitServices returns true if at least one of the services defines methods.
// It is used by the template to initialize service variables.
func mustInitServices(data []*service.Data) bool {
	for _, svc := range data {
		if len(svc.Methods) > 0 {
			return true
		}
	}
	return false
}

// hasJSONRPCEndpoints returns true if the service has JSON-RPC endpoints.
func hasJSONRPCEndpoints(root *expr.RootExpr, data *service.Data) bool {
	for _, svc := range root.API.JSONRPC.Services {
		if svc.Name() == data.Name {
			return true
		}
	}
	return false
}
