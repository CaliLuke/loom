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

	// ResponseContractLimitationCode identifies an endpoint feature that the
	// unary HTTP response contract analysis intentionally does not support.
	ResponseContractLimitationCode string

	// ResponseContractAnalysis describes the exhaustive response cases for a
	// supported unary HTTP endpoint. Cases is empty when Limitations is not.
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
		// Headers lists the declared response header assertions.
		Headers []ResponseContractHeader
		// Cookies lists the declared response cookie assertions.
		Cookies []ResponseContractCookie
		// Response is the source transport response used by later generators to
		// construct the result or typed error for this case.
		Response *ResponseStatus
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

	// ResponseContractMissingEndpoint indicates that no endpoint was supplied.
	ResponseContractMissingEndpoint ResponseContractLimitationCode = "missing_endpoint"
	// ResponseContractMissingIdentity indicates that a stable service or method
	// identity is unavailable.
	ResponseContractMissingIdentity ResponseContractLimitationCode = "missing_identity"
	// ResponseContractJSONRPC indicates that the endpoint uses JSON-RPC rather
	// than plain HTTP semantics.
	ResponseContractJSONRPC ResponseContractLimitationCode = "jsonrpc"
	// ResponseContractStreaming indicates that the endpoint uses SSE or
	// WebSocket streaming.
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
// for the declared response branches of a supported unary HTTP endpoint.
func AnalyzeResponseContractCases(endpoint *Endpoint) *ResponseContractAnalysis {
	analysis := &ResponseContractAnalysis{}
	analysis.Limitations = responseContractLimitations(endpoint)
	if len(analysis.Limitations) > 0 {
		return analysis
	}

	serviceName := endpoint.Service.Name
	methodName := endpoint.MethodName
	analysis.Cases = make([]*ResponseContractCase, 0, len(endpoint.Response.Responses)+len(endpoint.Response.ErrorResponses))
	for _, response := range endpoint.Response.Responses {
		analysis.Cases = append(analysis.Cases, newResponseContractCase(serviceName, methodName, response))
	}
	for _, response := range endpoint.Response.ErrorResponses {
		analysis.Cases = append(analysis.Cases, newResponseContractCase(serviceName, methodName, response))
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

// Supported reports whether the endpoint is inside the unary HTTP response
// contract analysis scope.
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
	if endpoint.Stream != nil && endpoint.Stream.IsStreaming {
		limitations = append(limitations, ResponseContractLimitation{
			Code:   ResponseContractStreaming,
			Detail: "SSE and WebSocket responses require stream-aware contract scenarios",
		})
	}
	if endpoint.Redirect != nil {
		limitations = append(limitations, ResponseContractLimitation{
			Code:   ResponseContractRedirect,
			Detail: "redirect endpoints do not execute declared result response branches",
		})
	}
	if endpoint.Request != nil && endpoint.Request.Multipart {
		limitations = append(limitations, ResponseContractLimitation{
			Code:   ResponseContractMultipart,
			Detail: "multipart requests may require application-owned codecs and fixtures",
		})
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

func newResponseContractCase(serviceName, methodName string, response *ResponseStatus) *ResponseContractCase {
	kind := ResponseContractSuccess
	errorName := ""
	if response.IsError {
		kind = ResponseContractError
		if response.Error != nil {
			errorName = response.Error.Name
		}
	}
	return &ResponseContractCase{
		ID:           responseContractCaseID(serviceName, methodName, kind, response),
		Kind:         kind,
		StatusCode:   response.StatusCode,
		ErrorName:    errorName,
		TagName:      response.TagName,
		TagValue:     response.TagValue,
		ContentTypes: append([]string(nil), response.ContentTypes...),
		Headers:      responseContractHeaders(response.Headers),
		Cookies:      responseContractCookies(response.Cookies),
		Response:     response,
	}
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
