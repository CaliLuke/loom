package codegen

import (
	"path"
	"path/filepath"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

// ClientTypeFiles returns the client types files containing all the client
// interfaces and types needed to implement gRPC client.
func ClientTypeFiles(genpkg string, services *ServicesData) []*codegen.File {
	fw := make([]*codegen.File, len(services.Root.API.GRPC.Services))
	for i, svc := range services.Root.API.GRPC.Services {
		fw[i] = clientType(genpkg, svc, services)
	}
	return fw
}

// clientType returns the file defining the gRPC client types.
func clientType(genpkg string, svc *expr.GRPCServiceExpr, services *ServicesData) *codegen.File {
	sd := services.Get(svc.Name())
	initData := collectClientInitData(svc, sd)
	svcName := sd.Service.PathName
	fpath := filepath.Join(codegen.Gendir, "grpc", svcName, "client", "types.go")
	imports := grpcTypeImports(genpkg, svc, sd)
	sections := []codegen.Section{codegen.Header(svc.Name()+" gRPC client types", "client", imports)}
	for _, init := range initData {
		sections = append(sections, grpcTypeInitSection(init))
	}
	for _, data := range sd.validations {
		if data.Kind == validateServer {
			continue
		}
		sections = append(sections, grpcValidateSection(data))
	}
	for _, h := range sd.transformHelpers {
		sections = append(sections, grpcTransformHelperSection(h))
	}
	return &codegen.File{Path: fpath, Sections: sections}
}

func collectClientInitData(svc *expr.GRPCServiceExpr, sd *ServiceData) []*InitData {
	return collectInitData(svc, sd, func(ed *EndpointData, collect func(*ConvertData)) {
		collect(ed.Request.ClientConvert)
		collect(ed.Response.ClientConvert)
		if ed.ClientStream != nil {
			collect(ed.ClientStream.RecvConvert)
			collect(ed.ClientStream.SendConvert)
		}
		for _, e := range ed.Errors {
			collect(e.Response.ClientConvert)
		}
	})
}

func collectInitData(svc *expr.GRPCServiceExpr, sd *ServiceData, gather func(*EndpointData, func(*ConvertData))) []*InitData {
	var initData []*InitData
	seen := make(map[string]struct{})
	collect := func(c *ConvertData) {
		if c == nil || c.Init == nil {
			return
		}
		if _, ok := seen[c.Init.Name]; ok {
			return
		}
		seen[c.Init.Name] = struct{}{}
		initData = append(initData, c.Init)
	}
	for _, a := range svc.GRPCEndpoints {
		gather(sd.Endpoint(a.Name()), collect)
	}
	return initData
}

func grpcTypeImports(genpkg string, svc *expr.GRPCServiceExpr, sd *ServiceData) []*codegen.ImportSpec {
	svcName := sd.Service.PathName
	imports := []*codegen.ImportSpec{
		{Path: "unicode/utf8"},
		codegen.LoomImport(""),
		{Path: path.Join(genpkg, svcName), Name: sd.Service.PkgName},
		{Path: path.Join(genpkg, svcName, "views"), Name: sd.Service.ViewsPkg},
		{Path: path.Join(genpkg, "grpc", svcName, pbPkgName), Name: sd.PkgName},
	}
	if grpcServiceNeedsAnyTypeImports(svc) {
		imports = append(imports, &codegen.ImportSpec{Path: "fmt"})
		imports = append(imports, &codegen.ImportSpec{Path: "google.golang.org/protobuf/types/known/structpb", Name: "structpb"})
	}
	imports = append(imports, sd.Service.ProtoImports...)
	return imports
}

func grpcServiceNeedsAnyTypeImports(svc *expr.GRPCServiceExpr) bool {
	for _, e := range svc.GRPCEndpoints {
		if hasAnyType(e.MethodExpr.Payload) || hasAnyType(e.MethodExpr.Result) {
			return true
		}
		for _, er := range e.MethodExpr.Errors {
			if hasAnyType(er.AttributeExpr) {
				return true
			}
		}
	}
	return false
}
