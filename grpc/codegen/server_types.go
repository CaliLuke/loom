package codegen

import (
	"path"
	"path/filepath"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

// ServerTypeFiles returns the server types files containing all the server
// interfaces and types needed to implement gRPC server.
func ServerTypeFiles(genpkg string, services *ServicesData) []*codegen.File {
	fw := make([]*codegen.File, len(services.Root.API.GRPC.Services))
	for i, svc := range services.Root.API.GRPC.Services {
		fw[i] = serverType(genpkg, svc, services)
	}
	return fw
}

// serverType returns the file defining the gRPC server types.
func serverType(genpkg string, svc *expr.GRPCServiceExpr, services *ServicesData) *codegen.File {
	var (
		initData []*InitData

		sd         = services.Get(svc.Name())
		foundInits = make(map[string]struct{})
	)
	{
		collect := func(c *ConvertData) {
			if c.Init != nil {
				initData = append(initData, c.Init)
			}
		}
		for _, a := range svc.GRPCEndpoints {
			ed := sd.Endpoint(a.Name())
			if c := ed.Request.ServerConvert; c != nil {
				collect(c)
			}
			if c := ed.Response.ServerConvert; c != nil {
				collect(c)
			}
			if ed.ServerStream != nil {
				if c := ed.ServerStream.SendConvert; c != nil {
					collect(c)
				}
				if c := ed.ServerStream.RecvConvert; c != nil {
					collect(c)
				}
			}
			for _, e := range ed.Errors {
				if c := e.Response.ServerConvert; c != nil {
					collect(c)
				}
			}
		}
	}

	var (
		fpath    string
		sections []codegen.Section
	)
	{
		svcName := sd.Service.PathName
		fpath = filepath.Join(codegen.Gendir, "grpc", svcName, "server", "types.go")
		imports := []*codegen.ImportSpec{
			{Path: "unicode/utf8"},
			codegen.GoaImport(""),
			{Path: path.Join(genpkg, svcName), Name: sd.Service.PkgName},
			{Path: path.Join(genpkg, svcName, "views"), Name: sd.Service.ViewsPkg},
			{Path: path.Join(genpkg, "grpc", svcName, pbPkgName), Name: sd.PkgName},
		}
		// Add imports if Any type is used
		needsAnyTypeImports := false
		for _, e := range svc.GRPCEndpoints {
			if hasAnyType(e.MethodExpr.Payload) || hasAnyType(e.MethodExpr.Result) {
				needsAnyTypeImports = true
				break
			}
			for _, er := range e.MethodExpr.Errors {
				if hasAnyType(er.AttributeExpr) {
					needsAnyTypeImports = true
					break
				}
			}
			if needsAnyTypeImports {
				break
			}
		}
		if needsAnyTypeImports {
			imports = append(imports, &codegen.ImportSpec{Path: "fmt"})
			imports = append(imports, &codegen.ImportSpec{Path: "google.golang.org/protobuf/types/known/structpb", Name: "structpb"})
		}
		imports = append(imports, sd.Service.ProtoImports...)
		sections = []codegen.Section{codegen.Header(svc.Name()+" gRPC server types", "server", imports)}
		for _, init := range initData {
			if _, ok := foundInits[init.Name]; ok {
				continue
			}
			sections = append(sections, grpcTypeInitSection(init))
			foundInits[init.Name] = struct{}{}
		}
		for _, data := range sd.validations {
			if data.Kind == validateClient {
				continue
			}
			sections = append(sections, grpcValidateSection(data))
		}
		for _, h := range sd.transformHelpers {
			sections = append(sections, grpcTransformHelperSection(h))
		}
	}
	return &codegen.File{Path: fpath, Sections: sections}
}
