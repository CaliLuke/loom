package generator

import (
	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/example"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
	grpccodegen "github.com/CaliLuke/loom/grpc/codegen"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
	jsonrpccodegen "github.com/CaliLuke/loom/jsonrpc/codegen"
)

// Example iterates through the roots and returns files that implement an
// example service, server, and client.
func Example(genpkg string, roots []eval.Root) ([]*codegen.File, error) {
	var files []*codegen.File
	for _, root := range roots {
		r, ok := root.(*expr.RootExpr)
		if !ok {
			continue // could be a plugin root expression
		}
		services := service.NewServicesData(r)
		for _, s := range r.Services {
			service.SetUserTypeImports(genpkg, services.Get(s.Name))
		}
		var rootFiles []*codegen.File
		rootFiles = append(rootFiles, baseExampleFiles(genpkg, r, services)...)
		rootFiles = append(rootFiles, httpExampleFiles(genpkg, r, services)...)
		rootFiles = append(rootFiles, jsonrpcExampleFiles(genpkg, r, services, rootFiles)...)
		rootFiles = append(rootFiles, grpcExampleFiles(genpkg, r, services)...)
		addServicesMetaTypeImports(rootFiles, services, r.Services)
		files = append(files, rootFiles...)
	}
	return files, nil
}

func baseExampleFiles(genpkg string, root *expr.RootExpr, services *service.ServicesData) []*codegen.File {
	files := make([]*codegen.File, 0, 4)
	files = append(files, service.ExampleServiceFiles(genpkg, root, services)...)
	files = append(files, service.ExampleInterceptorsFiles(genpkg, root, services)...)
	files = append(files, example.ServerFiles(genpkg, root, services)...)
	files = append(files, example.CLIFiles(genpkg, root)...)
	return files
}

func httpExampleFiles(genpkg string, root *expr.RootExpr, services *service.ServicesData) []*codegen.File {
	if len(root.API.HTTP.Services) == 0 {
		return nil
	}
	httpServices := httpcodegen.NewServicesData(services, root.API.HTTP)
	files := httpcodegen.ExampleServerFiles(genpkg, httpServices)
	return append(files, httpcodegen.ExampleCLIFiles(genpkg, httpServices)...)
}

func jsonrpcExampleFiles(genpkg string, root *expr.RootExpr, services *service.ServicesData, existing []*codegen.File) []*codegen.File {
	if len(root.API.JSONRPC.Services) == 0 {
		return nil
	}
	jsonrpcServices := httpcodegen.NewServicesData(services, &root.API.JSONRPC.HTTPExpr)
	files := jsonrpccodegen.ExampleServerFiles(genpkg, jsonrpcServices, existing)
	return append(files, jsonrpccodegen.ExampleCLIFiles(genpkg, jsonrpcServices)...)
}

func grpcExampleFiles(genpkg string, root *expr.RootExpr, services *service.ServicesData) []*codegen.File {
	if len(root.API.GRPC.Services) == 0 {
		return nil
	}
	grpcServices := grpccodegen.NewServicesData(services)
	files := grpccodegen.ExampleServerFiles(genpkg, grpcServices)
	return append(files, grpccodegen.ExampleCLIFiles(genpkg, grpcServices)...)
}
