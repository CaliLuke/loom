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
	initData := collect(svc, sd)
	svcName := sd.Service.PathName
	fpath := filepath.Join(codegen.Gendir, "grpc", svcName, side, "types.go")
	imports := grpcTypeImports(genpkg, svc, sd)
	sections := []codegen.Section{codegen.Header(svc.Name()+" gRPC "+side+" types", side, imports)}
	for _, init := range initData {
		sections = append(sections, grpcTypeInitSection(init))
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
