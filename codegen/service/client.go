package service

import (
	"path/filepath"

	"goa.design/goa/v3/codegen"
	"goa.design/goa/v3/expr"
)

const (
	// clientStructName is the name of the generated client data structure.
	clientStructName = "Client"
)

// ClientFile returns the client file for the given service.
func ClientFile(_ string, service *expr.ServiceExpr, services *ServicesData) *codegen.File {
	svc := services.Get(service.Name)
	data := endpointData(svc)
	path := filepath.Join(codegen.Gendir, svc.PathName, "client.go")
	var sections []codegen.Section
	{
		imports := []*codegen.ImportSpec{
			{Path: "context"},
			{Path: "io"},
			codegen.GoaImport(""),
		}
		header := codegen.Header(service.Name+" client", svc.PkgName, imports)
		def := clientStructSection(data)
		init := clientInitSection(data)
		sections = make([]codegen.Section, 0, 3+len(data.Methods))
		sections = append(sections, header, def, init)
		for _, m := range data.Methods {
			sections = append(sections, methodSection(m))
		}
	}

	return &codegen.File{Path: path, Sections: sections}
}
