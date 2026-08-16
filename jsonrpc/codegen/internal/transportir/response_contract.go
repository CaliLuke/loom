package transportir

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/CaliLuke/loom/expr"
)

type (
	// ResponseContractCaseKind identifies a JSON-RPC response branch.
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

	// ResponseContractCase describes one declared JSON-RPC response branch.
	ResponseContractCase struct {
		// ID is stable while the service contract is unchanged.
		ID string
		// Kind identifies a success, service error, or notification.
		Kind ResponseContractCaseKind
		// ResultType is the designed success result type.
		ResultType string
		// HasResult reports whether a success envelope contains a result member.
		HasResult bool
		// ErrorCode is the declared JSON-RPC error code.
		ErrorCode int
		// ErrorName is the declared service error name.
		ErrorName string
		// ErrorDataType is the designed JSON error-data type.
		ErrorDataType string
		// Stream describes a supported streaming terminal contract.
		Stream *ResponseContractStream
	}

	// ResponseContractStream describes a selected JSON-RPC stream terminal.
	ResponseContractStream struct {
		// Transport identifies the streaming wire protocol.
		Transport string
		// Terminal identifies the expected terminal behavior.
		Terminal string
	}
)

const (
	// ResponseContractSuccess identifies a successful response.
	ResponseContractSuccess ResponseContractCaseKind = "success"
	// ResponseContractError identifies a service error.
	ResponseContractError ResponseContractCaseKind = "error"
	// ResponseContractNotification identifies response suppression for an ID-less request.
	ResponseContractNotification ResponseContractCaseKind = "notification"

	// ResponseContractMissingEndpoint indicates that no endpoint was supplied.
	ResponseContractMissingEndpoint ResponseContractLimitationCode = "missing_endpoint"
	// ResponseContractMissingIdentity indicates that service or method identity is unavailable.
	ResponseContractMissingIdentity ResponseContractLimitationCode = "missing_identity"
	// ResponseContractNotJSONRPC indicates that the endpoint is not JSON-RPC.
	ResponseContractNotJSONRPC ResponseContractLimitationCode = "not_jsonrpc"
	// ResponseContractStreaming indicates an unsupported stream lifecycle.
	ResponseContractStreaming ResponseContractLimitationCode = "streaming"
)

// AnalyzeResponseContractCases builds deterministic JSON-RPC response cases.
func AnalyzeResponseContractCases(endpoint *expr.HTTPEndpointExpr) *ResponseContractAnalysis {
	analysis := &ResponseContractAnalysis{Limitations: responseContractLimitations(endpoint)}
	if len(analysis.Limitations) > 0 {
		return analysis
	}
	serviceName := endpoint.Service.Name()
	methodName := endpoint.Name()
	successStream := responseContractStream(endpoint, "final_response")
	notificationStream := responseContractStream(endpoint, "suppressed")
	resultType := responseContractTypeName(endpoint.MethodExpr.Result)
	analysis.Cases = append(analysis.Cases, &ResponseContractCase{
		ID:         responseContractCaseID(serviceName, methodName, ResponseContractSuccess, "", 0),
		Kind:       ResponseContractSuccess,
		ResultType: resultType,
		HasResult:  resultType != "",
		Stream:     successStream,
	})
	for _, endpointError := range endpoint.HTTPErrors {
		if endpointError == nil || endpointError.Response == nil || endpointError.ErrorExpr == nil {
			continue
		}
		dataType := "jsonrpc.ErrorData"
		if endpointError.Type != expr.ErrorResult {
			dataType = responseContractTypeName(endpointError.AttributeExpr)
		}
		analysis.Cases = append(analysis.Cases, &ResponseContractCase{
			ID:            responseContractCaseID(serviceName, methodName, ResponseContractError, endpointError.Name, endpointError.Response.StatusCode),
			Kind:          ResponseContractError,
			ErrorCode:     endpointError.Response.StatusCode,
			ErrorName:     endpointError.Name,
			ErrorDataType: dataType,
			Stream:        successStream,
		})
	}
	analysis.Cases = append(analysis.Cases, &ResponseContractCase{
		ID:     responseContractCaseID(serviceName, methodName, ResponseContractNotification, "", 0),
		Kind:   ResponseContractNotification,
		Stream: notificationStream,
	})
	return analysis
}

// Supported reports whether cases were produced without limitations.
func (a *ResponseContractAnalysis) Supported() bool {
	return a != nil && len(a.Limitations) == 0
}

func responseContractLimitations(endpoint *expr.HTTPEndpointExpr) []ResponseContractLimitation {
	if endpoint == nil {
		return []ResponseContractLimitation{{Code: ResponseContractMissingEndpoint, Detail: "JSON-RPC response contract analysis requires an endpoint"}}
	}
	var limitations []ResponseContractLimitation
	if endpoint.Service == nil || strings.TrimSpace(endpoint.Service.Name()) == "" || strings.TrimSpace(endpoint.Name()) == "" {
		limitations = append(limitations, ResponseContractLimitation{Code: ResponseContractMissingIdentity, Detail: "JSON-RPC response contracts require service and method names"})
	}
	if !endpoint.IsJSONRPC() {
		limitations = append(limitations, ResponseContractLimitation{Code: ResponseContractNotJSONRPC, Detail: "endpoint does not use JSON-RPC"})
	}
	if endpoint.MethodExpr != nil && endpoint.MethodExpr.IsStreaming() && (endpoint.MethodExpr.Stream != expr.ServerStreamKind || endpoint.SSE == nil) {
		limitations = append(limitations, ResponseContractLimitation{Code: ResponseContractStreaming, Detail: "JSON-RPC response contracts currently support unary and server-SSE endpoints"})
	}
	return limitations
}

func responseContractStream(endpoint *expr.HTTPEndpointExpr, terminal string) *ResponseContractStream {
	if endpoint == nil || endpoint.MethodExpr == nil || endpoint.MethodExpr.Stream != expr.ServerStreamKind || endpoint.SSE == nil {
		return nil
	}
	return &ResponseContractStream{Transport: "sse", Terminal: terminal}
}

func responseContractTypeName(attribute *expr.AttributeExpr) string {
	if attribute == nil || attribute.Type == nil || attribute.Type == expr.Empty {
		return ""
	}
	return attribute.Type.Name()
}

func responseContractCaseID(service, method string, kind ResponseContractCaseKind, errorName string, code int) string {
	parts := []string{responseContractIDSegment(service), responseContractIDSegment(method), string(kind)}
	if errorName != "" {
		parts = append(parts, responseContractIDSegment(errorName), fmt.Sprintf("%d", code))
	}
	return strings.Join(parts, ".")
}

func responseContractIDSegment(value string) string {
	escaped := url.PathEscape(value)
	return strings.NewReplacer(".", "%2E", "=", "%3D").Replace(escaped)
}
