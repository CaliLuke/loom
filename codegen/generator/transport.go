package generator

import (
	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
	grpccodegen "github.com/CaliLuke/loom/grpc/codegen"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
	jsonrpccodegen "github.com/CaliLuke/loom/jsonrpc/codegen"
)

// Transport iterates through the roots and returns the files needed to render
// the transport code.
func Transport(genpkg string, roots []eval.Root) ([]*codegen.File, error) {
	var files []*codegen.File
	for _, root := range roots {
		r, ok := root.(*expr.RootExpr)
		if !ok {
			continue // could be a plugin root expression
		}
		rootFiles := make([]*codegen.File, 0, 32)

		// Create service data
		services := service.NewServicesData(r)
		for _, s := range r.Services {
			service.SetUserTypeImports(genpkg, services.Get(s.Name))
		}
		services.Ctx.Debug("transport codegen starting",
			"api", r.API.Name,
			"services", len(r.Services),
			"http_services", len(r.API.HTTP.Services),
			"grpc_services", len(r.API.GRPC.Services),
			"jsonrpc_services", len(r.API.JSONRPC.Services),
		)

		// HTTP
		httpServices := httpcodegen.NewServicesData(services, r.API.HTTP)
		rootFiles = append(rootFiles, httpcodegen.ServerFiles(genpkg, httpServices)...)
		rootFiles = append(rootFiles, httpcodegen.ClientFiles(genpkg, httpServices)...)
		rootFiles = append(rootFiles, httpcodegen.ServerTypeFiles(genpkg, httpServices)...)
		rootFiles = append(rootFiles, httpcodegen.ClientTypeFiles(genpkg, httpServices)...)
		rootFiles = append(rootFiles, httpcodegen.PathFiles(httpServices)...)
		rootFiles = append(rootFiles, httpcodegen.ClientCLIFiles(genpkg, httpServices)...)

		// GRPC
		grpcServices := grpccodegen.NewServicesData(services)
		rootFiles = append(rootFiles, grpccodegen.ProtoFiles(genpkg, grpcServices)...)
		rootFiles = append(rootFiles, grpccodegen.ServerFiles(genpkg, grpcServices)...)
		rootFiles = append(rootFiles, grpccodegen.ClientFiles(genpkg, grpcServices)...)
		rootFiles = append(rootFiles, grpccodegen.ServerTypeFiles(genpkg, grpcServices)...)
		rootFiles = append(rootFiles, grpccodegen.ClientTypeFiles(genpkg, grpcServices)...)
		rootFiles = append(rootFiles, grpccodegen.ClientCLIFiles(genpkg, grpcServices)...)

		// JSON-RPC
		jsonrpcServices := httpcodegen.NewServicesData(services, &r.API.JSONRPC.HTTPExpr)
		rootFiles = append(rootFiles, jsonrpccodegen.ServerFiles(genpkg, jsonrpcServices)...)
		rootFiles = append(rootFiles, jsonrpccodegen.ClientFiles(genpkg, jsonrpcServices)...)
		rootFiles = append(rootFiles, jsonrpccodegen.ServerTypeFiles(genpkg, jsonrpcServices)...)
		rootFiles = append(rootFiles, jsonrpccodegen.ClientTypeFiles(genpkg, jsonrpcServices)...)
		rootFiles = append(rootFiles, jsonrpccodegen.PathFiles(jsonrpcServices)...)
		rootFiles = append(rootFiles, jsonrpccodegen.ClientCLIFiles(genpkg, jsonrpcServices)...)
		rootFiles = append(rootFiles, jsonrpccodegen.SSEServerFiles(genpkg, jsonrpcServices)...)

		// Add service data meta type imports
		addServicesImports(rootFiles, services, r.Services)
		files = append(files, rootFiles...)
	}
	return files, nil
}
