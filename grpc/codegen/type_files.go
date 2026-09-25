package codegen

import (
	"path/filepath"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

func grpcTypeFile(
	genpkg string,
	svc *expr.GRPCServiceExpr,
	services *ServicesData,
	side string,
	collect func(*expr.GRPCServiceExpr, *ServiceData) []*InitData,
	skipKind validateKind,
) *codegen.File {
	sd := services.Get(svc.Name())
	sd, imports := services.fileData(svc.Name(), grpcTypeImports(genpkg, svc, sd))
	initData := collect(svc, sd)
	svcName := sd.Service.PathName
	fpath := filepath.Join(codegen.Gendir, "grpc", svcName, side, "types.go")
	sections := []codegen.Section{codegen.Header(svc.Name()+" gRPC "+side+" types", side, imports)}
	for _, init := range initData {
		sections = append(sections, grpcTypeInitSection(init, sd.Service.Scope))
	}
	for _, data := range sd.validations {
		if data.Kind == skipKind {
			continue
		}
		sections = append(sections, grpcValidateSection(data))
	}
	for _, h := range sd.transformHelpers {
		if h.Kind == skipKind {
			continue
		}
		sections = append(sections, grpcTransformHelperSection(h.TransformFunctionData))
	}
	return &codegen.File{Path: fpath, Sections: sections}
}
