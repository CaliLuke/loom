package codegen

import (
	"fmt"
	"path"
	"path/filepath"

	"goa.design/goa/v3/codegen"
	"goa.design/goa/v3/expr"
)

// ServerFiles returns all the server files for every gRPC service. The files
// contain the server which implements the generated gRPC server interface and
// encoders and decoders to transform protocol buffer types and gRPC metadata
// into goa types and vice versa.
func ServerFiles(genpkg string, services *ServicesData) []*codegen.File {
	svcLen := len(services.Root.API.GRPC.Services)
	fw := make([]*codegen.File, 2*svcLen)
	for i, svc := range services.Root.API.GRPC.Services {
		fw[i] = serverFile(genpkg, svc, services)
	}
	for i, svc := range services.Root.API.GRPC.Services {
		fw[i+svcLen] = serverEncodeDecode(genpkg, svc, services)
	}
	return fw
}

// serverFile returns the files defining the gRPC server.
func serverFile(genpkg string, svc *expr.GRPCServiceExpr, services *ServicesData) *codegen.File {
	var (
		fpath    string
		sections []codegen.Section

		data = services.Get(svc.Name())
	)
	{
		svcName := data.Service.PathName
		fpath = filepath.Join(codegen.Gendir, "grpc", svcName, "server", "server.go")
		imports := []*codegen.ImportSpec{
			{Path: "context"},
			{Path: "errors"},
			codegen.GoaImport(""),
			codegen.GoaNamedImport("grpc", "goagrpc"),
			{Path: "google.golang.org/grpc/codes"},
			{Path: path.Join(genpkg, svcName), Name: data.Service.PkgName},
			{Path: path.Join(genpkg, svcName, "views"), Name: data.Service.ViewsPkg},
			{Path: path.Join(genpkg, "grpc", svcName, pbPkgName), Name: data.PkgName},
		}
		sections = []codegen.Section{
			codegen.Header(svc.Name()+" gRPC server", "server", imports),
			grpcServerStructSection(data),
		}
		for _, e := range data.Endpoints {
			if e.ServerStream != nil {
				sections = append(sections, grpcStreamStructSection(e.ServerStream))
			}
		}
		sections = append(sections, grpcServerInitSection(data))
		for _, e := range data.Endpoints {
			sections = append(sections, grpcHandlerInitSection(e), grpcServerInterfaceSection(e))
		}
		for _, e := range data.Endpoints {
			if e.ServerStream != nil {
				if e.ServerStream.SendConvert != nil {
					sections = append(sections, grpcStreamSendSection(e.ServerStream))
				}
				if e.Method.StreamKind == expr.ClientStreamKind || e.Method.StreamKind == expr.BidirectionalStreamKind {
					sections = append(sections, grpcStreamRecvSection(e.ServerStream))
				}
				if e.ServerStream.MustClose {
					sections = append(sections, grpcStreamCloseSection(e.ServerStream))
				}
				if e.Method.ViewedResult != nil && e.Method.ViewedResult.ViewName == "" {
					sections = append(sections, grpcStreamSetViewSection(e.ServerStream))
				}
			}
		}
	}
	return &codegen.File{Path: fpath, Sections: sections}
}

// serverEncodeDecode returns the file defining the gRPC server encoding and
// decoding logic.
func serverEncodeDecode(genpkg string, svc *expr.GRPCServiceExpr, services *ServicesData) *codegen.File {
	var (
		fpath    string
		sections []codegen.Section

		data = services.Get(svc.Name())
	)
	{
		svcName := data.Service.PathName
		fpath = filepath.Join(codegen.Gendir, "grpc", svcName, "server", "encode_decode.go")
		title := fmt.Sprintf("%s gRPC server encoders and decoders", svc.Name())
		imports := []*codegen.ImportSpec{
			{Path: "context"},
			{Path: "strings"},
			{Path: "strconv"},
			{Path: "unicode/utf8"},
			{Path: "google.golang.org/grpc"},
			{Path: "google.golang.org/grpc/metadata"},
			codegen.GoaImport(""),
			codegen.GoaNamedImport("grpc", "goagrpc"),
			{Path: path.Join(genpkg, svcName), Name: data.Service.PkgName},
			{Path: path.Join(genpkg, svcName, "views"), Name: data.Service.ViewsPkg},
			{Path: path.Join(genpkg, "grpc", svcName, pbPkgName), Name: data.PkgName},
		}
		sections = []codegen.Section{codegen.Header(title, "server", imports)}

		for _, e := range data.Endpoints {
			if e.Response.ServerConvert != nil {
				sections = append(sections, grpcResponseEncoderSection(e))
			}
			if e.PayloadRef != "" {
				sections = append(sections, grpcRequestDecoderSection(e))
			}
		}
	}
	return &codegen.File{Path: fpath, Sections: sections}
}
