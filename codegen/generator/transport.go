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

		// Create service data
		services := service.NewServicesData(r)
		services.Ctx.Debug("transport codegen starting",
			"api", r.API.Name,
			"services", len(r.Services),
			"http_services", len(r.API.HTTP.Services),
			"grpc_services", len(r.API.GRPC.Services),
			"jsonrpc_services", len(r.API.JSONRPC.Services),
		)

		// HTTP
		httpServices := httpcodegen.NewServicesData(services, r.API.HTTP)
		files = append(files, httpcodegen.ServerFiles(genpkg, httpServices)...)
		files = append(files, httpcodegen.ClientFiles(genpkg, httpServices)...)
		files = append(files, httpcodegen.ServerTypeFiles(genpkg, httpServices)...)
		files = append(files, httpcodegen.ClientTypeFiles(genpkg, httpServices)...)
		files = append(files, httpcodegen.PathFiles(httpServices)...)
		files = append(files, httpcodegen.ClientCLIFiles(genpkg, httpServices)...)

		// GRPC
		grpcServices := grpccodegen.NewServicesData(services)
		files = append(files, grpccodegen.ProtoFiles(genpkg, grpcServices)...)
		files = append(files, grpccodegen.ServerFiles(genpkg, grpcServices)...)
		files = append(files, grpccodegen.ClientFiles(genpkg, grpcServices)...)
		files = append(files, grpccodegen.ServerTypeFiles(genpkg, grpcServices)...)
		files = append(files, grpccodegen.ClientTypeFiles(genpkg, grpcServices)...)
		files = append(files, grpccodegen.ClientCLIFiles(genpkg, grpcServices)...)

		// JSON-RPC
		jsonrpcServices := httpcodegen.NewServicesData(services, &r.API.JSONRPC.HTTPExpr)
		files = append(files, jsonrpccodegen.ServerFiles(genpkg, jsonrpcServices)...)
		files = append(files, jsonrpccodegen.ClientFiles(genpkg, jsonrpcServices)...)
		files = append(files, jsonrpccodegen.ServerTypeFiles(genpkg, jsonrpcServices)...)
		files = append(files, jsonrpccodegen.ClientTypeFiles(genpkg, jsonrpcServices)...)
		files = append(files, jsonrpccodegen.PathFiles(jsonrpcServices)...)
		files = append(files, jsonrpccodegen.ClientCLIFiles(genpkg, jsonrpcServices)...)
		files = append(files, jsonrpccodegen.SSEServerFiles(genpkg, jsonrpcServices)...)

		// Add service data meta type imports
		for _, f := range files {
			if header := f.HeaderTemplate(); header != nil {
				for _, s := range r.Services {
					d := services.Get(s.Name)
					service.AddServiceDataMetaTypeImports(header, s, d)
					service.AddUserTypeImports(genpkg, header, d)
				}
			}
		}
	}
	return files, nil
}
