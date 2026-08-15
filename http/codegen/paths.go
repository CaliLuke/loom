package codegen

import (
	"fmt"
	"path/filepath"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

// PathFiles returns the service path files.
func PathFiles(data *ServicesData) []*codegen.File {
	fw := make([]*codegen.File, 2*len(data.Expressions.Services))
	for i := 0; i < len(data.Expressions.Services); i++ {
		fw[i*2] = serverPath(data.Expressions.Services[i], data)
		fw[i*2+1] = clientPath(data.Expressions.Services[i], data)
	}
	return fw
}

// ServerPathFiles returns the service path files used by HTTP servers.
func ServerPathFiles(data *ServicesData) []*codegen.File {
	files := make([]*codegen.File, len(data.Expressions.Services))
	for i, service := range data.Expressions.Services {
		files[i] = serverPath(service, data)
	}
	return files
}

// serverPath returns the server file containing the request path constructors
// for the given service.
func serverPath(svc *expr.HTTPServiceExpr, services *ServicesData) *codegen.File {
	sd := services.Get(svc.Name())
	path := filepath.Join(codegen.Gendir, "http", sd.Service.PathName, "server", "paths.go")
	return &codegen.File{Path: path, Sections: pathSections(svc, "server", services)}
}

// clientPath returns the client file containing the request path constructors
// for the given service.
func clientPath(svc *expr.HTTPServiceExpr, services *ServicesData) *codegen.File {
	sd := services.Get(svc.Name())
	path := filepath.Join(codegen.Gendir, "http", sd.Service.PathName, "client", "paths.go")
	return &codegen.File{Path: path, Sections: pathSections(svc, "client", services)}
}

// pathSections returns the sections of the file of the pkg package that
// contains the request path constructors for the given service.
func pathSections(svc *expr.HTTPServiceExpr, pkg string, services *ServicesData) []codegen.Section {
	title := fmt.Sprintf("HTTP request path constructors for the %s service.", svc.Name())
	sections := make([]codegen.Section, 0, 1+len(svc.HTTPEndpoints))
	sections = append(sections,
		codegen.Header(title, pkg, []*codegen.ImportSpec{
			{Path: "fmt"},
			{Path: "strconv"},
			{Path: "strings"},
		}),
	)
	sdata := services.Get(svc.Name())
	for _, e := range svc.HTTPEndpoints {
		sections = append(sections, pathSection(sdata.Endpoint(e.Name())))
	}

	return sections
}
