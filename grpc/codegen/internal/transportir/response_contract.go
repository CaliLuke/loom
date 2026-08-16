package transportir

import (
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/CaliLuke/loom/expr"
)

type (
	// ResponseContractCaseKind identifies a successful result or service error.
	ResponseContractCaseKind string

	// ResponseContractLimitationCode identifies an unsupported endpoint shape.
	ResponseContractLimitationCode string

	// ResponseContractAnalysis contains supported cases or explicit limitations.
	ResponseContractAnalysis struct {
		// Cases lists response branches in design order.
		Cases []*ResponseContractCase
		// Limitations explains why cases could not be produced.
		Limitations []ResponseContractLimitation
	}

	// ResponseContractLimitation explains an unsupported endpoint shape.
	ResponseContractLimitation struct {
		// Code identifies the unsupported feature.
		Code ResponseContractLimitationCode
		// Detail explains the limitation.
		Detail string
	}

	// ResponseContractCase describes one declared gRPC response branch.
	ResponseContractCase struct {
		// ID is stable while the service contract is unchanged.
		ID string
		// Kind identifies a successful result or service error.
		Kind ResponseContractCaseKind
		// StatusCode is the declared gRPC status code.
		StatusCode int
		// MessageType is the protobuf response message name.
		MessageType string
		// ErrorName is the declared service error name.
		ErrorName string
		// DetailType is the protobuf status detail message name.
		DetailType string
		// RequiredHeaders lists required response metadata keys.
		RequiredHeaders []string
		// RequiredTrailers lists required trailer metadata keys.
		RequiredTrailers []string
		// Stream describes a supported streaming completion.
		Stream *ResponseContractStream
	}

	// ResponseContractStream describes a selected gRPC stream completion.
	ResponseContractStream struct {
		// Direction is the designed stream direction.
		Direction string
		// Terminal identifies the expected completion behavior.
		Terminal string
	}
)

const (
	// ResponseContractSuccess identifies a successful response.
	ResponseContractSuccess ResponseContractCaseKind = "success"
	// ResponseContractError identifies a service error.
	ResponseContractError ResponseContractCaseKind = "error"

	// ResponseContractMissingEndpoint indicates that no endpoint was supplied.
	ResponseContractMissingEndpoint ResponseContractLimitationCode = "missing_endpoint"
	// ResponseContractMissingIdentity indicates that service or method identity is unavailable.
	ResponseContractMissingIdentity ResponseContractLimitationCode = "missing_identity"
	// ResponseContractMissingResponse indicates that response metadata is unavailable.
	ResponseContractMissingResponse ResponseContractLimitationCode = "missing_response"
	// ResponseContractStreaming indicates an unsupported stream lifecycle.
	ResponseContractStreaming ResponseContractLimitationCode = "streaming"
)

// AnalyzeResponseContractCases builds deterministic gRPC response cases.
func AnalyzeResponseContractCases(endpoint *Endpoint) *ResponseContractAnalysis {
	analysis := &ResponseContractAnalysis{Limitations: responseContractLimitations(endpoint)}
	if len(analysis.Limitations) > 0 {
		return analysis
	}
	stream := responseContractStream(endpoint)
	analysis.Cases = append(analysis.Cases, &ResponseContractCase{
		ID:               responseContractCaseID(endpoint.Service.Name, endpoint.Name, ResponseContractSuccess, "", endpoint.Response.StatusCode),
		Kind:             ResponseContractSuccess,
		StatusCode:       endpoint.Response.StatusCode,
		MessageType:      responseContractMessageType(endpoint.Response.ProtoMessage),
		RequiredHeaders:  responseContractRequiredMetadata(endpoint.Response.Headers),
		RequiredTrailers: responseContractRequiredMetadata(endpoint.Response.Trailers),
		Stream:           stream,
	})
	for _, endpointError := range endpoint.Errors {
		if endpointError == nil || endpointError.Response == nil {
			continue
		}
		detailType := "loompb.ErrorResponse"
		if endpointError.Type != expr.ErrorResult && expr.IsObject(endpointError.Type) {
			detailType = responseContractMessageType(endpointError.Response.ProtoMessage)
		}
		analysis.Cases = append(analysis.Cases, &ResponseContractCase{
			ID:               responseContractCaseID(endpoint.Service.Name, endpoint.Name, ResponseContractError, endpointError.Name, endpointError.Response.StatusCode),
			Kind:             ResponseContractError,
			StatusCode:       endpointError.Response.StatusCode,
			ErrorName:        endpointError.Name,
			DetailType:       detailType,
			RequiredHeaders:  responseContractRequiredMetadata(endpointError.Response.Headers),
			RequiredTrailers: responseContractRequiredMetadata(endpointError.Response.Trailers),
			Stream:           stream,
		})
	}
	return analysis
}

// Supported reports whether cases were produced without limitations.
func (a *ResponseContractAnalysis) Supported() bool {
	return a != nil && len(a.Limitations) == 0
}

func responseContractLimitations(endpoint *Endpoint) []ResponseContractLimitation {
	if endpoint == nil {
		return []ResponseContractLimitation{{Code: ResponseContractMissingEndpoint, Detail: "gRPC response contract analysis requires an endpoint"}}
	}
	var limitations []ResponseContractLimitation
	if endpoint.Service == nil || strings.TrimSpace(endpoint.Service.Name) == "" || strings.TrimSpace(endpoint.Name) == "" {
		limitations = append(limitations, ResponseContractLimitation{Code: ResponseContractMissingIdentity, Detail: "gRPC response contracts require service and method names"})
	}
	if endpoint.Response == nil {
		limitations = append(limitations, ResponseContractLimitation{Code: ResponseContractMissingResponse, Detail: "gRPC response metadata is unavailable"})
	}
	if endpoint.Method != nil && (endpoint.Method.Stream == expr.ClientStreamKind || endpoint.Method.Stream == expr.BidirectionalStreamKind) {
		limitations = append(limitations, ResponseContractLimitation{Code: ResponseContractStreaming, Detail: "gRPC response contracts currently support unary and server-streaming endpoints"})
	}
	return limitations
}

func responseContractStream(endpoint *Endpoint) *ResponseContractStream {
	if endpoint == nil || endpoint.Method == nil || endpoint.Method.Stream != expr.ServerStreamKind {
		return nil
	}
	return &ResponseContractStream{Direction: "server", Terminal: "eof"}
}

func responseContractMessageType(message *expr.AttributeExpr) string {
	if message == nil {
		return ""
	}
	if names := message.Meta["struct:name:proto"]; len(names) > 0 {
		return names[0]
	}
	if message.Type == nil || message.Type == expr.Empty {
		return ""
	}
	return message.Type.Name()
}

func responseContractRequiredMetadata(mapped *expr.MappedAttributeExpr) []string {
	if mapped == nil {
		return nil
	}
	var names []string
	_ = expr.WalkMappedAttr(mapped, func(name, element string, attribute *expr.AttributeExpr) error {
		if mapped.IsRequired(name) {
			names = append(names, strings.ToLower(element))
		}
		return nil
	})
	sort.Strings(names)
	return names
}

func responseContractCaseID(service, method string, kind ResponseContractCaseKind, errorName string, statusCode int) string {
	parts := []string{responseContractIDSegment(service), responseContractIDSegment(method), string(kind)}
	if errorName != "" {
		parts = append(parts, responseContractIDSegment(errorName))
	}
	return strings.Join(append(parts, fmt.Sprintf("%d", statusCode)), ".")
}

func responseContractIDSegment(value string) string {
	escaped := url.PathEscape(value)
	return strings.NewReplacer(".", "%2E", "=", "%3D").Replace(escaped)
}
