package codegen

import (
	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

// finalizeServerMembers assigns roles in the concrete server receiver. Protobuf
// RPC identities are already final and never change to accommodate fields.
func finalizeServerMembers(data *ServiceData, service *expr.GRPCServiceExpr) {
	receiver := codegen.NewNameScope()
	for _, endpoint := range data.Endpoints {
		receiver.Unique(endpoint.RPCGoName)
	}
	embedding := "Unimplemented" + data.ServerInterface
	data.serverEmbedding = embedding
	if receiver.PeekUnique(embedding) != embedding {
		alias := codegen.NewNameScope()
		for _, endpoint := range data.Endpoints {
			alias.Unique(endpoint.RPCGoName)
		}
		for _, name := range serverPackageNames(data, service) {
			alias.Unique(name)
		}
		data.serverEmbeddingAlias = alias.Unique(codegen.Goify(embedding, false), "Embedded")
		data.serverEmbedding = data.serverEmbeddingAlias
	}
	receiver.Unique(data.serverEmbedding)
	for _, endpoint := range data.Endpoints {
		endpoint.ServerHandler = receiver.Unique(endpoint.Method.VarName+"H", "Handler")
		if admission := endpoint.Request.Admission; admission != nil {
			admission.CheckField = receiver.Unique("authenticate"+endpoint.Method.VarName, "Check")
		}
	}
}

// serverPackageNames lists declarations and imported type bindings that share
// the conditional embedding alias's package. Receiver-only roles stay separate.
func serverPackageNames(data *ServiceData, service *expr.GRPCServiceExpr) []string {
	names := []string{data.ServerStruct, data.ServerInit, data.PkgName, data.Service.PkgName,
		data.Service.ViewsPkg, "context", "errors", "io", "loom", "loomgrpc", "security", "codes",
		"fmt", "strings", "strconv", "utf8", "grpc", "metadata"}
	if grpcServiceNeedsAnyTypeImports(service) {
		names = append(names, "structpb")
	}
	for _, spec := range data.ProtoGoImports {
		if spec.Name != "" {
			names = append(names, spec.Name)
		}
	}
	contracts := false
	for _, endpoint := range data.Endpoints {
		method := endpoint.Method.VarName
		names = append(names, "New"+method+"Handler")
		if endpoint.PayloadRef != "" {
			names = append(names, "Decode"+method+"Request")
		}
		if endpoint.Response.ServerConvert != nil {
			names = append(names, "Encode"+method+"Response")
		}
		if len(endpoint.Errors) > 0 {
			names = append(names, "map"+method+"Error")
		}
		if endpoint.ServerStream != nil {
			names = append(names, endpoint.ServerStream.VarName)
		}
		if admission := endpoint.Request.Admission; admission != nil {
			names = append(names, admission.DecoderName, admission.MetadataName)
		}
		if len(endpoint.ResponseContractCases) > 0 {
			contracts = true
			names = append(names, endpoint.ResponseContractCasesInit)
		}
	}
	if contracts {
		names = append(names, "ResponseContractCases")
	}
	for _, init := range collectServerInitData(service, data) {
		names = append(names, init.Name)
	}
	for _, validation := range data.validations {
		if validation.Kind != validateClient {
			names = append(names, validation.Name)
		}
	}
	for _, helper := range data.transformHelpers {
		if helper.Kind != validateClient {
			names = append(names, helper.Name)
		}
	}
	return append(names, serverLocatedImportNames(data, service)...)
}

// serverLocatedImportNames resolves authored type imports through the current
// file scope, including aliases chosen by FileData reanalysis.
func serverLocatedImportNames(data *ServiceData, service *expr.GRPCServiceExpr) []string {
	var names []string
	for _, method := range service.ServiceExpr.Methods {
		attributes := []*expr.AttributeExpr{method.Payload, method.Result, method.StreamingPayload, method.StreamingResult}
		for _, err := range method.Errors {
			attributes = append(attributes, err.AttributeExpr)
		}
		for _, attribute := range attributes {
			if attribute == nil {
				continue
			}
			if err := codegen.Walk(attribute, func(a *expr.AttributeExpr) error {
				if location := codegen.UserTypeLocation(a.Type); location != nil {
					names = append(names, data.Service.Scope.PackageName(location))
				}
				return nil
			}); err != nil {
				panic(err)
			}
		}
	}
	return names
}
