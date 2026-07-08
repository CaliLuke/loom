package codegen

import (
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
	return grpcTypeFile(genpkg, svc, services, "server", collectServerInitData, validateClient)
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
