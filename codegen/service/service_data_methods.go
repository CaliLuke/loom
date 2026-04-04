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

		isJSONRPC bool
		reqs      RequirementsData
		schemes   SchemesData
	)
	vname = scope.Unique(codegen.Goify(m.Name, true), "Endpoint")
	desc = m.Description
	if desc == "" {
		desc = codegen.Goify(m.Name, true) + " implements " + m.Name + "."
	}
	payloadData = buildMethodAttributeProjection(m.Payload, "payload", m.Service.Name, m.Name, d.Root.API.ExampleGenerator, scope)
	resultData = buildMethodAttributeProjection(m.Result, "result", m.Service.Name, m.Name, d.Root.API.ExampleGenerator, scope)
	errors, errorLocs = buildMethodErrorData(m.Errors, scope)

	_, isJSONRPC = m.Meta["jsonrpc"]
	isJSONRPCSSE, isJSONRPCWebSocket := d.classifyJSONRPCStreamTransport(m, isJSONRPC)

	reqs, schemes = BuildRequirementsData(m.EffectiveRequirements(), m)

	skipRequestBodyEncodeDecode, skipResponseBodyEncodeDecode := d.httpSkipBodyFlags(m)

	data := &MethodData{
		Name:                         m.Name,
		VarName:                      vname,
		Description:                  desc,
		Payload:                      payloadData.Name,
		PayloadLoc:                   payloadData.Location,
		PayloadDef:                   payloadData.Definition,
		PayloadRef:                   payloadData.Reference,
		PayloadDesc:                  payloadData.Description,
		PayloadEx:                    payloadData.Example,
		PayloadDefault:               m.Payload.DefaultValue,
		Result:                       resultData.Name,
		ResultLoc:                    resultData.Location,
		ResultDef:                    resultData.Definition,
		ResultRef:                    resultData.Reference,
		ResultDesc:                   resultData.Description,
		ResultEx:                     resultData.Example,
		Errors:                       errors,
		ErrorLocs:                    errorLocs,
		IsJSONRPC:                    isJSONRPC,
		IsJSONRPCSSE:                 isJSONRPCSSE,
		IsJSONRPCWebSocket:           isJSONRPCWebSocket,
		Requirements:                 reqs,
		Schemes:                      schemes,
		StreamKind:                   m.Stream,
		HasMixedResults:              m.HasMixedResults(),
		SkipRequestBodyEncodeDecode:  skipRequestBodyEncodeDecode,
		SkipResponseBodyEncodeDecode: skipResponseBodyEncodeDecode,
		RequestStruct:                vname + "RequestData",
		ResponseStruct:               vname + "ResponseData",
	}

	d.initStreamData(data, m, vname, resultData.Name, resultData.Reference, scope)
	return data
}

func buildMethodAttributeProjection(att *expr.AttributeExpr, kind, serviceName, methodName string, gen *expr.ExampleGenerator, scope *codegen.NameScope) methodAttributeProjection {
	if att == nil || att.Type == expr.Empty {
		return methodAttributeProjection{}
	}

	projection := methodAttributeProjection{
		Name:        scope.GoTypeName(att),
		Description: att.Description,
		Example:     att.Example(gen),
	}
	if dt, ok := att.Type.(expr.UserType); ok {
		projection.Definition = scope.GoTypeDef(dt.Attribute(), false, true)
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

func (d *ServicesData) httpSkipBodyFlags(m *expr.MethodExpr) (bool, bool) {
	for _, svc := range d.Root.API.HTTP.Services {
		if svc.Name() != m.Service.Name {
			continue
		}
		httpMethod := svc.Endpoint(m.Name)
		if httpMethod == nil {
			return false, false
		}
		return httpMethod.SkipRequestBodyEncodeDecode, httpMethod.SkipResponseBodyEncodeDecode
	}
	return false, false
}

// initStreamData initializes the streaming payload data structures and methods.
func (d *ServicesData) initStreamData(data *MethodData, m *expr.MethodExpr, vname, rname, resultRef string, scope *codegen.NameScope) {
	if !m.IsStreaming() && !m.HasMixedResults() {
		return
	}
	var (
		spayloadName string
		spayloadRef  string
		spayloadDef  string
		spayloadDesc string
		spayloadEx   any
		srname       = rname
		srref        = resultRef
	)

	if m.HasMixedResults() && m.StreamingResult != nil && m.StreamingResult.Type != expr.Empty {
		srname = scope.GoTypeName(m.StreamingResult)
		srref = scope.GoTypeRef(m.StreamingResult)
		data.StreamingResult = srname
		data.StreamingResultRef = srref
		if dt, ok := m.StreamingResult.Type.(expr.UserType); ok {
			data.StreamingResultDef = scope.GoTypeDef(dt.Attribute(), false, true)
		}
		data.StreamingResultDesc = m.StreamingResult.Description
		if data.StreamingResultDesc == "" {
			data.StreamingResultDesc = fmt.Sprintf("%s is the streaming result type of the %s service %s method.",
				srname, m.Service.Name, m.Name)
		}
		data.StreamingResultEx = m.StreamingResult.Example(d.Root.API.ExampleGenerator)
	}

	if m.StreamingPayload != nil && m.StreamingPayload.Type != expr.Empty {
		spayloadName = scope.GoTypeName(m.StreamingPayload)
		spayloadRef = scope.GoTypeRef(m.StreamingPayload)
		if dt, ok := m.StreamingPayload.Type.(expr.UserType); ok {
			spayloadDef = scope.GoTypeDef(dt.Attribute(), false, true)
		}
		spayloadDesc = m.StreamingPayload.Description
		if spayloadDesc == "" {
			spayloadDesc = fmt.Sprintf("%s is the streaming payload type of the %s service %s method.",
				spayloadName, m.Service.Name, m.Name)
		}
		spayloadEx = m.StreamingPayload.Example(d.Root.API.ExampleGenerator)
	}
	streamKind := streamDataKind(m)
	svrStream, cliStream := buildBaseStreamData(m, vname, srname, srref, scope, data.IsJSONRPC, data.IsJSONRPCSSE)
	applyJSONRPCStreamAdjustments(svrStream, m, resultRef, srname, data.IsJSONRPCSSE)
	if streamKind == expr.ClientStreamKind || streamKind == expr.BidirectionalStreamKind {
		switch streamKind {
		case expr.ClientStreamKind:
			if srref != "" {
				svrStream.SendName = "SendAndClose"
				svrStream.SendDesc = fmt.Sprintf("SendAndClose streams instances of %q and closes the stream.", srname)
				svrStream.SendWithContextName = "SendAndCloseWithContext"
				svrStream.SendWithContextDesc = fmt.Sprintf("SendAndCloseWithContext streams instances of %q and closes the stream with context.", srname)
				svrStream.MustClose = false
				cliStream.RecvName = "CloseAndRecv"
				cliStream.RecvDesc = fmt.Sprintf("CloseAndRecv stops sending messages to the stream and reads instances of %q from the stream.", srname)
				cliStream.RecvWithContextName = "CloseAndRecvWithContext"
				cliStream.RecvWithContextDesc = fmt.Sprintf("CloseAndRecvWithContext stops sending messages to the stream and reads instances of %q from the stream with context.", srname)
			} else {
				cliStream.MustClose = true
			}
		case expr.BidirectionalStreamKind:
			cliStream.MustClose = true
		}
		svrStream.RecvName = "Recv"
		svrStream.RecvDesc = fmt.Sprintf("Recv reads instances of %q from the stream.", spayloadName)
		svrStream.RecvWithContextName = "RecvWithContext"
		svrStream.RecvWithContextDesc = fmt.Sprintf("RecvWithContext reads instances of %q from the stream with context.", spayloadName)
		svrStream.RecvTypeName = spayloadName
		svrStream.RecvTypeRef = spayloadRef
		cliStream.SendName = "Send"
		cliStream.SendDesc = fmt.Sprintf("Send streams instances of %q.", spayloadName)
		cliStream.SendWithContextName = "SendWithContext"
		cliStream.SendWithContextDesc = fmt.Sprintf("SendWithContext streams instances of %q with context.", spayloadName)
		cliStream.SendTypeName = spayloadName
		cliStream.SendTypeRef = spayloadRef
	}
	data.ClientStream = cliStream
	data.ServerStream = svrStream
	data.StreamingPayload = spayloadName
	data.StreamingPayloadDef = spayloadDef
	data.StreamingPayloadRef = spayloadRef
	data.StreamingPayloadDesc = spayloadDesc
	data.StreamingPayloadEx = spayloadEx
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
