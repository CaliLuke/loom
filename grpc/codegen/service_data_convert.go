package codegen

import (
	"fmt"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/grpc/codegen/internal/transportir"
)

// buildRequestConvertData builds the convert data for the server and client
// requests.
//   - server side - converts the one-shot gRPC request message (if any) and
//     gRPC metadata to the method payload type.
//   - client side - converts the method payload type to the one-shot gRPC
//     request message sent before any stream items.
//
// svr param indicates that the convert data is generated for server side.
func (d *ServicesData) buildRequestConvertData(endpoint *transportir.Endpoint, md []*MetadataData, sd *ServiceData, svr bool) *ConvertData {
	request := endpoint.Request.ProtoMessage
	payload := endpoint.Request.Payload
	if svr && isEmpty(payload.Type) {
		return nil
	}
	if !svr && endpoint.Stream.IsPayloadStreaming && isEmpty(endpoint.Request.Message.Type) {
		return nil
	}
	if svr && endpoint.Stream.IsPayloadStreaming && isEmpty(endpoint.Request.Message.Type) && !expr.IsObject(payload.Type) {
		return nil
	}

	svc := sd.Service
	pkg := service.DefaultPackageName(svc.Method(endpoint.Name).PayloadLoc, svc.PkgName)
	svcCtx := serviceTypeContext(pkg, svc.Scope)
	if svr {
		// server side
		data := d.buildInitData(request, payload, "message", "v", svcCtx, false, svr, false, sd)
		data.Name = fmt.Sprintf("New%sPayload", codegen.Goify(endpoint.Name, true))
		data.Description = fmt.Sprintf("%s builds the payload of the %q endpoint of the %q service from the gRPC request type.", data.Name, endpoint.Name, svc.Name)
		for _, m := range md {
			// pass the metadata as arguments to payload constructor in server
			data.Args = append(data.Args, &InitArgData{
				Name:         m.VarName,
				Ref:          m.VarName,
				FieldName:    m.FieldName,
				FieldPointer: payload.IsPrimitivePointer(m.AttributeName, true),
				FieldType:    m.FieldType,
				TypeName:     m.TypeName,
				TypeRef:      m.TypeRef,
				Type:         m.Type,
				Pointer:      m.Pointer,
				Required:     m.Required,
				Validate:     m.Validate,
				Example:      m.Example,
			})
		}
		return &ConvertData{
			SrcName:    protoBufGoFullTypeName(request, sd.PkgName, sd.Scope),
			SrcRef:     protoBufGoFullTypeRef(request, sd.PkgName, sd.Scope),
			TgtName:    svc.Scope.GoFullTypeName(payload, svcCtx.Pkg(payload)),
			TgtRef:     svc.Scope.GoFullTypeRef(payload, svcCtx.Pkg(payload)),
			Init:       data,
			Validation: addValidation(request, "message", sd, true),
		}
	}

	// client side
	data := d.buildInitData(payload, request, "payload", "message", svcCtx, true, svr, false, sd)
	data.Description = fmt.Sprintf("%s builds the gRPC request type from the payload of the %q endpoint of the %q service.", data.Name, endpoint.Name, svc.Name)
	return &ConvertData{
		SrcName: svc.Scope.GoFullTypeName(payload, pkg),
		SrcRef:  svc.Scope.GoFullTypeRef(payload, pkg),
		TgtName: protoBufGoFullTypeName(request, sd.PkgName, sd.Scope),
		TgtRef:  protoBufGoFullTypeRef(request, sd.PkgName, sd.Scope),
		Init:    data,
	}
}

// buildResponseConvertData builds the convert data for the server and client
// responses.
//   - server side - converts method result type to generated gRPC response
//     type in *.pb.go
//   - client side - converts generated gRPC response type in *.pb.go and
//     response metadata to method result type.
//
// svr param indicates that the convert data is generated for server side.
func (d *ServicesData) buildResponseConvertData(endpoint *transportir.Endpoint, result *expr.AttributeExpr, svcCtx *codegen.AttributeContext, hdrs, trlrs []*MetadataData, sd *ServiceData, svr bool) *ConvertData {
	response := endpoint.Response.ProtoMessage
	if !svr && (endpoint.Stream.IsStreaming || isEmpty(endpoint.Response.Result.Type)) {
		return nil
	}
	svc := sd.Service
	if svr {
		// server side
		data := d.buildInitData(result, response, "result", "message", svcCtx, true, svr, false, sd)
		data.Description = fmt.Sprintf("%s builds the gRPC response type from the result of the %q endpoint of the %q service.", data.Name, endpoint.Name, svc.Name)
		return &ConvertData{
			SrcName: svcCtx.Scope.Name(result, svcCtx.Pkg(result), svcCtx.Pointer, svcCtx.UseDefault),
			SrcRef:  svcCtx.Scope.Ref(result, svcCtx.Pkg(result)),
			TgtName: protoBufGoFullTypeName(response, sd.PkgName, sd.Scope),
			TgtRef:  protoBufGoFullTypeRef(response, sd.PkgName, sd.Scope),
			Init:    data,
		}
	}

	// client side
	data := d.buildInitData(response, result, "message", "result", svcCtx, false, svr, false, sd)
	data.Name = fmt.Sprintf("New%sResult", codegen.Goify(endpoint.Name, true))
	data.Description = fmt.Sprintf("%s builds the result type of the %q endpoint of the %q service from the gRPC response type.", data.Name, endpoint.Name, svc.Name)
	for _, m := range hdrs {
		// pass the headers as arguments to result constructor in client
		data.Args = append(data.Args, &InitArgData{
			Name:         m.VarName,
			Ref:          m.VarName,
			FieldName:    m.FieldName,
			FieldPointer: svcCtx.IsPrimitivePointer(m.AttributeName, result),
			FieldType:    m.FieldType,
			TypeName:     m.TypeName,
			TypeRef:      m.TypeRef,
			Type:         m.Type,
			Pointer:      m.Pointer,
			Required:     m.Required,
			Validate:     m.Validate,
			Example:      m.Example,
		})
	}
	for _, m := range trlrs {
		// pass the trailers as arguments to result constructor in client
		data.Args = append(data.Args, &InitArgData{
			Name:         m.VarName,
			Ref:          m.VarName,
			FieldName:    m.FieldName,
			FieldPointer: svcCtx.IsPrimitivePointer(m.AttributeName, result),
			FieldType:    m.FieldType,
			TypeName:     m.TypeName,
			TypeRef:      m.TypeRef,
			Type:         m.Type,
			Pointer:      m.Pointer,
			Required:     m.Required,
			Validate:     m.Validate,
			Example:      m.Example,
		})
	}
	return &ConvertData{
		SrcName:    protoBufGoFullTypeName(response, sd.PkgName, sd.Scope),
		SrcRef:     protoBufGoFullTypeRef(response, sd.PkgName, sd.Scope),
		TgtName:    svcCtx.Scope.Name(result, svcCtx.Pkg(result), svcCtx.Pointer, svcCtx.UseDefault),
		TgtRef:     svcCtx.Scope.Ref(result, svcCtx.Pkg(result)),
		Init:       data,
		Validation: addValidation(response, "message", sd, false),
	}
}

// buildInitData builds the transformation code to convert source to target.
//
// source, target are the source and target attributes used in the
// transformation
// sourceVar, targetVar are the source and target variable names used in the
// transformation
// svcCtx is the attribute context for service type
// proto if true indicates the target type is a protocol buffer type
// svr if true indicates the code is generated for conversion server side
func (d *ServicesData) buildInitData(source, target *expr.AttributeExpr, sourceVar, targetVar string, svcCtx *codegen.AttributeContext, proto, svr, usesrc bool, sd *ServiceData) *InitData {
	pbCtx := protoBufTypeContext(sd.PkgName, sd.Scope, false)
	name := "New"
	srcCtx := pbCtx
	tgtCtx := svcCtx
	if proto {
		srcCtx = svcCtx
		tgtCtx = pbCtx
		name += "Proto"
	}
	isStruct := expr.IsObject(target.Type) || expr.IsUnion(target.Type)
	if _, ok := source.Type.(expr.UserType); ok && usesrc {
		name += protoBufGoTypeName(source, sd.Scope)
	}
	n := protoBufGoTypeName(target, sd.Scope)
	if !isStruct {
		// If target is array, map, or primitive the name will be suffixed with
		// the definition (e.g int, []string, map[int]string) which is incorrect.
		n = protoBufGoTypeName(source, sd.Scope)
	}
	name += n
	code, helpers, err := protoBufTransform(source, target, sourceVar, targetVar, srcCtx, tgtCtx, proto, true)
	if err != nil {
		panic(codegen.NewError(d.ServicesData.Ctx, target, fmt.Errorf("build gRPC transform %s to %s: %w", source.Type.Name(), target.Type.Name(), err)))
	}
	sd.transformHelpers = appendTransformHelpers(sd.transformHelpers, helpers, svr)
	var args []*InitArgData
	if (!proto && !isEmpty(source.Type)) || (proto && !isEmpty(target.Type)) {
		args = []*InitArgData{{
			Name:     sourceVar,
			Ref:      sourceVar,
			TypeName: srcCtx.Scope.Name(source, srcCtx.Pkg(source), srcCtx.Pointer, srcCtx.UseDefault),
			TypeRef:  srcCtx.Scope.Ref(source, srcCtx.Pkg(source)),
			Example:  source.Example(d.Root.API.ExampleGenerator),
		}}
	}
	return &InitData{
		Name:           name,
		ReturnVarName:  targetVar,
		ReturnTypeRef:  tgtCtx.Scope.Ref(target, tgtCtx.Pkg(target)),
		ReturnIsStruct: isStruct,
		ReturnTypePkg:  tgtCtx.Pkg(target),
		Code:           code,
		Args:           args,
		ErrorAware:     containsAny(source) || containsAny(target),
	}
}

func appendTransformHelpers(oldH []*TransformHelperData, newH []*codegen.TransformFunctionData, svr bool) []*TransformHelperData {
	kind := validateClient
	if svr {
		kind = validateServer
	}
	for _, h := range newH {
		found := false
		for _, h2 := range oldH {
			if h.Name != h2.Name {
				continue
			}
			found = true
			if h2.Kind != kind {
				h2.Kind = validateBoth
			}
			break
		}
		if !found {
			oldH = append(oldH, &TransformHelperData{
				TransformFunctionData: h,
				Kind:                  kind,
			})
		}
	}
	return oldH
}

// buildErrorsData builds the error data for all the error responses in the
// endpoint expression. The response message for each error response are
// inferred from the method's error expression if not specified explicitly.
func (d *ServicesData) buildErrorsData(endpoint *transportir.Endpoint, sd *ServiceData) []*ErrorData {
	svc := sd.Service
	method := svc.Method(endpoint.Name)
	errors := make([]*ErrorData, 0, len(endpoint.Errors))
	for _, v := range endpoint.Errors {
		responseData := &ResponseData{
			StatusCode:    statusCodeToGRPCConst(v.Response.StatusCode),
			Description:   v.Response.Description,
			ServerConvert: d.buildErrorConvertData(v, endpoint, sd, true),
			ClientConvert: d.buildErrorConvertData(v, endpoint, sd, false),
		}
		errorDesc := service.BuildErrorDescriptor(svc, method, v.Name, v.Attribute)
		errors = append(errors, &ErrorData{
			Name:     v.Name,
			Ref:      errorDesc.Type.Ref,
			Response: responseData,
		})
	}
	return errors
}

func (d *ServicesData) buildErrorConvertData(grpcErr *transportir.Error, endpoint *transportir.Endpoint, sd *ServiceData, svr bool) *ConvertData {
	// No need to build transformation functions for default error or non-object
	// types.
	if grpcErr.Type == expr.ErrorResult || !expr.IsObject(grpcErr.Attribute.Type) {
		return nil
	}
	svc := sd.Service
	svcCtx := serviceTypeContext(svc.PkgName, svc.Scope)
	if svr {
		// server side
		data := d.buildInitData(grpcErr.Attribute, grpcErr.Response.ProtoMessage, "er", "message", svcCtx, true, svr, false, sd)
		data.Name = fmt.Sprintf("New%s%sError", codegen.Goify(endpoint.Name, true), codegen.Goify(grpcErr.Name, true))
		data.Description = fmt.Sprintf("%s builds the gRPC error response type from the error of the %q endpoint of the %q service.", data.Name, endpoint.Name, svc.Name)
		return &ConvertData{
			SrcName: svcCtx.Scope.Name(grpcErr.Attribute, svcCtx.Pkg(grpcErr.Attribute), svcCtx.Pointer, svcCtx.UseDefault),
			SrcRef:  svcCtx.Scope.Ref(grpcErr.Attribute, svcCtx.Pkg(grpcErr.Attribute)),
			TgtName: protoBufGoFullTypeName(grpcErr.Response.ProtoMessage, sd.PkgName, sd.Scope),
			TgtRef:  protoBufGoFullTypeRef(grpcErr.Response.ProtoMessage, sd.PkgName, sd.Scope),
			Init:    data,
		}
	}

	// client side
	data := d.buildInitData(grpcErr.Response.ProtoMessage, grpcErr.Attribute, "message", "er", svcCtx, false, svr, false, sd)
	data.Name = fmt.Sprintf("New%s%sError", codegen.Goify(endpoint.Name, true), codegen.Goify(grpcErr.Name, true))
	data.Description = fmt.Sprintf("%s builds the error type of the %q endpoint of the %q service from the gRPC error response type.", data.Name, endpoint.Name, svc.Name)
	return &ConvertData{
		SrcName:    protoBufGoFullTypeName(grpcErr.Response.ProtoMessage, sd.PkgName, sd.Scope),
		SrcRef:     protoBufGoFullTypeRef(grpcErr.Response.ProtoMessage, sd.PkgName, sd.Scope),
		TgtName:    svcCtx.Scope.Name(grpcErr.Attribute, svcCtx.Pkg(grpcErr.Attribute), svcCtx.Pointer, svcCtx.UseDefault),
		TgtRef:     svcCtx.Scope.Ref(grpcErr.Attribute, svcCtx.Pkg(grpcErr.Attribute)),
		Init:       data,
		Validation: addValidation(grpcErr.Response.ProtoMessage, "errmsg", sd, false),
	}
}
