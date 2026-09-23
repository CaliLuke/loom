package codegen

import (
	"fmt"
	"strings"

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
	if err := checkProtoNames(svc, irService); err != nil {
		panic(err)
	}
	scope := codegen.NewNameScope()
	pkg := svc.PathName + pbPkgName
	svcVarN := scope.HashedUnique(gs.ServiceExpr, protoServiceName(svc.StructName))
	goName := protoBufIdentifier(svcVarN, true, true)
	sd = &ServiceData{
		Service:             svc,
		Name:                svcVarN,
		GoName:              goName,
		Description:         svc.Description,
		PkgName:             pkg,
		ProtoPkg:            pkgName(gs, svc.PathName),
		ServerStruct:        "Server",
		ClientStruct:        "Client",
		ServerInit:          "New",
		ClientInit:          "NewClient",
		ServerInterface:     goName + "Server",
		ClientInterface:     goName + "Client",
		ClientInterfaceInit: fmt.Sprintf("%s.New%sClient", pkg, goName),
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
	responseContractCases, responseContractWarnings := buildResponseContractCaseData(endpointIR, sd.ProtoPkg)
	md := svc.Method(endpointIR.Name)
	payloadDesc := service.BuildPayloadDescriptor(svc, md, endpointIR.Request.Payload)
	resultDesc := service.BuildResultDescriptor(svc, md, endpointIR.Response.Result)
	errors := d.buildErrorsData(endpointIR, sd)
	collector.collectErrorMessages(endpointIR)
	request := d.buildRequestData(endpointIR, svc, sd, collector)
	response := d.buildResponseData(endpointIR, svc, sd, collector)
	msgSch, metSch := partitionSecuritySchemes(endpointIR, md)
	ed := &EndpointData{
		ServiceName:               svc.Name,
		RPCName:                   protoServiceName(md.VarName),
		RPCGoName:                 protoBufIdentifier(protoServiceName(md.VarName), true, true),
		PkgName:                   sd.PkgName,
		ServicePkgName:            svc.PkgName,
		Method:                    md,
		PayloadType:               endpointIR.Request.Payload.Type,
		PayloadRef:                payloadDesc.Ref,
		ResultRef:                 resultDesc.Declared.Ref,
		ViewedResultRef:           resultDesc.ViewedRef,
		Request:                   request,
		Response:                  response,
		MessageSchemes:            msgSch,
		MetadataSchemes:           metSch,
		Errors:                    errors,
		ResponseContractCasesInit: fmt.Sprintf("%sResponseContractCases", md.VarName),
		ResponseContractCases:     responseContractCases,
		ResponseContractWarnings:  responseContractWarnings,
		ServerStruct:              sd.ServerStruct,
		ServerInterface:           sd.ServerInterface,
		ClientStruct:              sd.ClientStruct,
		ClientInterface:           sd.ClientInterface,
	}
	sd.Endpoints = append(sd.Endpoints, ed)
	if endpointIR.Stream.IsStreaming {
		ed.ServerStream = d.buildStreamData(endpointIR, sd, true)
		ed.ClientStream = d.buildStreamData(endpointIR, sd, false)
	}
}

func buildResponseContractCaseData(endpoint *transportir.Endpoint, protoPkg string) ([]*ResponseContractCaseData, []string) {
	analysis := transportir.AnalyzeResponseContractCases(endpoint)
	if !analysis.Supported() {
		return nil, responseContractLimitationWarnings(endpoint, analysis.Limitations)
	}
	cases := make([]*ResponseContractCaseData, 0, len(analysis.Cases))
	for _, contractCase := range analysis.Cases {
		data := &ResponseContractCaseData{
			ID:               contractCase.ID,
			IsError:          contractCase.Kind == transportir.ResponseContractError,
			StatusCode:       statusCodeToGRPCConst(contractCase.StatusCode),
			MessageType:      qualifyResponseContractMessage(protoPkg, contractCase.MessageType),
			ErrorName:        contractCase.ErrorName,
			DetailType:       qualifyResponseContractMessage(protoPkg, contractCase.DetailType),
			RequiredHeaders:  append([]string(nil), contractCase.RequiredHeaders...),
			RequiredTrailers: append([]string(nil), contractCase.RequiredTrailers...),
		}
		if contractCase.Stream != nil {
			data.Stream = &ResponseContractStreamData{
				Direction: contractCase.Stream.Direction,
				Terminal:  contractCase.Stream.Terminal,
			}
		}
		cases = append(cases, data)
	}
	return cases, nil
}

func qualifyResponseContractMessage(protoPkg, message string) string {
	if message == "" || strings.Contains(message, ".") {
		return message
	}
	return protoPkg + "." + message
}

func responseContractLimitationWarnings(endpoint *transportir.Endpoint, limitations []transportir.ResponseContractLimitation) []string {
	if endpoint == nil || endpoint.Service == nil || len(limitations) == 0 {
		return nil
	}
	prefix := fmt.Sprintf("gRPC response contract omitted for %s.%s", endpoint.Service.Name, endpoint.Name)
	warnings := make([]string, 0, len(limitations))
	for _, limitation := range limitations {
		warnings = append(warnings, fmt.Sprintf("%s: %s: %s", prefix, limitation.Code, limitation.Detail))
	}
	return warnings
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
		registerProtoMessage(sd, grpcErr.Response.ProtoMessage, endpoint.Name)
	}
	for _, message := range []*expr.AttributeExpr{
		endpoint.Request.ProtoMessage,
		endpoint.Request.ProtoStreamingInput,
		endpoint.Request.ProtoStreamEnvelope,
		endpoint.Response.ProtoMessage,
	} {
		if message != nil {
			registerProtoMessage(sd, message, endpoint.Name)
		}
	}
}

func (d *ServicesData) buildRequestData(endpoint *transportir.Endpoint, svc *service.Data, sd *ServiceData, collector *messageCollector) *RequestData {
	reqMD := extractMetadata(endpoint.Request.Metadata, endpoint.Request.Payload, svc.Scope, codegen.NewNameScope(), *d)
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
	vars := codegen.NewNameScope()
	hdrs := extractMetadata(endpoint.Response.Headers, result, svc.Scope, vars, *d)
	trlrs := extractMetadata(endpoint.Response.Trailers, result, svc.Scope, vars, *d)
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

// protoServiceName returns the protocol buffer name of a service or rpc
// whose Go name is goName. ASCII names are used unchanged so that the wire
// path of existing services is stable. Protocol buffer identifiers are ASCII
// only, so any other rune becomes a word separator, as in protoBufify.
func protoServiceName(goName string) string {
	ascii := strings.Map(asciiIdentifierRune, goName)
	if ascii == goName {
		return goName
	}
	if name := codegen.Goify(ascii, true); name != "" {
		return name
	}
	return "Val"
}

// checkProtoNames returns an error when two methods of the service map to the
// same protocol buffer rpc name.
func checkProtoNames(svc *service.Data, irService *transportir.Service) error {
	rpcs := make(map[string]string, len(irService.Endpoints))
	for _, endpoint := range irService.Endpoints {
		rpc := protoServiceName(svc.Method(endpoint.Name).VarName)
		if other, ok := rpcs[rpc]; ok {
			return fmt.Errorf("methods %q and %q of service %q both map to protocol buffer rpc %q", other, endpoint.Name, svc.Name, rpc)
		}
		rpcs[rpc] = endpoint.Name
	}
	return nil
}

// registerProtoMessage records the top-level protocol buffer message att
// generated for method. Generated messages are identified by name, so it
// panics when a message of another method has the same name but different
// fields: one of the two shapes would be silently lost. Messages with the
// same name and fields are shared.
func registerProtoMessage(sd *ServiceData, att *expr.AttributeExpr, method string) {
	ut, ok := att.Type.(expr.UserType)
	if !ok {
		return
	}
	if sd.protoMessages == nil {
		sd.protoMessages = make(map[string]protoMessageShape)
	}
	shape := protoMessageShape{method: method, hash: protoMessageHash(ut)}
	other, ok := sd.protoMessages[ut.Name()]
	if !ok {
		sd.protoMessages[ut.Name()] = shape
		return
	}
	if other.hash != shape.hash {
		panic(fmt.Errorf("methods %q and %q of service %q both map to protocol buffer message %q with different fields",
			other.method, method, sd.Service.Name, protoBufify(ut.Name(), true, true)))
	}
}

// protoMessageHash returns a hash of the fields of the message ut and of every
// message reachable from it, including the numbers and requiredness of the
// fields at every depth.
func protoMessageHash(ut expr.UserType) string {
	var b strings.Builder
	b.WriteString(expr.Hash(ut, false, false, false))
	writeProtoFieldShapes(&b, &expr.AttributeExpr{Type: ut}, make(map[expr.UserType]struct{}))
	return b.String()
}

// writeProtoFieldShapes writes the name, number and requiredness of the
// fields of every object reachable from att to b.
func writeProtoFieldShapes(b *strings.Builder, att *expr.AttributeExpr, seen map[expr.UserType]struct{}) {
	switch dt := att.Type.(type) {
	case expr.UserType:
		if _, ok := seen[dt]; ok {
			return
		}
		seen[dt] = struct{}{}
		fmt.Fprintf(b, "|%s{", dt.Name())
		writeProtoFieldShapes(b, dt.Attribute(), seen)
		b.WriteString("}")
	case *expr.Object:
		for _, nat := range *dt {
			fmt.Fprintf(b, "|%q=%d,%t", nat.Name, rpcTag(nat.Attribute), att.IsRequired(nat.Name))
			writeProtoFieldShapes(b, nat.Attribute, seen)
		}
	case *expr.Array:
		writeProtoFieldShapes(b, dt.ElemType, seen)
	case *expr.Map:
		writeProtoFieldShapes(b, dt.KeyType, seen)
		writeProtoFieldShapes(b, dt.ElemType, seen)
	case *expr.Union:
		for _, nat := range dt.Values {
			fmt.Fprintf(b, "|%q=%d", nat.Name, rpcTag(nat.Attribute))
			writeProtoFieldShapes(b, nat.Attribute, seen)
		}
	}
}
