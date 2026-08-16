package generator

import (
	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
	grpccodegen "github.com/CaliLuke/loom/grpc/codegen"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

// TestScaffold returns consumer-owned test scaffolds for the evaluated design
// roots. Existing scaffold files are never overwritten.
func TestScaffold(genpkg string, roots []eval.Root) ([]*codegen.File, error) {
	var files []*codegen.File
	for _, root := range roots {
		r, ok := root.(*expr.RootExpr)
		if !ok {
			continue
		}
		services := service.NewServicesData(r)
		httpServices := httpcodegen.NewServicesData(services, r.API.HTTP)
		files = append(files, httpcodegen.ResponseContractTestFiles(genpkg, httpServices)...)
		grpcServices := grpccodegen.NewServicesData(services)
		files = append(files, grpccodegen.ResponseContractTestFiles(genpkg, grpcServices)...)
	}
	return files, nil
}
