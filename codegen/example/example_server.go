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

// serverMainLocalNames lists the locals that the example server main
// declares in scope where it refers to the service packages. A service
// package named like one of them is imported under an alias, see
// exampleSvrMain. TestServerMainLocalNameReservationsCoverGeneratedLocals
// collects these names from the generated main and fails when the list
// misses one.
var serverMainLocalNames = []string{"ctx", "format"}

// ServerFiles returns an example server main implementation for every server
// expression in the service design.
func ServerFiles(genpkg string, root *expr.RootExpr, services *service.ServicesData) []*codegen.File {
	var fw []*codegen.File
	servers := NewServersData()
	for _, svr := range root.API.Servers {
		if m := exampleSvrMain(genpkg, root, svr, services, servers); m != nil {
			fw = append(fw, m)
		}
	}
	return fw
}

// exampleSvrMain returns the default main function for the given server
// expression.
func exampleSvrMain(genpkg string, root *expr.RootExpr, svr *expr.ServerExpr, services *service.ServicesData, servers ServersData) *codegen.File {
	svrdata := servers.Get(svr, root)
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
		{Path: "github.com/CaliLuke/loom/clue/log"},
	}

	scope := codegen.NewNameScope()
	for _, name := range serverMainLocalNames {
		scope.Unique(name)
	}
	svcData := serverMainServices(svr, services, scope)
	for _, sd := range svcData {
		specs = append(specs, &codegen.ImportSpec{
			Path: path.Join(genpkg, sd.PathName),
			Name: sd.PkgName,
		})
	}
	served := servedServices(svrdata, svcData)
	hasInterceptors := false
	for _, sd := range served {
		hasInterceptors = hasInterceptors || len(sd.ServerInterceptors) > 0
	}
	interPkg := scope.Unique("interceptors", "ex")

	var (
		rootPath string
		apiPkg   string
	)
	{
		// genpkg is created by path.Join so the separator is / regardless of operating system
		rootPath = "."
		if parent, _, ok := strings.CutLast(genpkg, "/"); ok && parent != "" {
			rootPath = parent
		}
		apiPkg = scope.Unique(service.PackageBaseName(root.API.Name), "api")
	}
	specs = append(specs, &codegen.ImportSpec{Path: rootPath, Name: apiPkg})
	if hasInterceptors {
		specs = append(specs, &codegen.ImportSpec{Path: path.Join(rootPath, "interceptors"), Name: interPkg})
	}

	sections := []codegen.Section{
		codegen.Header("", "main", specs),
		newRenderSection("server-main", func() string {
			return renderServerMain(svrdata, served, apiPkg, interPkg, hasInterceptors)
		}),
	}

	return &codegen.File{Path: mainPath, Sections: sections, SkipExist: true}
}

// serverMainServices returns the data of the services that svr hosts in the
// order it lists them. It allocates the import name of each service package
// in scope. The main qualifies the service types with PkgName, so the data
// of a service imported under an alias is a copy that holds the alias.
func serverMainServices(svr *expr.ServerExpr, services *service.ServicesData, scope *codegen.NameScope) []*service.Data {
	svcData := make([]*service.Data, len(svr.Services))
	for i, svc := range svr.Services {
		sd := services.Get(svc)
		if alias := scope.Unique(sd.PkgName, "svc"); alias != sd.PkgName {
			aliased := *sd
			aliased.PkgName = alias
			sd = &aliased
		}
		svcData[i] = sd
	}
	return svcData
}

// servedServices returns the services, in server order, whose endpoints the
// example main passes to the handler of a server URI. The main initializes
// only these services: the variables of a service that no transport of the
// server serves, such as a service without a transport, would be unused.
func servedServices(server *Data, services []*service.Data) []*service.Data {
	served := make(map[string]bool, len(services))
	for _, h := range server.Hosts {
		for _, u := range h.URIs {
			if !server.HasTransport(u.Transport.Type) {
				continue
			}
			for _, arg := range u.HandlerArgs {
				if arg.Endpoint != "" {
					served[arg.ServiceName] = true
				}
			}
		}
	}
	out := make([]*service.Data, 0, len(services))
	for _, svc := range services {
		if served[svc.Name] {
			out = append(out, svc)
		}
	}
	return out
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
