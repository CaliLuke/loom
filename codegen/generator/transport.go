package generator

import (
	"fmt"
	"path/filepath"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
	grpccodegen "github.com/CaliLuke/loom/grpc/codegen"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
	jsonrpccodegen "github.com/CaliLuke/loom/jsonrpc/codegen"
)

const (
	httpGenerationAll    = "all"
	httpGenerationServer = "server"
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
		httpFiles, err := httpTransportFiles(genpkg, r, services)
		if err != nil {
			return nil, err
		}
		rootFiles = append(rootFiles, httpFiles...)

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

func httpTransportFiles(genpkg string, root *expr.RootExpr, services *service.ServicesData) ([]*codegen.File, error) {
	mode, err := httpGenerationMode(root.API.Meta)
	if err != nil {
		return nil, err
	}
	httpServices := httpcodegen.NewServicesData(services, root.API.HTTP)
	serverFiles := httpcodegen.ServerFiles(genpkg, httpServices)
	files := append([]*codegen.File(nil), serverFiles...)
	if mode == httpGenerationAll {
		files = append(files, httpcodegen.ClientFiles(genpkg, httpServices)...)
	}
	files = append(files, httpcodegen.ServerTypeFiles(genpkg, httpServices)...)
	if mode == httpGenerationAll {
		files = append(files, httpcodegen.ClientTypeFiles(genpkg, httpServices)...)
		files = append(files, httpcodegen.PathFiles(httpServices)...)
		files = append(files, httpcodegen.ClientCLIFiles(genpkg, httpServices)...)
		return files, nil
	}
	files = append(files, httpcodegen.ServerPathFiles(httpServices)...)
	if len(serverFiles) > 0 {
		serverFiles[0].RemovePaths = append(serverFiles[0].RemovePaths, staleHTTPClientPaths(httpServices)...)
	}
	return files, nil
}

func httpGenerationMode(meta expr.MetaExpr) (string, error) {
	mode, ok := meta.Last("http:generate")
	if !ok {
		return httpGenerationAll, nil
	}
	switch mode {
	case httpGenerationAll, httpGenerationServer:
		return mode, nil
	default:
		return "", fmt.Errorf("HTTP generation mode %q must be one of all or server", mode)
	}
}

func staleHTTPClientPaths(services *httpcodegen.ServicesData) []string {
	paths := make([]string, 0, 1+len(services.Expressions.Services))
	paths = append(paths, filepath.Join(codegen.Gendir, "http", "cli"))
	for _, service := range services.Expressions.Services {
		serviceData := services.Get(service.Name())
		paths = append(paths, filepath.Join(codegen.Gendir, "http", serviceData.Service.PathName, "client"))
	}
	return paths
}
