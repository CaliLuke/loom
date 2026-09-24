package example

import (
	"os"
	"path/filepath"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

// CLIFiles returns example client tool main implementation for each server
// expression in the design.
func CLIFiles(genpkg string, root *expr.RootExpr) []*codegen.File {
	var fw []*codegen.File
	servers := NewServersData()
	for _, svr := range root.API.Servers {
		if m := exampleCLIMain(genpkg, root, svr, servers); m != nil {
			fw = append(fw, m)
		}
	}
	return fw
}

// exampleCLIMain returns an example client tool main implementation for the
// given server expression.
func exampleCLIMain(_ string, root *expr.RootExpr, svr *expr.ServerExpr, servers ServersData) *codegen.File {
	svrdata := servers.Get(svr, root)

	// Skip CLI generation for servers with no transports (e.g., agent-only services)
	if svrdata.DefaultTransport() == nil {
		return nil
	}

	path := filepath.Join("cmd", svrdata.Dir+"-cli", "main.go")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return nil // file already exists, skip it.
	}
	specs := []*codegen.ImportSpec{
		{Path: "context"},
		{Path: "encoding/json/jsontext"},
		{Path: "encoding/json/v2", Name: "json"},
		{Path: "errors"},
		{Path: "flag"},
		{Path: "fmt"},
		{Path: "net/url"},
		{Path: "os"},
		{Path: "sort"},
		{Path: "slices"},
		{Path: "strings"},
		codegen.LoomImport(""),
	}
	sections := []codegen.Section{
		codegen.Header("", "main", specs),
		newRenderSection("cli-main", func() string {
			return renderClientMain(svrdata, svrdata.HostsJSONRPC(), svrdata.HostsHTTP())
		}),
		newRenderSection("cli-main-usage", func() string {
			return renderUsage(root.API.Name, svrdata, svrdata.HostsJSONRPC(), svrdata.HostsHTTP())
		}),
	}
	return &codegen.File{Path: path, Sections: sections, SkipExist: true}
}
