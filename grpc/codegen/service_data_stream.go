package codegen

import (
	"fmt"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

// buildStreamData builds the StreamData for the server and client streams.
//
// svr param indicates that the stream data is built for the server.
func (d *ServicesData) buildStreamData(e *expr.GRPCEndpointExpr, sd *ServiceData, svr bool) *StreamData {
	svc := sd.Service
	ed := sd.Endpoint(e.Name())
	md := ed.Method
	svcCtx := serviceTypeContext(svc.PkgName, svc.Scope)
	result, resCtx := resultContext(e, sd)
	resVar := "result"
	if md.ViewedResult != nil {
		resVar = "vresult"
	}

	var data *StreamData
	if svr {
		data = d.buildServerStreamData(e, sd, svcCtx, resCtx, result, resVar)
	} else {
		data = d.buildClientStreamData(e, sd, svcCtx, resCtx, result, resVar)
	}
	describeStreamData(data, md.Name)
	return data
}

func (d *ServicesData) buildServerStreamData(e *expr.GRPCEndpointExpr, sd *ServiceData, svcCtx, resCtx *codegen.AttributeContext, result *expr.AttributeExpr, resVar string) *StreamData {
	ed := sd.Endpoint(e.Name())
	md := ed.Method
	data := &StreamData{
		VarName:          md.ServerStream.VarName,
		Type:             "server",
		Interface:        fmt.Sprintf("%s.%s_%sServer", sd.PkgName, sd.Service.StructName, md.VarName),
		ServiceInterface: fmt.Sprintf("%s.%s", sd.Service.PkgName, md.ServerStream.Interface),
		Endpoint:         ed,
		MustClose:        md.ServerStream.MustClose,
	}
	if e.MethodExpr.Result.Type != expr.Empty {
		data.SendName = md.ServerStream.SendName
		data.SendWithContextName = md.ServerStream.SendWithContextName
		data.SendRef = ed.ResultRef
		data.SendConvert = &ConvertData{
			SrcName: resCtx.Scope.Name(result, resCtx.Pkg(result), resCtx.Pointer, resCtx.UseDefault),
			SrcRef:  resCtx.Scope.Ref(result, resCtx.Pkg(result)),
			TgtName: protoBufGoFullTypeName(e.Response.Message, sd.PkgName, sd.Scope),
			TgtRef:  protoBufGoFullTypeRef(e.Response.Message, sd.PkgName, sd.Scope),
			Init:    d.buildInitData(result, e.Response.Message, resVar, "v", resCtx, true, true, true, sd),
		}
	}
	if e.MethodExpr.StreamingPayload.Type != expr.Empty {
		data.RecvName = md.ServerStream.RecvName
		data.RecvWithContextName = md.ServerStream.RecvWithContextName
		data.RecvRef = svcCtx.Scope.Ref(e.MethodExpr.StreamingPayload, svcCtx.Pkg(e.MethodExpr.StreamingPayload))
		data.RecvConvert = &ConvertData{
			SrcName:    protoBufGoFullTypeName(e.StreamingRequest, sd.PkgName, sd.Scope),
			SrcRef:     protoBufGoFullTypeRef(e.StreamingRequest, sd.PkgName, sd.Scope),
			TgtName:    svcCtx.Scope.Name(e.MethodExpr.StreamingPayload, svcCtx.Pkg(e.MethodExpr.StreamingPayload), svcCtx.Pointer, svcCtx.UseDefault),
			TgtRef:     data.RecvRef,
			Init:       d.buildInitData(e.StreamingRequest, e.MethodExpr.StreamingPayload, "v", "spayload", svcCtx, false, true, true, sd),
			Validation: addValidation(e.StreamingRequest, "stream", sd, true),
		}
	}
	return data
}

func (d *ServicesData) buildClientStreamData(e *expr.GRPCEndpointExpr, sd *ServiceData, svcCtx, resCtx *codegen.AttributeContext, result *expr.AttributeExpr, resVar string) *StreamData {
	ed := sd.Endpoint(e.Name())
	md := ed.Method
	data := &StreamData{
		VarName:          md.ClientStream.VarName,
		Type:             "client",
		Interface:        fmt.Sprintf("%s.%s_%sClient", sd.PkgName, sd.Service.StructName, md.VarName),
		ServiceInterface: fmt.Sprintf("%s.%s", sd.Service.PkgName, md.ClientStream.Interface),
		Endpoint:         ed,
		MustClose:        md.ClientStream.MustClose,
	}
	if e.MethodExpr.StreamingPayload.Type != expr.Empty {
		data.SendName = md.ClientStream.SendName
		data.SendWithContextName = md.ClientStream.SendWithContextName
		data.SendRef = svcCtx.Scope.Ref(e.MethodExpr.StreamingPayload, svcCtx.Pkg(e.MethodExpr.StreamingPayload))
		data.SendConvert = &ConvertData{
			SrcName: svcCtx.Scope.Name(e.MethodExpr.StreamingPayload, svcCtx.Pkg(e.MethodExpr.StreamingPayload), svcCtx.Pointer, svcCtx.UseDefault),
			SrcRef:  data.SendRef,
			TgtName: protoBufGoFullTypeName(e.StreamingRequest, sd.PkgName, sd.Scope),
			TgtRef:  protoBufGoFullTypeRef(e.StreamingRequest, sd.PkgName, sd.Scope),
			Init:    d.buildInitData(e.MethodExpr.StreamingPayload, e.StreamingRequest, "spayload", "v", svcCtx, true, false, true, sd),
		}
	}
	if e.MethodExpr.Result.Type != expr.Empty {
		data.RecvName = md.ClientStream.RecvName
		data.RecvWithContextName = md.ClientStream.RecvWithContextName
		data.RecvRef = ed.ResultRef
		data.RecvConvert = &ConvertData{
			SrcName:    protoBufGoFullTypeName(e.Response.Message, sd.PkgName, sd.Scope),
			SrcRef:     protoBufGoFullTypeRef(e.Response.Message, sd.PkgName, sd.Scope),
			TgtName:    resCtx.Scope.Name(result, resCtx.Pkg(result), resCtx.Pointer, resCtx.UseDefault),
			TgtRef:     resCtx.Scope.Ref(result, resCtx.Pkg(result)),
			Init:       d.buildInitData(e.Response.Message, result, "v", resVar, resCtx, false, false, true, sd),
			Validation: addValidation(e.Response.Message, "stream", sd, false),
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
