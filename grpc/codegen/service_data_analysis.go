package codegen

import (
	"fmt"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
)

// analyze creates the data necessary to render the code of the given service.
func (d *ServicesData) analyze(gs *expr.GRPCServiceExpr) *ServiceData {
	svc := d.ServicesData.Get(gs.Name())
	scope := codegen.NewNameScope()
	pkg := codegen.SnakeCase(codegen.Goify(svc.Name, false)) + pbPkgName
	svcVarN := scope.HashedUnique(gs.ServiceExpr, codegen.Goify(svc.Name, true))
	sd := &ServiceData{
		Service:             svc,
		Name:                svcVarN,
		Description:         svc.Description,
		PkgName:             pkg,
		ServerStruct:        "Server",
		ClientStruct:        "Client",
		ServerInit:          "New",
		ClientInit:          "NewClient",
		ServerInterface:     svcVarN + "Server",
		ClientInterface:     svcVarN + "Client",
		ClientInterfaceInit: fmt.Sprintf("%s.New%sClient", pkg, svcVarN),
		Scope:               scope,
	}
	collector := newMessageCollector(sd)
	for _, e := range gs.GRPCEndpoints {
		d.convertEndpointMessages(e, sd)
		md := svc.Method(e.Name())
		payloadRef, resultRef, viewedResultRef := endpointRefs(e, svc, md)
		errors := d.buildErrorsData(e, sd)
		collector.collectErrorMessages(e)
		request := d.buildRequestData(e, svc, sd, collector)
		response := d.buildResponseData(e, svc, sd, collector)
		msgSch, metSch := partitionSecuritySchemes(e, md)
		ed := &EndpointData{
			ServiceName:      svc.Name,
			PkgName:          sd.PkgName,
			ServicePkgName:   svc.PkgName,
			Method:           md,
			PayloadType:      e.MethodExpr.Payload.Type,
			PayloadRef:       payloadRef,
			ResultRef:        resultRef,
			ViewedResultRef:  viewedResultRef,
			Request:          request,
			Response:         response,
			MessageSchemes:   msgSch,
			MetadataSchemes:  metSch,
			Errors:           errors,
			ServerStruct:     sd.ServerStruct,
			ServerInterface:  sd.ServerInterface,
			ClientMethodName: protoBufify(md.VarName, true, true),
			ClientStruct:     sd.ClientStruct,
			ClientInterface:  sd.ClientInterface,
		}
		sd.Endpoints = append(sd.Endpoints, ed)
		if e.MethodExpr.IsStreaming() {
			ed.ServerStream = d.buildStreamData(e, sd, true)
			ed.ClientStream = d.buildStreamData(e, sd, false)
		}
	}
	return sd
}

type messageCollector struct {
	sd       *ServiceData
	seen     map[string]struct{}
	imported map[string]struct{}
}

func newMessageCollector(sd *ServiceData) *messageCollector {
	return &messageCollector{
		sd:       sd,
		seen:     make(map[string]struct{}),
		imported: make(map[string]struct{}),
	}
}

func (c *messageCollector) collect(att *expr.AttributeExpr) *service.UserTypeData {
	msgs, imports := collectMessages(att, c.sd, c.seen)
	c.appendImports(imports)
	if len(msgs) > 0 {
		c.sd.Messages = append(c.sd.Messages, msgs...)
		return msgs[0]
	}
	return c.lookupMessage(att)
}

func (c *messageCollector) appendImports(imports []string) {
	for _, imp := range imports {
		if _, ok := c.imported[imp]; ok {
			continue
		}
		c.imported[imp] = struct{}{}
		c.sd.ProtoImports = append(c.sd.ProtoImports, imp)
	}
}

func (c *messageCollector) lookupMessage(att *expr.AttributeExpr) *service.UserTypeData {
	ut, ok := att.Type.(expr.UserType)
	if !ok {
		return nil
	}
	name := ut.Name()
	if n := att.Meta["struct:name:proto"]; n != nil {
		name = n[0]
	}
	for _, t := range c.sd.Messages {
		if t.Name == name {
			return t
		}
	}
	return nil
}

func (c *messageCollector) collectErrorMessages(e *expr.GRPCEndpointExpr) {
	for _, er := range e.GRPCErrors {
		if er.Type == expr.ErrorResult || !expr.IsObject(er.Type) {
			continue
		}
		c.collect(er.Response.Message)
	}
}

func (d *ServicesData) convertEndpointMessages(e *expr.GRPCEndpointExpr, sd *ServiceData) {
	e.Request = makeProtoBufMessage(e.Request, protoBufify(e.Name()+"_request", true, true), sd)
	if e.MethodExpr.StreamingPayload.Type != expr.Empty {
		e.StreamingRequest = makeProtoBufMessage(e.StreamingRequest, protoBufify(e.Name()+"_streaming_request", true, true), sd)
	}
	e.Response.Message = makeProtoBufMessage(e.Response.Message, protoBufify(e.Name()+"_response", true, true), sd)
	for _, er := range e.GRPCErrors {
		if er.Type == expr.ErrorResult || !expr.IsObject(er.Type) {
			continue
		}
		er.Response.Message = makeProtoBufMessage(er.Response.Message, protoBufify(e.Name()+"_"+er.Name+"_error", true, true), sd)
	}
}

func endpointRefs(e *expr.GRPCEndpointExpr, svc *service.Data, md *service.MethodData) (string, string, string) {
	var payloadRef, resultRef, viewedResultRef string
	if e.MethodExpr.Payload.Type != expr.Empty {
		payloadRef = svc.Scope.GoFullTypeRef(e.MethodExpr.Payload,
			pkgWithDefault(md.PayloadLoc, svc.PkgName))
	}
	if e.MethodExpr.Result.Type != expr.Empty {
		resultRef = svc.Scope.GoFullTypeRef(e.MethodExpr.Result,
			pkgWithDefault(md.ResultLoc, svc.PkgName))
	}
	if md.ViewedResult != nil {
		viewedResultRef = md.ViewedResult.FullRef
	}
	return payloadRef, resultRef, viewedResultRef
}

func (d *ServicesData) buildRequestData(e *expr.GRPCEndpointExpr, svc *service.Data, sd *ServiceData, collector *messageCollector) *RequestData {
	reqMD := extractMetadata(e.Metadata, e.MethodExpr.Payload, svc.Scope, *d)
	request := &RequestData{
		Description:   e.Request.Description,
		Metadata:      reqMD,
		ServerConvert: d.buildRequestConvertData(e.Request, e.MethodExpr.Payload, reqMD, e, sd, true),
		ClientConvert: d.buildRequestConvertData(e.Request, e.MethodExpr.Payload, reqMD, e, sd, false),
	}
	if obj := expr.AsObject(e.Request.Type); (obj != nil && len(*obj) > 0) || expr.IsUnion(e.Request.Type) {
		request.CLIArgs = append(request.CLIArgs, &InitArgData{
			Name:     "message",
			Ref:      "message",
			TypeName: protoBufGoFullTypeName(e.Request, sd.PkgName, sd.Scope),
			TypeRef:  protoBufGoFullTypeRef(e.Request, sd.PkgName, sd.Scope),
			Example:  e.Request.Example(d.Root.API.ExampleGenerator),
		})
	}
	for _, m := range reqMD {
		request.CLIArgs = append(request.CLIArgs, &InitArgData{
			Name:         m.VarName,
			Ref:          m.VarName,
			FieldName:    m.FieldName,
			FieldType:    m.FieldType,
			TypeName:     m.TypeName,
			TypeRef:      m.TypeRef,
			Type:         m.Type,
			Pointer:      m.Pointer,
			Required:     m.Required,
			Validate:     m.Validate,
			Example:      m.Example,
			DefaultValue: m.DefaultValue,
		})
	}
	if e.StreamingRequest.Type != expr.Empty {
		request.Message = collector.collect(e.StreamingRequest)
	} else {
		request.Message = collector.collect(e.Request)
	}
	return request
}

func (d *ServicesData) buildResponseData(e *expr.GRPCEndpointExpr, svc *service.Data, sd *ServiceData, collector *messageCollector) *ResponseData {
	result, svcCtx := resultContext(e, sd)
	hdrs := extractMetadata(e.Response.Headers, result, svc.Scope, *d)
	trlrs := extractMetadata(e.Response.Trailers, result, svc.Scope, *d)
	response := &ResponseData{
		StatusCode:    statusCodeToGRPCConst(e.Response.StatusCode),
		Description:   e.Response.Description,
		Headers:       hdrs,
		Trailers:      trlrs,
		ServerConvert: d.buildResponseConvertData(e.Response.Message, result, svcCtx, hdrs, trlrs, e, sd, true),
		ClientConvert: d.buildResponseConvertData(e.Response.Message, result, svcCtx, hdrs, trlrs, e, sd, false),
	}
	if e.Response.Message.Type != expr.Empty || !e.MethodExpr.IsStreaming() {
		response.Message = collector.collect(e.Response.Message)
	}
	return response
}

func partitionSecuritySchemes(e *expr.GRPCEndpointExpr, md *service.MethodData) (service.SchemesData, service.SchemesData) {
	var msgSch service.SchemesData
	var metSch service.SchemesData
	for _, req := range e.Requirements {
		for _, sch := range req.Schemes {
			s := md.Requirements.Scheme(sch.SchemeName).Dup()
			s.In = sch.In
			switch s.In {
			case "message":
				msgSch = msgSch.Append(s)
			default:
				metSch = metSch.Append(s)
			}
		}
	}
	return msgSch, metSch
}
