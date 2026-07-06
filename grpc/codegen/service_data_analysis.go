package codegen

import (
	"fmt"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/grpc/codegen/internal/transportir"
)

// analyze creates the data necessary to render the code of the given service.
// Panics are wrapped with DSL attribution (service, method, source location)
// and re-panicked so opaque failures surface with navigable context.
func (d *ServicesData) analyze(gs *expr.GRPCServiceExpr) (sd *ServiceData) {
	svc := d.ServicesData.Get(gs.Name())
	ctx := d.ServicesData.Ctx.WithService(gs.ServiceExpr)
	defer func() {
		if err := codegen.RecoverPanic(recover()); err != nil {
			panic(codegen.NewError(ctx, nil, err))
		}
	}()
	irService := transportir.BuildService(gs)
	scope := codegen.NewNameScope()
	pkg := codegen.SnakeCase(codegen.Goify(svc.Name, false)) + pbPkgName
	svcVarN := scope.HashedUnique(gs.ServiceExpr, codegen.Goify(svc.Name, true))
	sd = &ServiceData{
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
	for _, endpointIR := range irService.Endpoints {
		epCtx := ctx.WithMethod(gs.ServiceExpr.Method(endpointIR.Name))
		d.buildEndpointDataWithContext(epCtx, endpointIR, svc, sd, collector)
	}
	return sd
}

func (d *ServicesData) buildEndpointDataWithContext(
	ctx *codegen.Context,
	endpointIR *transportir.Endpoint,
	svc *service.Data,
	sd *ServiceData,
	collector *messageCollector,
) {
	defer func() {
		if err := codegen.RecoverPanic(recover()); err != nil {
			panic(codegen.NewError(ctx, nil, err))
		}
	}()
	prepareEndpointProtoMessages(endpointIR, sd)
	md := svc.Method(endpointIR.Name)
	payloadDesc := service.BuildPayloadDescriptor(svc, md, endpointIR.Request.Payload)
	resultDesc := service.BuildResultDescriptor(svc, md, endpointIR.Response.Result)
	errors := d.buildErrorsData(endpointIR, sd)
	collector.collectErrorMessages(endpointIR)
	request := d.buildRequestData(endpointIR, svc, sd, collector)
	response := d.buildResponseData(endpointIR, svc, sd, collector)
	msgSch, metSch := partitionSecuritySchemes(endpointIR, md)
	ed := &EndpointData{
		ServiceName:      svc.Name,
		PkgName:          sd.PkgName,
		ServicePkgName:   svc.PkgName,
		Method:           md,
		PayloadType:      endpointIR.Request.Payload.Type,
		PayloadRef:       payloadDesc.Ref,
		ResultRef:        resultDesc.Declared.Ref,
		ViewedResultRef:  resultDesc.ViewedRef,
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
	if endpointIR.Stream.IsStreaming {
		ed.ServerStream = d.buildStreamData(endpointIR, sd, true)
		ed.ClientStream = d.buildStreamData(endpointIR, sd, false)
	}
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

func (c *messageCollector) collectErrorMessages(endpoint *transportir.Endpoint) {
	for _, er := range endpoint.Errors {
		if er.Type == expr.ErrorResult || !expr.IsObject(er.Attribute.Type) {
			continue
		}
		c.collect(er.Response.ProtoMessage)
	}
}

func prepareEndpointProtoMessages(endpoint *transportir.Endpoint, sd *ServiceData) {
	useEnvelope := usesStreamEnvelope(endpoint)
	endpoint.Request.ProtoMessage = makeProtoBufMessage(endpoint.Request.Message, protoBufify(endpoint.Name+"_request", true, true), sd)
	if endpoint.Request.StreamingPayload.Type != expr.Empty {
		streamName := protoBufify(endpoint.Name+"_streaming_request", true, true)
		if useEnvelope {
			streamName = protoBufify(endpoint.Name+"_stream_item", true, true)
		}
		endpoint.Request.ProtoStreamingInput = makeProtoBufMessage(endpoint.Request.StreamingMessage, streamName, sd)
	}
	if useEnvelope {
		endpoint.Request.ProtoStreamEnvelope = makeProtoBufStreamEnvelope(
			endpoint.Request.ProtoMessage,
			endpoint.Request.ProtoStreamingInput,
			protoBufify(endpoint.Name+"_streaming_request", true, true),
			sd,
		)
	}
	endpoint.Response.ProtoMessage = makeProtoBufMessage(endpoint.Response.Message, protoBufify(endpoint.Name+"_response", true, true), sd)
	for _, grpcErr := range endpoint.Errors {
		if grpcErr.Type == expr.ErrorResult || !expr.IsObject(grpcErr.Attribute.Type) {
			continue
		}
		grpcErr.Response.ProtoMessage = makeProtoBufMessage(grpcErr.Response.Message, protoBufify(endpoint.Name+"_"+grpcErr.Name+"_error", true, true), sd)
	}
}

func (d *ServicesData) buildRequestData(endpoint *transportir.Endpoint, svc *service.Data, sd *ServiceData, collector *messageCollector) *RequestData {
	reqMD := extractMetadata(endpoint.Request.Metadata, endpoint.Request.Payload, svc.Scope, *d)
	request := &RequestData{
		Description:   endpoint.Request.ProtoMessage.Description,
		Metadata:      reqMD,
		ServerConvert: d.buildRequestConvertData(endpoint, reqMD, sd, true),
		ClientConvert: d.buildRequestConvertData(endpoint, reqMD, sd, false),
	}
	hasRequestMessage := !isEmpty(endpoint.Request.Message.Type)
	if obj := expr.AsObject(endpoint.Request.ProtoMessage.Type); (obj != nil && len(*obj) > 0) || expr.IsUnion(endpoint.Request.ProtoMessage.Type) {
		request.CLIArgs = append(request.CLIArgs, &InitArgData{
			Name:     "message",
			Ref:      "message",
			TypeName: protoBufGoFullTypeName(endpoint.Request.ProtoMessage, sd.PkgName, sd.Scope),
			TypeRef:  protoBufGoFullTypeRef(endpoint.Request.ProtoMessage, sd.PkgName, sd.Scope),
			Example:  endpoint.Request.ProtoMessage.Example(d.Root.API.ExampleGenerator),
		})
	}
	for _, m := range reqMD {
		request.CLIArgs = append(request.CLIArgs, &InitArgData{
			Name:         m.VarName,
			Ref:          m.VarName,
			FieldName:    m.FieldName,
			FieldPointer: endpoint.Request.Payload.IsPrimitivePointer(m.AttributeName, true),
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
	if hasRequestMessage {
		request.PayloadMessage = collector.collect(endpoint.Request.ProtoMessage)
	}
	switch {
	case endpoint.Request.ProtoStreamEnvelope != nil:
		request.Message = collector.collect(endpoint.Request.ProtoStreamEnvelope)
		request.StreamEnvelope = buildStreamEnvelopeData(endpoint.Request.ProtoStreamEnvelope, request.Message, sd)
	case endpoint.Request.ProtoStreamingInput != nil && endpoint.Request.ProtoStreamingInput.Type != expr.Empty:
		request.Message = collector.collect(endpoint.Request.ProtoStreamingInput)
	default:
		request.Message = collector.collect(endpoint.Request.ProtoMessage)
	}
	return request
}

func (d *ServicesData) buildResponseData(endpoint *transportir.Endpoint, svc *service.Data, sd *ServiceData, collector *messageCollector) *ResponseData {
	result, svcCtx := resultContext(endpoint, sd)
	hdrs := extractMetadata(endpoint.Response.Headers, result, svc.Scope, *d)
	trlrs := extractMetadata(endpoint.Response.Trailers, result, svc.Scope, *d)
	response := &ResponseData{
		StatusCode:    statusCodeToGRPCConst(endpoint.Response.StatusCode),
		Description:   endpoint.Response.Description,
		Headers:       hdrs,
		Trailers:      trlrs,
		ServerConvert: d.buildResponseConvertData(endpoint, result, svcCtx, hdrs, trlrs, sd, true),
		ClientConvert: d.buildResponseConvertData(endpoint, result, svcCtx, hdrs, trlrs, sd, false),
	}
	if endpoint.Response.ProtoMessage.Type != expr.Empty || !endpoint.Stream.IsStreaming {
		response.Message = collector.collect(endpoint.Response.ProtoMessage)
	}
	return response
}

func partitionSecuritySchemes(endpoint *transportir.Endpoint, md *service.MethodData) (service.SchemesData, service.SchemesData) {
	expanded := service.ExpandRequirementSchemes(endpoint.Requirements, md.Requirements)
	_, grouped, fallback := service.PartitionSchemesByIn(expanded)
	msgSch := grouped["message"]
	metSch := append(service.SchemesData(nil), fallback...)
	metSch = append(metSch, grouped["metadata"]...)
	return msgSch, metSch
}
