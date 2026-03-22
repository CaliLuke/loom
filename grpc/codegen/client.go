package codegen

import (
	"path"
	"path/filepath"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
)

// ClientFiles returns the client files that contain client methods to call the
// corresponding service methods along with the encoding and decoding logic.
func ClientFiles(genpkg string, services *ServicesData) []*codegen.File {
	svcLen := len(services.Root.API.GRPC.Services)
	fw := make([]*codegen.File, 2*svcLen)
	for i, svc := range services.Root.API.GRPC.Services {
		fw[i] = clientFile(genpkg, svc, services)
	}
	for i, svc := range services.Root.API.GRPC.Services {
		fw[i+svcLen] = clientEncodeDecode(genpkg, svc, services)
	}
	return fw
}

// clientFile returns the file implementing the gRPC client.
func clientFile(genpkg string, svc *expr.GRPCServiceExpr, services *ServicesData) *codegen.File {
	var (
		fpath    string
		sections []codegen.Section

		data = services.Get(svc.Name())
	)
	{
		svcName := data.Service.PathName
		fpath = filepath.Join(codegen.Gendir, "grpc", svcName, "client", "client.go")
		imports := []*codegen.ImportSpec{
			{Path: "context"},
			{Path: "google.golang.org/grpc"},
			codegen.GoaImport(""),
			codegen.GoaNamedImport("grpc", "goagrpc"),
			codegen.GoaNamedImport("grpc/pb", "goapb"),
			{Path: path.Join(genpkg, svcName), Name: data.Service.PkgName},
			{Path: path.Join(genpkg, svcName, "views"), Name: data.Service.ViewsPkg},
			{Path: path.Join(genpkg, "grpc", svcName, pbPkgName), Name: data.PkgName},
		}
		sections = []codegen.Section{
			codegen.Header(svc.Name()+" gRPC client", "client", imports),
			grpcClientStructSection(data),
		}
		for _, e := range data.Endpoints {
			if e.ClientStream != nil {
				sections = append(sections, grpcStreamStructSection(e.ClientStream))
			}
		}
		sections = append(sections, grpcClientInitSection(data))
		for _, e := range data.Endpoints {
			sections = append(sections, grpcClientEndpointInitSection(e))
		}
		for _, e := range data.Endpoints {
			if e.ClientStream != nil {
				if e.ClientStream.RecvConvert != nil {
					sections = append(sections, grpcStreamRecvSection(e.ClientStream))
				}
				if e.Method.StreamKind == expr.ClientStreamKind || e.Method.StreamKind == expr.BidirectionalStreamKind {
					sections = append(sections, grpcStreamSendSection(e.ClientStream))
				}
				if e.ClientStream.MustClose {
					sections = append(sections, grpcStreamCloseSection(e.ClientStream))
				}
				if e.Method.ViewedResult != nil && e.Method.ViewedResult.ViewName == "" {
					sections = append(sections, grpcStreamSetViewSection(e.ClientStream))
				}
			}
		}
	}
	return &codegen.File{Path: fpath, Sections: sections}
}

// clientEncodeDecode returns the file containing the gRPC client encoding and
// decoding logic.
func clientEncodeDecode(genpkg string, svc *expr.GRPCServiceExpr, services *ServicesData) *codegen.File {
	var (
		fpath    string
		sections []codegen.Section

		data = services.Get(svc.Name())
	)
	{
		svcName := data.Service.PathName
		fpath = filepath.Join(codegen.Gendir, "grpc", svcName, "client", "encode_decode.go")
		imports := []*codegen.ImportSpec{
			{Path: "fmt"},
			{Path: "context"},
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
		sections = []codegen.Section{codegen.Header(svc.Name()+" gRPC client encoders and decoders", "client", imports)}
		for _, e := range data.Endpoints {
			sections = append(sections, grpcRemoteMethodBuilderSection(e))
			if e.PayloadRef != "" {
				sections = append(sections, grpcRequestEncoderSection(e))
			}
			if e.ResultRef != "" || e.ClientStream != nil {
				sections = append(sections, grpcResponseDecoderSection(e))
			}
		}
	}
	return &codegen.File{Path: fpath, Sections: sections}
}

// isBearer returns true if the security scheme uses a Bearer scheme.
func isBearer(schemes []*service.SchemeData) bool {
	for _, s := range schemes {
		if s.Name != "Authorization" {
			continue
		}
		if s.Type == "JWT" || s.Type == "OAuth2" {
			return true
		}
	}
	return false
}
