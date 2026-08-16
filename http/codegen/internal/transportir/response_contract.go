package transportir

import (
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/CaliLuke/loom/expr"
)

type (
	// ResponseContractCaseKind identifies whether a contract case exercises a
	// successful result or a service error.
	ResponseContractCaseKind string

	// ResponseContractTransport identifies the wire protocol validated by a
	// response contract case.
	ResponseContractTransport string

	// ResponseContractLimitationCode identifies an endpoint feature that the
	// HTTP response contract analysis intentionally does not support.
	ResponseContractLimitationCode string

	// ResponseContractAnalysis describes the exhaustive response cases for a
	// supported HTTP endpoint. Cases is empty when Limitations is not.
	ResponseContractAnalysis struct {
		// Cases lists the declared response branches in design order.
		Cases []*ResponseContractCase
		// Limitations explains why cases could not be produced.
		Limitations []ResponseContractLimitation
	}

	// ResponseContractLimitation explains why an endpoint is outside the
	// response contract analysis scope.
	ResponseContractLimitation struct {
		// Code identifies the unsupported endpoint feature.
		Code ResponseContractLimitationCode
		// Detail explains the limitation for diagnostics.
		Detail string
	}

	// ResponseContractCase describes one declared HTTP response branch.
	ResponseContractCase struct {
		// ID is stable across generation when the service contract is unchanged.
		ID string
		// Kind distinguishes successful results from service errors.
		Kind ResponseContractCaseKind
		// Transport identifies the response protocol.
		Transport ResponseContractTransport
		// StatusCode is the declared HTTP response status.
		StatusCode int
		// ErrorName is the declared service error name for error cases.
		ErrorName string
		// TagName is the result field selecting a tagged success response.
		TagName string
		// TagValue is the result field value selecting a tagged success response.
		TagValue string
		// ContentTypes lists the declared response media types.
		ContentTypes []string
		// HasBody reports whether the response contract asserts a body media type.
		HasBody bool
		// Headers lists the declared response header assertions.
		Headers []ResponseContractHeader
		// Cookies lists the declared response cookie assertions.
		Cookies []ResponseContractCookie
		// Multipart describes the designed multipart request, if present.
		Multipart *ResponseContractMultipartRequest
		// SSE describes stream assertions for an SSE success case.
		SSE *ResponseContractSSE
		// WebSocket describes stream assertions for a WebSocket success case.
		WebSocket *ResponseContractWebSocket
	}

	// ResponseContractMultipartRequest describes a multipart request shape that
	// can be represented without an application-owned codec.
	ResponseContractMultipartRequest struct {
		// ContentType is the request media type.
		ContentType string
		// Parts lists the designed multipart fields in body order.
		Parts []ResponseContractMultipartPart
	}

	// ResponseContractMultipartPart describes one designed multipart field.
	ResponseContractMultipartPart struct {
		// Name is the multipart form field name.
		Name string
		// MediaType is the default media type for the part value.
		MediaType string
		// Required reports whether the request body requires the part.
		Required bool
	}

	// ResponseContractSSE describes a generated Server-Sent Events contract.
	ResponseContractSSE struct {
		// Direction is the designed stream direction.
		Direction string
		// MessageType is the designed streaming result type name.
		MessageType string
		// DataField is the result field encoded into SSE data, if any.
		DataField string
		// DataEncoding identifies whether SSE data is JSON or plain text.
		DataEncoding string
		// IDField is the result field encoded into SSE id, if any.
		IDField string
		// EventField is the result field encoded into SSE event, if any.
		EventField string
		// RetryField is the result field encoded into SSE retry, if any.
		RetryField string
		// IDRequired reports whether every observed event must include an ID.
		IDRequired bool
		// EventTypeRequired reports whether every observed event must include a type.
		EventTypeRequired bool
		// EventTypes lists allowed projection discriminator values, if constrained.
		EventTypes []string
		// Terminal identifies the expected stream completion behavior.
		Terminal string
	}

	// ResponseContractWebSocket describes a generated WebSocket contract.
	ResponseContractWebSocket struct {
		// Direction is the designed stream direction.
		Direction string
		// InboundMessageType is the designed client-to-server message type name.
		InboundMessageType string
		// OutboundMessageType is the designed server-to-client message type name.
		OutboundMessageType string
		// HandshakeHeaders lists required WebSocket upgrade response headers.
		HandshakeHeaders []string
		// Terminal identifies the expected stream completion behavior.
		Terminal string
	}

	// ResponseContractHeader describes a declared response header assertion.
	ResponseContractHeader struct {
		// Name is the result or error attribute mapped to the header.
		Name string
		// HTTPName is the wire header name.
		HTTPName string
		// Required reports whether the mapped attribute is required.
		Required bool
	}

	// ResponseContractCookie describes a declared response cookie assertion.
	ResponseContractCookie struct {
		// Name is the result or error attribute mapped to the cookie.
		Name string
		// HTTPName is the wire cookie name.
		HTTPName string
		// Required reports whether the mapped attribute is required.
		Required bool
		// Path is the declared cookie Path attribute.
		Path string
		// Domain is the declared cookie Domain attribute.
		Domain string
		// MaxAge is the declared cookie Max-Age attribute.
		MaxAge string
		// Secure reports whether the cookie is restricted to secure transports.
		Secure bool
		// HTTPOnly reports whether scripts are denied cookie access.
		HTTPOnly bool
		// SameSite is the declared cookie SameSite policy.
		SameSite expr.CookieSameSiteValue
	}
)

const (
	// ResponseContractSuccess identifies a successful response contract case.
	ResponseContractSuccess ResponseContractCaseKind = "success"
	// ResponseContractError identifies an error response contract case.
	ResponseContractError ResponseContractCaseKind = "error"
	// ResponseContractHTTP identifies an ordinary HTTP response contract.
	ResponseContractHTTP ResponseContractTransport = "http"
	// ResponseContractSSE identifies a Server-Sent Events response contract.
	ResponseContractSSETransport ResponseContractTransport = "sse"
	// ResponseContractWebSocketTransport identifies a WebSocket response contract.
	ResponseContractWebSocketTransport ResponseContractTransport = "websocket"

	// ResponseContractMissingEndpoint indicates that no endpoint was supplied.
	ResponseContractMissingEndpoint ResponseContractLimitationCode = "missing_endpoint"
	// ResponseContractMissingIdentity indicates that a stable service or method
	// identity is unavailable.
	ResponseContractMissingIdentity ResponseContractLimitationCode = "missing_identity"
	// ResponseContractJSONRPC indicates that the endpoint uses JSON-RPC rather
	// than plain HTTP semantics.
	ResponseContractJSONRPC ResponseContractLimitationCode = "jsonrpc"
	// ResponseContractStreaming indicates that a streaming endpoint shape is
	// outside the supported SSE or WebSocket response contract scope.
	ResponseContractStreaming ResponseContractLimitationCode = "streaming"
	// ResponseContractRedirect indicates that the endpoint is a redirect.
	ResponseContractRedirect ResponseContractLimitationCode = "redirect"
	// ResponseContractMultipart indicates that request construction depends on
	// multipart handling outside the unary contract case.
	ResponseContractMultipart ResponseContractLimitationCode = "multipart"
	// ResponseContractRawRequestBody indicates that the service owns the raw
	// request body stream.
	ResponseContractRawRequestBody ResponseContractLimitationCode = "raw_request_body"
	// ResponseContractRawResponseBody indicates that the service owns the raw
	// response body stream.
	ResponseContractRawResponseBody ResponseContractLimitationCode = "raw_response_body"
	// ResponseContractDuplicateCaseID indicates that distinct response branches
	// produced the same stable identifier.
	ResponseContractDuplicateCaseID ResponseContractLimitationCode = "duplicate_case_id"
)

// AnalyzeResponseContractCases builds deterministic, exhaustive descriptors
// for the declared response branches of a supported HTTP endpoint.
func AnalyzeResponseContractCases(endpoint *Endpoint) *ResponseContractAnalysis {
	analysis := &ResponseContractAnalysis{}
	analysis.Limitations = responseContractLimitations(endpoint)
	if len(analysis.Limitations) > 0 {
		return analysis
	}

	serviceName := endpoint.Service.Name
	methodName := endpoint.MethodName
	multipart, _ := responseContractMultipartRequest(endpoint.Request)
	analysis.Cases = make([]*ResponseContractCase, 0, len(endpoint.Response.Responses)+len(endpoint.Response.ErrorResponses))
	for _, response := range endpoint.Response.Responses {
		contractResponse := response
		if endpoint.Stream != nil && endpoint.Stream.IsWebSocket {
			webSocketResponse := *response
			webSocketResponse.StatusCode = endpoint.Stream.HandshakeStatus
			webSocketResponse.ContentTypes = nil
			webSocketResponse.Body = &expr.AttributeExpr{Type: expr.Empty}
			contractResponse = &webSocketResponse
		}
		contractCase := newResponseContractCase(serviceName, methodName, endpoint.Response.FileResponse, contractResponse)
		contractCase.Multipart = multipart
		if endpoint.Stream != nil && endpoint.Stream.IsSSE {
			contractCase.Transport = ResponseContractSSETransport
			contractCase.SSE = newResponseContractSSE(endpoint.Stream)
		} else if endpoint.Stream != nil && endpoint.Stream.IsWebSocket {
			contractCase.Transport = ResponseContractWebSocketTransport
			contractCase.WebSocket = newResponseContractWebSocket(endpoint.Stream, endpoint.Response.Result)
		}
		analysis.Cases = append(analysis.Cases, contractCase)
	}
	for _, response := range endpoint.Response.ErrorResponses {
		contractCase := newResponseContractCase(serviceName, methodName, endpoint.Response.FileResponse, response)
		contractCase.Multipart = multipart
		analysis.Cases = append(analysis.Cases, contractCase)
	}

	seen := make(map[string]struct{}, len(analysis.Cases))
	for _, contractCase := range analysis.Cases {
		if _, ok := seen[contractCase.ID]; ok {
			analysis.Cases = nil
			analysis.Limitations = []ResponseContractLimitation{{
				Code:   ResponseContractDuplicateCaseID,
				Detail: fmt.Sprintf("response contract case ID %q is not unique", contractCase.ID),
			}}
			return analysis
		}
		seen[contractCase.ID] = struct{}{}
	}
	return analysis
}

// Supported reports whether the endpoint is inside the HTTP response contract
// analysis scope.
func (a *ResponseContractAnalysis) Supported() bool {
	return a != nil && len(a.Limitations) == 0
}

func responseContractLimitations(endpoint *Endpoint) []ResponseContractLimitation {
	if endpoint == nil {
		return []ResponseContractLimitation{{
			Code:   ResponseContractMissingEndpoint,
			Detail: "response contract analysis requires an endpoint",
		}}
	}

	var limitations []ResponseContractLimitation
	if endpoint.Service == nil || strings.TrimSpace(endpoint.Service.Name) == "" || strings.TrimSpace(endpoint.MethodName) == "" {
		limitations = append(limitations, ResponseContractLimitation{
			Code:   ResponseContractMissingIdentity,
			Detail: "response contract cases require service and method names",
		})
	}
	if endpoint.IsJSONRPC {
		limitations = append(limitations, ResponseContractLimitation{
			Code:   ResponseContractJSONRPC,
			Detail: "JSON-RPC response envelopes are outside unary HTTP contract scope",
		})
	}
	limitations = append(limitations, responseContractStreamingLimitations(endpoint.Stream)...)
	if endpoint.Redirect != nil {
		limitations = append(limitations, ResponseContractLimitation{
			Code:   ResponseContractRedirect,
			Detail: "redirect endpoints do not execute declared result response branches",
		})
	}
	if endpoint.Request != nil && endpoint.Request.Multipart {
		if endpoint.Stream != nil && endpoint.Stream.IsSSE {
			limitations = append(limitations, *unsupportedMultipartContract(
				"multipart response contracts do not support SSE endpoints",
			))
		} else if endpoint.Stream != nil && endpoint.Stream.IsWebSocket {
			limitations = append(limitations, *unsupportedMultipartContract(
				"multipart response contracts do not support WebSocket endpoints",
			))
		} else if _, limitation := responseContractMultipartRequest(endpoint.Request); limitation != nil {
			limitations = append(limitations, *limitation)
		}
	}
	if endpoint.Request != nil && endpoint.Request.SkipBodyEncode {
		limitations = append(limitations, ResponseContractLimitation{
			Code:   ResponseContractRawRequestBody,
			Detail: "raw request bodies require an application-owned stream fixture",
		})
	}
	if endpoint.Response == nil {
		limitations = append(limitations, ResponseContractLimitation{
			Code:   ResponseContractRawResponseBody,
			Detail: "endpoint response metadata is unavailable",
		})
	} else if endpoint.Response.SkipBodyEncode {
		limitations = append(limitations, ResponseContractLimitation{
			Code:   ResponseContractRawResponseBody,
			Detail: "raw response bodies require an application-owned stream fixture",
		})
	}
	return limitations
}

func responseContractMultipartRequest(request *Request) (*ResponseContractMultipartRequest, *ResponseContractLimitation) {
	if request == nil || !request.Multipart {
		return nil, nil
	}
	if request.Body == nil {
		return nil, unsupportedMultipartContract("multipart response contracts require a request body")
	}
	object := expr.AsObject(request.Body.Type)
	if object == nil || len(*object) == 0 {
		return nil, unsupportedMultipartContract("multipart response contracts require a non-empty object request body")
	}

	contract := &ResponseContractMultipartRequest{
		ContentType: "multipart/form-data",
		Parts:       make([]ResponseContractMultipartPart, 0, len(*object)),
	}
	seen := make(map[string]struct{}, len(*object))
	for _, namedAttribute := range *object {
		if namedAttribute == nil || namedAttribute.Attribute == nil {
			return nil, unsupportedMultipartContract("multipart response contracts require named request parts")
		}
		name := strings.SplitN(namedAttribute.Name, ":", 2)[0]
		if strings.TrimSpace(name) == "" {
			return nil, unsupportedMultipartContract("multipart response contracts require non-empty part names")
		}
		if _, ok := seen[name]; ok {
			return nil, unsupportedMultipartContract(fmt.Sprintf("multipart part name %q is not unique", name))
		}
		seen[name] = struct{}{}

		if !expr.IsPrimitive(namedAttribute.Attribute.Type) || namedAttribute.Attribute.Type.Kind() == expr.AnyKind {
			return nil, unsupportedMultipartContract(fmt.Sprintf(
				"multipart part %q does not have a primitive or bytes shape",
				name,
			))
		}
		mediaType := "text/plain"
		if namedAttribute.Attribute.Type.Kind() == expr.BytesKind {
			mediaType = "application/octet-stream"
		}
		contract.Parts = append(contract.Parts, ResponseContractMultipartPart{
			Name:      name,
			MediaType: mediaType,
			Required:  request.Body.IsRequired(namedAttribute.Name),
		})
	}
	return contract, nil
}

func unsupportedMultipartContract(detail string) *ResponseContractLimitation {
	return &ResponseContractLimitation{
		Code:   ResponseContractMultipart,
		Detail: detail,
	}
}

func responseContractStreamingLimitations(stream *Stream) []ResponseContractLimitation {
	if stream == nil || !stream.IsStreaming {
		return nil
	}
	if !stream.IsSSE {
		return responseContractWebSocketLimitations(stream)
	}
	var limitations []ResponseContractLimitation
	if stream.HasMixedResults {
		limitations = append(limitations, ResponseContractLimitation{
			Code:   ResponseContractStreaming,
			Detail: "mixed unary and SSE results require separate negotiated contract cases",
		})
	}
	if stream.Direction != "server" {
		limitations = append(limitations, ResponseContractLimitation{
			Code:   ResponseContractStreaming,
			Detail: "SSE response contracts currently require a server stream",
		})
	}
	if stream.ResponseMessage == nil || stream.ResponseMessage.Type == nil || stream.ResponseMessage.Type == expr.Empty {
		limitations = append(limitations, ResponseContractLimitation{
			Code:   ResponseContractStreaming,
			Detail: "SSE response contracts require a streaming result message",
		})
	}
	return limitations
}

func responseContractWebSocketLimitations(stream *Stream) []ResponseContractLimitation {
	limitation := func(detail string) []ResponseContractLimitation {
		return []ResponseContractLimitation{{
			Code:   ResponseContractStreaming,
			Detail: detail,
		}}
	}
	if stream.HasMixedResults {
		return limitation("mixed unary and WebSocket results require separate negotiated contract cases")
	}
	if stream.HandshakeStatus != expr.StatusSwitchingProtocols {
		return limitation("WebSocket response contracts require a 101 switching-protocols handshake")
	}
	switch stream.Direction {
	case "server":
		if responseContractMessageType(stream.ResponseMessage) == "" {
			return limitation("server WebSocket response contracts require an outbound message")
		}
	case "client":
		if responseContractMessageType(stream.RequestMessage) == "" {
			return limitation("client WebSocket response contracts require an inbound message")
		}
	case "bidirectional":
		if responseContractMessageType(stream.RequestMessage) == "" || responseContractMessageType(stream.ResponseMessage) == "" {
			return limitation("bidirectional WebSocket response contracts require inbound and outbound messages")
		}
	default:
		return limitation(fmt.Sprintf("WebSocket response contracts do not support stream direction %q", stream.Direction))
	}
	return nil
}

func newResponseContractCase(serviceName, methodName string, fileResponse bool, response *ResponseStatus) *ResponseContractCase {
	// IsError == (Error != nil) is guaranteed at construction (see
	// buildResponseStatus in request_response.go), so Error is dereferenced
	// unguarded here, matching responseContractCaseID below.
	kind := ResponseContractSuccess
	errorName := ""
	if response.IsError {
		kind = ResponseContractError
		errorName = response.Error.Name
	}
	return &ResponseContractCase{
		ID:           responseContractCaseID(serviceName, methodName, kind, response),
		Kind:         kind,
		Transport:    ResponseContractHTTP,
		StatusCode:   response.StatusCode,
		ErrorName:    errorName,
		TagName:      response.TagName,
		TagValue:     response.TagValue,
		ContentTypes: append([]string(nil), response.ContentTypes...),
		HasBody:      (fileResponse && !response.IsError) || responseHasBody(response),
		Headers:      responseContractHeaders(response.Headers),
		Cookies:      responseContractCookies(response.Cookies),
	}
}

func newResponseContractSSE(stream *Stream) *ResponseContractSSE {
	contract := &ResponseContractSSE{
		Direction: "server",
		Terminal:  "eof",
	}
	if stream == nil {
		return contract
	}
	contract.Direction = stream.Direction
	if stream.ResponseMessage != nil && stream.ResponseMessage.Type != nil {
		contract.MessageType = stream.ResponseMessage.Type.Name()
	}
	if stream.SSE == nil {
		return contract
	}
	contract.DataField = stream.SSE.DataField
	contract.DataEncoding = responseContractSSEDataEncoding(stream)
	contract.IDField = stream.SSE.IDField
	contract.EventField = stream.SSE.EventField
	contract.RetryField = stream.SSE.RetryField
	if stream.ResponseMessage != nil {
		contract.IDRequired = stream.SSE.IDField != "" && stream.ResponseMessage.IsRequired(stream.SSE.IDField)
		contract.EventTypeRequired = stream.SSE.EventField != "" && stream.ResponseMessage.IsRequired(stream.SSE.EventField)
	}
	for _, projection := range stream.SSE.Projections {
		if projection != nil {
			contract.EventTypes = append(contract.EventTypes, projection.EventType)
		}
	}
	return contract
}

func newResponseContractWebSocket(stream *Stream, result *expr.AttributeExpr) *ResponseContractWebSocket {
	contract := &ResponseContractWebSocket{
		HandshakeHeaders: []string{"Connection", "Sec-WebSocket-Accept", "Upgrade"},
		Terminal:         "normal_close",
	}
	if stream == nil {
		return contract
	}
	contract.Direction = stream.Direction
	contract.InboundMessageType = responseContractMessageType(stream.RequestMessage)
	contract.OutboundMessageType = responseContractMessageType(stream.ResponseMessage)
	if stream.Direction == "client" && contract.OutboundMessageType == "" {
		contract.OutboundMessageType = responseContractMessageType(result)
	}
	if stream.Direction == "client" && contract.OutboundMessageType != "" {
		contract.Terminal = "final_message"
	}
	return contract
}

func responseContractMessageType(message *expr.AttributeExpr) string {
	if message == nil || message.Type == nil || message.Type == expr.Empty {
		return ""
	}
	return message.Type.Name()
}

func responseContractSSEDataEncoding(stream *Stream) string {
	if stream == nil || stream.ResponseMessage == nil || stream.ResponseMessage.Type == nil {
		return "json"
	}
	data := stream.ResponseMessage
	if stream.SSE != nil && stream.SSE.DataField != "" {
		if field := stream.ResponseMessage.Find(stream.SSE.DataField); field != nil {
			data = field
		}
	}
	if data.Type == expr.String || data.Type == expr.Bytes {
		return "text"
	}
	return "json"
}

func responseHasBody(response *ResponseStatus) bool {
	return response != nil && response.Body != nil && response.Body.Type != expr.Empty
}

func responseContractCaseID(serviceName, methodName string, kind ResponseContractCaseKind, response *ResponseStatus) string {
	parts := []string{
		responseContractIDSegment(serviceName),
		responseContractIDSegment(methodName),
		string(kind),
	}
	if kind == ResponseContractError {
		parts = append(parts, responseContractIDSegment(response.Error.Name))
	}
	parts = append(parts, fmt.Sprintf("%d", response.StatusCode))
	if kind == ResponseContractSuccess && response.TagName != "" {
		parts = append(parts, responseContractIDSegment(response.TagName)+"="+responseContractIDSegment(response.TagValue))
	}
	return strings.Join(parts, ".")
}

func responseContractIDSegment(value string) string {
	escaped := url.PathEscape(value)
	return strings.NewReplacer(".", "%2E", "=", "%3D").Replace(escaped)
}

func responseContractHeaders(headers []*Header) []ResponseContractHeader {
	result := make([]ResponseContractHeader, 0, len(headers))
	for _, header := range headers {
		if header == nil {
			continue
		}
		result = append(result, ResponseContractHeader{
			Name:     header.Name,
			HTTPName: header.HTTPName,
			Required: header.Required,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].HTTPName == result[j].HTTPName {
			return result[i].Name < result[j].Name
		}
		return result[i].HTTPName < result[j].HTTPName
	})
	return result
}

func responseContractCookies(cookies []*Cookie) []ResponseContractCookie {
	result := make([]ResponseContractCookie, 0, len(cookies))
	for _, cookie := range cookies {
		if cookie == nil {
			continue
		}
		result = append(result, ResponseContractCookie{
			Name:     cookie.Name,
			HTTPName: cookie.HTTPName,
			Required: cookie.Required,
			Path:     cookie.Path,
			Domain:   cookie.Domain,
			MaxAge:   cookie.MaxAge,
			Secure:   cookie.Secure,
			HTTPOnly: cookie.HTTPOnly,
			SameSite: cookie.SameSite,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].HTTPName == result[j].HTTPName {
			return result[i].Name < result[j].Name
		}
		return result[i].HTTPName < result[j].HTTPName
	})
	return result
}
