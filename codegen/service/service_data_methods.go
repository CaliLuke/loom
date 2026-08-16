package service

import (
	"fmt"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

// buildMethodData creates the data needed to render the given endpoint. It
// records the user types needed by the service definition in userTypes.
func (d *ServicesData) buildMethodData(m *expr.MethodExpr, scope *codegen.NameScope) *MethodData {
	var (
		vname string
		desc  string

		payloadData methodAttributeProjection
		resultData  methodAttributeProjection

		errors    []*ErrorInitData
		errorLocs map[string]*codegen.Location
	)
	vname = scope.Unique(codegen.Goify(m.Name, true), "Endpoint")
	desc = m.Description
	if desc == "" {
		desc = codegen.Goify(m.Name, true) + " implements " + m.Name + "."
	}
	payloadData = buildMethodAttributeProjection(m.Payload, "payload", m.Service.Name, m.Name, d.Root.API.ExampleGenerator, scope)
	resultData = buildMethodAttributeProjection(m.Result, "result", m.Service.Name, m.Name, d.Root.API.ExampleGenerator, scope)
	errors, errorLocs = buildMethodErrorData(m.Errors, scope)

	data := &MethodData{
		Name:                m.Name,
		VarName:             vname,
		Description:         desc,
		MethodPayloadData:   buildMethodPayloadData(m, payloadData),
		MethodResultData:    buildMethodResultData(resultData),
		MethodSecurityData:  buildMethodSecurityData(m, errors, errorLocs),
		MethodTransportData: d.buildMethodTransportData(m, vname),
		MethodStreamingData: buildMethodStreamingData(m),
	}

	d.initStreamData(data, m, vname, resultData.Name, resultData.Reference, scope)
	return data
}

func buildMethodPayloadData(m *expr.MethodExpr, payloadData methodAttributeProjection) MethodPayloadData {
	var payloadDefault any
	if m.Payload != nil {
		payloadDefault = m.Payload.DefaultValue
	}
	return MethodPayloadData{
		Payload:        payloadData.Name,
		PayloadLoc:     payloadData.Location,
		PayloadDef:     payloadData.Definition,
		PayloadRef:     payloadData.Reference,
		PayloadDesc:    payloadData.Description,
		PayloadEx:      payloadData.Example,
		PayloadDefault: payloadDefault,
	}
}

func buildMethodResultData(resultData methodAttributeProjection) MethodResultData {
	return MethodResultData{
		Result:     resultData.Name,
		ResultLoc:  resultData.Location,
		ResultDef:  resultData.Definition,
		ResultRef:  resultData.Reference,
		ResultDesc: resultData.Description,
		ResultEx:   resultData.Example,
	}
}

func buildMethodSecurityData(m *expr.MethodExpr, errors []*ErrorInitData, errorLocs map[string]*codegen.Location) MethodSecurityData {
	reqs, schemes := BuildRequirementsData(m.EffectiveRequirements(), m)
	return MethodSecurityData{
		Errors:       errors,
		ErrorLocs:    errorLocs,
		Requirements: reqs,
		Schemes:      schemes,
	}
}

func (d *ServicesData) buildMethodTransportData(m *expr.MethodExpr, vname string) MethodTransportData {
	_, isJSONRPC := m.Meta["jsonrpc"]
	isJSONRPCSSE, isJSONRPCWebSocket := d.classifyJSONRPCStreamTransport(m, isJSONRPC)
	skipRequestBodyEncodeDecode, skipResponseBodyEncodeDecode, fileResponse := d.httpTransportFlags(m)
	return MethodTransportData{
		IsJSONRPC:                    isJSONRPC,
		IsJSONRPCSSE:                 isJSONRPCSSE,
		IsJSONRPCWebSocket:           isJSONRPCWebSocket,
		SkipRequestBodyEncodeDecode:  skipRequestBodyEncodeDecode,
		SkipResponseBodyEncodeDecode: skipResponseBodyEncodeDecode,
		FileResponse:                 fileResponse,
		RequestStruct:                vname + "RequestData",
		ResponseStruct:               vname + "ResponseData",
		FileResponseStruct:           vname + "FileResponseData",
	}
}

func buildMethodStreamingData(m *expr.MethodExpr) MethodStreamingData {
	return MethodStreamingData{
		StreamKind:      m.Stream,
		HasMixedResults: m.HasMixedResults(),
	}
}

func buildMethodAttributeProjection(att *expr.AttributeExpr, kind, serviceName, methodName string, gen *expr.ExampleGenerator, scope *codegen.NameScope) methodAttributeProjection {
	if att == nil || att.Type == expr.Empty {
		return methodAttributeProjection{}
	}

	projection := methodAttributeProjection{
		Name:        scope.GoValueTypeName(att),
		Description: att.Description,
		Example:     att.Example(gen),
	}
	if dt, ok := att.Type.(expr.UserType); ok {
		projection.Definition = scope.GoValueTypeDef(dt.Attribute(), false, true)
		projection.Location = codegen.UserTypeLocation(dt)
	}
	projection.Reference = scope.GoFullTypeRef(att, projection.Location.PackageName())
	if projection.Description == "" {
		projection.Description = fmt.Sprintf("%s is the %s type of the %s service %s method.",
			projection.Name, kind, serviceName, methodName)
	}
	return projection
}

func buildMethodErrorData(methodErrors []*expr.ErrorExpr, scope *codegen.NameScope) ([]*ErrorInitData, map[string]*codegen.Location) {
	if len(methodErrors) == 0 {
		return nil, nil
	}
	errors := make([]*ErrorInitData, len(methodErrors))
	errorLocs := make(map[string]*codegen.Location, len(methodErrors))
	for i, methodError := range methodErrors {
		errors[i] = buildErrorInitData(methodError, scope)
		errorLocs[methodError.Name] = codegen.UserTypeLocation(methodError.Type)
	}
	return errors, errorLocs
}

func (d *ServicesData) classifyJSONRPCStreamTransport(m *expr.MethodExpr, isJSONRPC bool) (bool, bool) {
	if !isJSONRPC || !m.IsStreaming() {
		return false, false
	}
	httpJSONRPCSvc := d.Root.API.JSONRPC.HTTPExpr.Service(m.Service.Name)
	if httpJSONRPCSvc == nil {
		return false, false
	}
	for _, e := range httpJSONRPCSvc.HTTPEndpoints {
		if e.MethodExpr.Name != m.Name {
			continue
		}
		if e.SSE != nil {
			return true, false
		}
		return false, true
	}
	return false, false
}

func (d *ServicesData) httpTransportFlags(m *expr.MethodExpr) (bool, bool, bool) {
	for _, svc := range d.Root.API.HTTP.Services {
		if svc.Name() != m.Service.Name {
			continue
		}
		httpMethod := svc.Endpoint(m.Name)
		if httpMethod == nil {
			return false, false, false
		}
		return httpMethod.SkipRequestBodyEncodeDecode, httpMethod.SkipResponseBodyEncodeDecode, httpMethod.FileResponse
	}
	return false, false, false
}

func hasFileResponse(methods []*MethodData) bool {
	for _, method := range methods {
		if method.FileResponse {
			return true
		}
	}
	return false
}

// initStreamData initializes the streaming payload data structures and methods.
func (d *ServicesData) initStreamData(data *MethodData, m *expr.MethodExpr, vname, rname, resultRef string, scope *codegen.NameScope) {
	if !m.IsStreaming() && !m.HasMixedResults() {
		return
	}
	sresult := d.buildStreamingResultData(data, m, rname, resultRef, scope)
	spayload := d.buildStreamingPayloadData(m, scope)
	streamKind := streamDataKind(m)
	svrStream, cliStream := buildBaseStreamData(m, vname, sresult.Name, sresult.Ref, scope, data.IsJSONRPC, data.IsJSONRPCSSE)
	applyJSONRPCStreamAdjustments(svrStream, m, resultRef, sresult.Name, data.IsJSONRPCSSE)
	applyStreamDirectionData(streamKind, svrStream, cliStream, sresult, spayload)
	data.ClientStream = cliStream
	data.ServerStream = svrStream
	data.StreamingPayload = spayload.Name
	data.StreamingPayloadDef = spayload.Def
	data.StreamingPayloadRef = spayload.Ref
	data.StreamingPayloadDesc = spayload.Desc
	data.StreamingPayloadEx = spayload.Example
}

type streamAttributeData struct {
	Name    string
	Ref     string
	Def     string
	Desc    string
	Example any
}

func (d *ServicesData) buildStreamingResultData(data *MethodData, m *expr.MethodExpr, rname, resultRef string, scope *codegen.NameScope) streamAttributeData {
	sresult := streamAttributeData{Name: rname, Ref: resultRef}
	if !m.HasMixedResults() || m.StreamingResult == nil || m.StreamingResult.Type == expr.Empty {
		return sresult
	}
	sresult = buildStreamAttributeData(m.StreamingResult, m, scope, d.Root.API.ExampleGenerator)
	data.StreamingResult = sresult.Name
	data.StreamingResultRef = sresult.Ref
	data.StreamingResultDef = sresult.Def
	data.StreamingResultDesc = sresult.Desc
	data.StreamingResultEx = sresult.Example
	return sresult
}

func (d *ServicesData) buildStreamingPayloadData(m *expr.MethodExpr, scope *codegen.NameScope) streamAttributeData {
	if m.StreamingPayload == nil || m.StreamingPayload.Type == expr.Empty {
		return streamAttributeData{}
	}
	return buildStreamAttributeData(m.StreamingPayload, m, scope, d.Root.API.ExampleGenerator)
}

func buildStreamAttributeData(att *expr.AttributeExpr, m *expr.MethodExpr, scope *codegen.NameScope, examples *expr.ExampleGenerator) streamAttributeData {
	data := streamAttributeData{
		Name:    scope.GoValueTypeName(att),
		Ref:     scope.GoTypeRef(att),
		Desc:    att.Description,
		Example: att.Example(examples),
	}
	if dt, ok := att.Type.(expr.UserType); ok {
		data.Def = scope.GoValueTypeDef(dt.Attribute(), false, true)
	}
	if data.Desc == "" {
		data.Desc = streamAttributeDescription(data.Name, att, m)
	}
	return data
}

func streamAttributeDescription(name string, att *expr.AttributeExpr, m *expr.MethodExpr) string {
	role := "result"
	if att == m.StreamingPayload {
		role = "payload"
	}
	return fmt.Sprintf("%s is the streaming %s type of the %s service %s method.", name, role, m.Service.Name, m.Name)
}

func applyStreamDirectionData(kind expr.StreamKind, svrStream, cliStream *StreamData, sresult, spayload streamAttributeData) {
	if kind != expr.ClientStreamKind && kind != expr.BidirectionalStreamKind {
		return
	}
	applyClientOrBidirectionalAdjustments(kind, svrStream, cliStream, sresult)
	applyStreamingPayloadIO(svrStream, cliStream, spayload)
}

func applyClientOrBidirectionalAdjustments(kind expr.StreamKind, svrStream, cliStream *StreamData, sresult streamAttributeData) {
	switch kind {
	case expr.ClientStreamKind:
		if sresult.Ref == "" {
			cliStream.MustClose = true
			return
		}
		svrStream.SendName = "SendAndClose"
		svrStream.SendDesc = fmt.Sprintf("SendAndClose streams instances of %q and closes the stream.", sresult.Name)
		svrStream.SendWithContextName = "SendAndCloseWithContext"
		svrStream.SendWithContextDesc = fmt.Sprintf("SendAndCloseWithContext streams instances of %q and closes the stream with context.", sresult.Name)
		svrStream.MustClose = false
		cliStream.RecvName = "CloseAndRecv"
		cliStream.RecvDesc = fmt.Sprintf("CloseAndRecv stops sending messages to the stream and reads instances of %q from the stream.", sresult.Name)
		cliStream.RecvWithContextName = "CloseAndRecvWithContext"
		cliStream.RecvWithContextDesc = fmt.Sprintf("CloseAndRecvWithContext stops sending messages to the stream and reads instances of %q from the stream with context.", sresult.Name)
	case expr.BidirectionalStreamKind:
		cliStream.MustClose = true
	}
}

func applyStreamingPayloadIO(svrStream, cliStream *StreamData, spayload streamAttributeData) {
	svrStream.RecvName = "Recv"
	svrStream.RecvDesc = fmt.Sprintf("Recv reads instances of %q from the stream.", spayload.Name)
	svrStream.RecvWithContextName = "RecvWithContext"
	svrStream.RecvWithContextDesc = fmt.Sprintf("RecvWithContext reads instances of %q from the stream with context.", spayload.Name)
	svrStream.RecvTypeName = spayload.Name
	svrStream.RecvTypeRef = spayload.Ref
	cliStream.SendName = "Send"
	cliStream.SendDesc = fmt.Sprintf("Send streams instances of %q.", spayload.Name)
	cliStream.SendWithContextName = "SendWithContext"
	cliStream.SendWithContextDesc = fmt.Sprintf("SendWithContext streams instances of %q with context.", spayload.Name)
	cliStream.SendTypeName = spayload.Name
	cliStream.SendTypeRef = spayload.Ref
}

func streamDataKind(m *expr.MethodExpr) expr.StreamKind {
	if m.HasMixedResults() && !m.IsStreaming() {
		return expr.ServerStreamKind
	}
	return m.Stream
}

func buildBaseStreamData(m *expr.MethodExpr, vname, srname, srref string, scope *codegen.NameScope, isJSONRPC, isJSONRPCSSE bool) (*StreamData, *StreamData) {
	endpointStruct := vname + "EndpointInput"
	if isJSONRPC && m.IsStreaming() && !isJSONRPCSSE && m.Stream == expr.ClientStreamKind {
		endpointStruct = ""
	}
	streamKind := streamDataKind(m)
	svrStream := &StreamData{
		Interface:           vname + "ServerStream",
		VarName:             scope.Unique(codegen.Goify(m.Name, true), "ServerStream"),
		EndpointStruct:      endpointStruct,
		Kind:                streamKind,
		SendName:            "Send",
		SendDesc:            fmt.Sprintf("Send streams instances of %q.", srname),
		SendWithContextName: "SendWithContext",
		SendWithContextDesc: fmt.Sprintf("SendWithContext streams instances of %q with context.", srname),
		SendTypeName:        srname,
		SendTypeRef:         srref,
		MustClose:           true,
	}
	cliStream := &StreamData{
		Interface:           vname + "ClientStream",
		VarName:             scope.Unique(codegen.Goify(m.Name, true), "ClientStream"),
		Kind:                streamKind,
		RecvName:            "Recv",
		RecvDesc:            fmt.Sprintf("Recv reads instances of %q from the stream.", srname),
		RecvWithContextName: "RecvWithContext",
		RecvWithContextDesc: fmt.Sprintf("RecvWithContext reads instances of %q from the stream with context.", srname),
		RecvTypeName:        srname,
		RecvTypeRef:         srref,
	}
	return svrStream, cliStream
}

func applyJSONRPCStreamAdjustments(svrStream *StreamData, m *expr.MethodExpr, resultRef, srname string, isJSONRPCSSE bool) {
	if !isJSONRPCSSE || m.Stream != expr.ServerStreamKind || resultRef == "" {
		return
	}
	svrStream.SendAndCloseName = "SendAndClose"
	svrStream.SendAndCloseDesc = fmt.Sprintf("SendAndClose sends a final response with %q and closes the stream.", srname)
	svrStream.SendWithContextName = "Send"
	svrStream.RecvWithContextName = "Recv"
	svrStream.SendDesc = fmt.Sprintf("Send streams JSON-RPC notifications with %q. Notifications do not expect a response.", srname)
}

// BuildSchemeData builds the scheme data for the given scheme and method expr.
func BuildSchemeData(s *expr.SchemeExpr, m *expr.MethodExpr) *SchemeData {
	if !expr.IsObject(m.Payload.Type) {
		return nil
	}
	scopes := schemeScopeNames(s)
	switch s.Kind {
	case expr.BasicAuthKind:
		userAtt := expr.TaggedAttribute(m.Payload, "security:username")
		user := codegen.Goify(userAtt, true)
		passAtt := expr.TaggedAttribute(m.Payload, "security:password")
		pass := codegen.Goify(passAtt, true)
		return &SchemeData{
			Type:             s.Kind.String(),
			SchemeName:       s.SchemeName,
			UsernameAttr:     userAtt,
			UsernameField:    user,
			UsernamePointer:  m.Payload.IsPrimitivePointer(userAtt, true),
			UsernameRequired: m.Payload.IsRequired(userAtt),
			PasswordAttr:     passAtt,
			PasswordField:    pass,
			PasswordPointer:  m.Payload.IsPrimitivePointer(passAtt, true),
			PasswordRequired: m.Payload.IsRequired(passAtt),
			Scopes:           scopes,
		}
	case expr.APIKeyKind:
		return buildCredentialSchemeData(s, m, "security:apikey:"+s.SchemeName, scopes)
	case expr.JWTKind:
		return buildCredentialSchemeData(s, m, "security:token", scopes)
	case expr.OAuth2Kind:
		data := buildCredentialSchemeData(s, m, "security:accesstoken", scopes)
		if data != nil {
			data.Flows = s.Flows
		}
		return data
	}
	return nil
}

func schemeScopeNames(s *expr.SchemeExpr) []string {
	if len(s.Scopes) == 0 {
		return nil
	}
	scopes := make([]string, len(s.Scopes))
	for i, scope := range s.Scopes {
		scopes[i] = scope.Name
	}
	return scopes
}

func buildCredentialSchemeData(s *expr.SchemeExpr, m *expr.MethodExpr, tag string, scopes []string) *SchemeData {
	keyAtt := expr.TaggedAttribute(m.Payload, tag)
	if keyAtt == "" {
		if isTransportOwnedCookieSchemeData(s, m) {
			return &SchemeData{
				Type:           s.Kind.String(),
				Name:           s.Name,
				SchemeName:     s.SchemeName,
				Scopes:         scopes,
				In:             s.In,
				TransportOwned: true,
			}
		}
		return nil
	}
	return &SchemeData{
		Type:         s.Kind.String(),
		Name:         s.Name,
		SchemeName:   s.SchemeName,
		CredField:    codegen.Goify(keyAtt, true),
		CredPointer:  m.Payload.IsPrimitivePointer(keyAtt, true),
		CredRequired: m.Payload.IsRequired(keyAtt),
		KeyAttr:      keyAtt,
		Scopes:       scopes,
		In:           s.In,
	}
}

func isTransportOwnedCookieSchemeData(s *expr.SchemeExpr, m *expr.MethodExpr) bool {
	if s == nil || m == nil || s.Kind != expr.APIKeyKind {
		return false
	}
	for _, sessionAuth := range m.EffectiveSessionAuths() {
		for _, transport := range sessionAuth.Transports {
			if transport == nil || transport.Kind != expr.SessionCookieTransportKind || transport.PayloadOwned() || transport.Scheme == nil {
				continue
			}
			if transport.Scheme.SchemeName == s.SchemeName {
				return true
			}
		}
	}
	return false
}
