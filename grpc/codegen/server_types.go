package codegen

import (
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
	sd := services.Get(svc.Name())
	initData := collectServerInitData(svc, sd)
	svcName := sd.Service.PathName
	fpath := filepath.Join(codegen.Gendir, "grpc", svcName, "server", "types.go")
	imports := grpcTypeImports(genpkg, svc, sd)
	sections := []codegen.Section{codegen.Header(svc.Name()+" gRPC server types", "server", imports)}
	for _, init := range initData {
		sections = append(sections, grpcTypeInitSection(init))
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
	return &codegen.File{Path: fpath, Sections: sections}
}

func collectServerInitData(svc *expr.GRPCServiceExpr, sd *ServiceData) []*InitData {
	return collectInitData(svc, sd, func(ed *EndpointData, collect func(*ConvertData)) {
		collect(ed.Request.ServerConvert)
		collect(ed.Response.ServerConvert)
		if ed.ServerStream != nil {
			collect(ed.ServerStream.SendConvert)
			collect(ed.ServerStream.RecvConvert)
		}
		for _, e := range ed.Errors {
			collect(e.Response.ServerConvert)
		}
	})
}
