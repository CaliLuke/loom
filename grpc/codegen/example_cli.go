package codegen

import (
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/example"
	"github.com/CaliLuke/loom/expr"
)

// ExampleCLIFiles returns an example gRPC client tool implementation.
func ExampleCLIFiles(genpkg string, services *ServicesData) []*codegen.File {
	var files []*codegen.File
	for _, svr := range services.Root.API.Servers {
		if f := exampleCLI(genpkg, services, svr); f != nil {
			files = append(files, f)
		}
	}
	return files
}

// exampleCLI returns an example client tool HTTP implementation for the given
// server expression.
func exampleCLI(genpkg string, services *ServicesData, svr *expr.ServerExpr) *codegen.File {
	var (
		mainPath string
		rootPath string

		svrdata = example.Servers.Get(svr, services.Root)
	)
	mainPath = filepath.Join("cmd", svrdata.Dir+"-cli", "grpc.go")
	if _, err := os.Stat(mainPath); !os.IsNotExist(err) {
		return nil // file already exists, skip it.
	}
	idx := strings.LastIndex(genpkg, string("/"))
	rootPath = "."
	if idx > 0 {
		rootPath = genpkg[:idx]
	}

	specs := []*codegen.ImportSpec{
		{Path: "context"},
		{Path: "encoding/json"},
		{Path: "flag"},
		{Path: "fmt"},
		{Path: "google.golang.org/grpc"},
		{Path: "google.golang.org/grpc/credentials/insecure"},
		{Path: "os"},
		{Path: "time"},
		codegen.GoaImport(""),
		codegen.GoaNamedImport("grpc", "goagrpc"),
		{Path: rootPath + "/interceptors"},
		{Path: path.Join(genpkg, "grpc", "cli", svrdata.Dir), Name: "cli"},
	}

	var svcData []*ServiceData
	for _, svc := range svr.Services {
		if data := services.Get(svc); data != nil {
			svcData = append(svcData, data)
		}
	}

	sections := []codegen.Section{
		codegen.Header("", "main", specs),
		grpcExampleCLISection(string(svrdata.DefaultTransport().Type), svcData, "interceptors"),
	}

	return &codegen.File{Path: mainPath, Sections: sections, SkipExist: true}
}
