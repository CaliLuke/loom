package codegen

import (
	"fmt"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/grpc/codegen/internal/transportir"
)

// buildStreamData builds the StreamData for the server and client streams.
//
// svr param indicates that the stream data is built for the server.
func (d *ServicesData) buildStreamData(endpoint *transportir.Endpoint, sd *ServiceData, svr bool) *StreamData {
	svc := sd.Service
	ed := sd.Endpoint(endpoint.Name)
	md := ed.Method
	streamDesc := service.BuildStreamDescriptor(svc, md, endpoint.Request.StreamingPayload, endpoint.Response.Result)
	svcCtx := serviceTypeContext(svc.PkgName, svc.Scope)
	result, resCtx := resultContext(endpoint, sd)
	resVar := "result"
	if streamDesc.Result.UsesViewedResult {
		resVar = "vresult"
	}

	var data *StreamData
	if svr {
		data = d.buildServerStreamData(endpoint, sd, svcCtx, resCtx, result, resVar, streamDesc)
	} else {
		data = d.buildClientStreamData(endpoint, sd, svcCtx, resCtx, result, resVar, streamDesc)
	}
	describeStreamData(data, md.Name)
	return data
}

func (d *ServicesData) buildServerStreamData(endpoint *transportir.Endpoint, sd *ServiceData, svcCtx, resCtx *codegen.AttributeContext, result *expr.AttributeExpr, resVar string, streamDesc service.StreamDescriptor) *StreamData {
	ed := sd.Endpoint(endpoint.Name)
	md := ed.Method
	data := &StreamData{
		VarName:          md.ServerStream.VarName,
		Type:             "server",
		Interface:        fmt.Sprintf("%s.%s_%sServer", sd.PkgName, sd.Service.StructName, md.VarName), // nolint: namescope -- proto-generated interface name stitched from scoped identifiers
		ServiceInterface: fmt.Sprintf("%s.%s", sd.Service.PkgName, md.ServerStream.Interface),          // nolint: namescope -- sd.Service.PkgName is the exact import alias
		Endpoint:         ed,
		MustClose:        md.ServerStream.MustClose,
	}
	if streamDesc.HasResult {
		data.SendName = md.ServerStream.SendName
		data.SendWithContextName = md.ServerStream.SendWithContextName
		data.SendRef = ed.ResultRef
		data.SendConvert = &ConvertData{
			SrcName: resCtx.Scope.Name(result, resCtx.Pkg(result), resCtx.Pointer, resCtx.UseDefault),
			SrcRef:  resCtx.Scope.Ref(result, resCtx.Pkg(result)),
			TgtName: protoBufGoFullTypeName(endpoint.Response.ProtoMessage, sd.PkgName, sd.Scope),
			TgtRef:  protoBufGoFullTypeRef(endpoint.Response.ProtoMessage, sd.PkgName, sd.Scope),
			Init:    d.buildInitData(result, endpoint.Response.ProtoMessage, resVar, "v", resCtx, true, true, true, sd),
		}
	}
	if streamDesc.HasPayload {
		data.RecvName = md.ServerStream.RecvName
		data.RecvWithContextName = md.ServerStream.RecvWithContextName
		data.RecvRef = streamDesc.Payload.Ref
		data.RecvConvert = &ConvertData{
			SrcName:    protoBufGoFullTypeName(endpoint.Request.ProtoStreamingInput, sd.PkgName, sd.Scope),
			SrcRef:     protoBufGoFullTypeRef(endpoint.Request.ProtoStreamingInput, sd.PkgName, sd.Scope),
			TgtName:    svcCtx.Scope.Name(endpoint.Request.StreamingPayload, svcCtx.Pkg(endpoint.Request.StreamingPayload), svcCtx.Pointer, svcCtx.UseDefault),
			TgtRef:     data.RecvRef,
			Init:       d.buildInitData(endpoint.Request.ProtoStreamingInput, endpoint.Request.StreamingPayload, "v", "spayload", svcCtx, false, true, true, sd),
			Validation: addValidation(endpoint.Request.ProtoStreamingInput, "stream", sd, true),
		}
	}
	return data
}

func (d *ServicesData) buildClientStreamData(endpoint *transportir.Endpoint, sd *ServiceData, svcCtx, resCtx *codegen.AttributeContext, result *expr.AttributeExpr, resVar string, streamDesc service.StreamDescriptor) *StreamData {
	ed := sd.Endpoint(endpoint.Name)
	md := ed.Method
	data := &StreamData{
		VarName:          md.ClientStream.VarName,
		Type:             "client",
		Interface:        fmt.Sprintf("%s.%s_%sClient", sd.PkgName, sd.Service.StructName, md.VarName), // nolint: namescope -- proto-generated interface name stitched from scoped identifiers
		ServiceInterface: fmt.Sprintf("%s.%s", sd.Service.PkgName, md.ClientStream.Interface),          // nolint: namescope -- sd.Service.PkgName is the exact import alias
		Endpoint:         ed,
		MustClose:        md.ClientStream.MustClose,
	}
	if streamDesc.HasPayload {
		data.SendName = md.ClientStream.SendName
		data.SendWithContextName = md.ClientStream.SendWithContextName
		data.SendRef = streamDesc.Payload.Ref
		data.SendConvert = &ConvertData{
			SrcName: svcCtx.Scope.Name(endpoint.Request.StreamingPayload, svcCtx.Pkg(endpoint.Request.StreamingPayload), svcCtx.Pointer, svcCtx.UseDefault),
			SrcRef:  data.SendRef,
			TgtName: protoBufGoFullTypeName(endpoint.Request.ProtoStreamingInput, sd.PkgName, sd.Scope),
			TgtRef:  protoBufGoFullTypeRef(endpoint.Request.ProtoStreamingInput, sd.PkgName, sd.Scope),
			Init:    d.buildInitData(endpoint.Request.StreamingPayload, endpoint.Request.ProtoStreamingInput, "spayload", "v", svcCtx, true, false, true, sd),
		}
	}
	if streamDesc.HasResult {
		data.RecvName = md.ClientStream.RecvName
		data.RecvWithContextName = md.ClientStream.RecvWithContextName
		data.RecvRef = ed.ResultRef
		data.RecvConvert = &ConvertData{
			SrcName:    protoBufGoFullTypeName(endpoint.Response.ProtoMessage, sd.PkgName, sd.Scope),
			SrcRef:     protoBufGoFullTypeRef(endpoint.Response.ProtoMessage, sd.PkgName, sd.Scope),
			TgtName:    resCtx.Scope.Name(result, resCtx.Pkg(result), resCtx.Pointer, resCtx.UseDefault),
			TgtRef:     resCtx.Scope.Ref(result, resCtx.Pkg(result)),
			Init:       d.buildInitData(endpoint.Response.ProtoMessage, result, "v", resVar, resCtx, false, false, true, sd),
			Validation: addValidation(endpoint.Response.ProtoMessage, "stream", sd, false),
		}
	}
	return data
}

func describeStreamData(data *StreamData, methodName string) {
	if data.SendConvert != nil {
		data.SendDesc = fmt.Sprintf("%s streams instances of %q to the %q endpoint gRPC stream.", data.SendName, data.SendConvert.TgtName, methodName)
		data.SendWithContextDesc = fmt.Sprintf("%s streams instances of %q to the %q endpoint gRPC stream with context.", data.SendWithContextName, data.SendConvert.TgtName, methodName)
	}
	if data.RecvConvert != nil {
		data.RecvDesc = fmt.Sprintf("%s reads instances of %q from the %q endpoint gRPC stream.", data.RecvName, data.RecvConvert.SrcName, methodName)
		data.RecvWithContextDesc = fmt.Sprintf("%s reads instances of %q from the %q endpoint gRPC stream with context.", data.RecvWithContextName, data.RecvConvert.SrcName, methodName)
	}
}
