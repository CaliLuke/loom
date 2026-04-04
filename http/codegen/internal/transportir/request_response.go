package transportir

import (
	"fmt"
	"net/textproto"
	"strings"

	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
)

func buildRequest(endpoint *expr.HTTPEndpointExpr) *Request {
	if endpoint == nil {
		return nil
	}
	payload := endpoint.MethodExpr.Payload
	body := normalizeHTTPAttribute(endpoint.Body)
	streamingBody := endpoint.StreamingBody
	bodyOrigin := attributeOrigin(body)
	mustHaveBody := body != nil && body.Type != expr.Empty
	if endpoint.OptionalRequestBody {
		mustHaveBody = false
	}
	if bodyOrigin != "" && payload != nil && !payload.IsRequired(bodyOrigin) {
		mustHaveBody = false
	}
	return &Request{
		Payload:             payload,
		Body:                body,
		RawBody:             endpoint.Body,
		StreamingBody:       streamingBody,
		BodyOrigin:          bodyOrigin,
		PathParams:          buildPathParameters(endpoint),
		QueryParams:         buildQueryParameters(endpoint),
		Headers:             buildHeaderParameters(endpoint),
		Cookies:             buildCookieParameters(endpoint),
		MapQueryParams:      endpoint.MapQueryParams,
		Multipart:           endpoint.MultipartRequest,
		FormEncoded:         endpoint.FormRequest,
		OptionalBody:        endpoint.OptionalRequestBody,
		MustHaveBody:        mustHaveBody,
		SkipBodyEncode:      endpoint.SkipRequestBodyEncodeDecode,
		IDAttribute:         endpoint.PayloadIDAttribute,
		IDAttributeRequired: payload != nil && endpoint.PayloadIDAttribute != "" && payload.IsRequired(endpoint.PayloadIDAttribute),
	}
}

func buildResponse(endpoint *expr.HTTPEndpointExpr) *Response {
	if endpoint == nil {
		return nil
	}
	result := endpoint.MethodExpr.Result
	response := &Response{
		Result:              result,
		StreamingResult:     endpoint.MethodExpr.StreamingResult,
		HasMixedResults:     endpoint.MethodExpr.HasMixedResults(),
		SkipBodyEncode:      endpoint.SkipResponseBodyEncodeDecode,
		IDAttribute:         endpoint.ResultIDAttribute,
		IDAttributeRequired: result != nil && endpoint.ResultIDAttribute != "" && result.IsRequired(endpoint.ResultIDAttribute),
	}
	for _, status := range endpoint.Responses {
		response.Responses = append(response.Responses, buildResponseStatus(status, nil))
	}
	for _, httpError := range endpoint.HTTPErrors {
		response.ErrorResponses = append(response.ErrorResponses, buildResponseStatus(httpError.Response, httpError))
	}
	return response
}

func buildRedirect(endpoint *expr.HTTPEndpointExpr) *Redirect {
	if endpoint == nil || endpoint.Redirect == nil {
		return nil
	}
	return &Redirect{
		URL:        endpoint.Redirect.URL,
		StatusCode: endpoint.Redirect.StatusCode,
	}
}

func buildResponseStatus(status *expr.HTTPResponseExpr, httpErrorExpr *expr.HTTPErrorExpr) *ResponseStatus {
	if status == nil {
		return nil
	}
	documentBody := responseDocumentBody(status)
	httpError := buildError(httpErrorExpr)
	return &ResponseStatus{
		Error:        httpError,
		StatusCode:   status.StatusCode,
		Description:  status.Description,
		ContentType:  status.ContentType,
		ContentTypes: responseContentTypes(status),
		Headers:      buildHeaders(status.Headers),
		Cookies:      buildCookies(status.Cookies),
		Body:         normalizeHTTPAttribute(status.Body),
		DocumentBody: documentBody,
		BodyOrigin:   attributeOrigin(status.Body),
		TagName:      status.Tag[0],
		TagValue:     status.Tag[1],
		IsError:      httpError != nil,
		EmitExamples: shouldEmitResponseExamples(status),
		IsWebSocket:  isWebSocketResponse(status, status.StatusCode),
		BinaryBody:   status.StatusCode != expr.StatusNoContent && isSkipResponseBodyEncodeDecode(status.Parent),
		Meta:         status.Meta,
		Links:        buildResponseLinks(status),
	}
}

func attributeOrigin(attr *expr.AttributeExpr) string {
	if attr == nil {
		return ""
	}
	if origin, ok := attr.Meta["origin:attribute"]; ok && len(origin) > 0 {
		return origin[0]
	}
	return ""
}

func responseDocumentBody(resp *expr.HTTPResponseExpr) *expr.AttributeExpr {
	if resp == nil {
		return &expr.AttributeExpr{Type: expr.Empty}
	}
	body := resp.Body
	if resp.OpenAPIBody != nil {
		body = resp.OpenAPIBody
	}
	body = normalizeHTTPAttribute(body)
	if body == nil {
		return &expr.AttributeExpr{Type: expr.Empty}
	}
	view, ok := body.Meta.Last(expr.ViewMetaKey)
	if !ok || view == "" {
		return body
	}
	rt, ok := body.Type.(*expr.ResultTypeExpr)
	if !ok {
		return body
	}
	projected, err := expr.Project(expr.Dup(rt).(*expr.ResultTypeExpr), view)
	if err != nil {
		panic(fmt.Sprintf("failed to project %q to view %q", body.Type.Name(), view))
	}
	body.Type = projected
	return body
}

func responseContentTypes(resp *expr.HTTPResponseExpr) []string {
	if contentTypes := responseContentTypeHeaderEnums(resp); len(contentTypes) > 0 {
		return contentTypes
	}
	body := responseDocumentBody(resp)
	contentType := resp.ContentType
	if body != nil {
		if rt, ok := body.Type.(*expr.ResultTypeExpr); ok && contentType == "" {
			contentType = rt.ContentType
		}
	}
	if contentType == "" && isSSEResponse(resp) {
		contentType = "text/event-stream"
	}
	if contentType == "" {
		contentType = "application/json"
	}
	return []string{contentType}
}

func shouldEmitResponseExamples(resp *expr.HTTPResponseExpr) bool {
	if resp == nil {
		return false
	}
	endpoint, ok := resp.Parent.(*expr.HTTPEndpointExpr)
	if !ok || endpoint.MethodExpr == nil {
		return true
	}
	return !endpoint.MethodExpr.IsStreaming()
}

func isSSEResponse(resp *expr.HTTPResponseExpr) bool {
	if resp == nil {
		return false
	}
	endpoint, ok := resp.Parent.(*expr.HTTPEndpointExpr)
	return ok && endpoint.SSE != nil
}

func isWebSocketResponse(resp *expr.HTTPResponseExpr, statusCode int) bool {
	if resp == nil || statusCode != expr.StatusSwitchingProtocols {
		return false
	}
	endpoint, ok := resp.Parent.(*expr.HTTPEndpointExpr)
	return ok && endpoint.MethodExpr != nil && endpoint.MethodExpr.IsStreaming() && endpoint.SSE == nil
}

func responseContentTypeHeaderEnums(resp *expr.HTTPResponseExpr) []string {
	if resp == nil {
		return nil
	}
	var contentTypes []string
	seen := map[string]struct{}{}
	for _, header := range buildHeaders(resp.Headers) {
		if textproto.CanonicalMIMEHeaderKey(header.HTTPName) != "Content-Type" || header.Attribute == nil || header.Attribute.Validation == nil {
			continue
		}
		for _, raw := range header.Attribute.Validation.Values {
			value, ok := raw.(string)
			if !ok {
				continue
			}
			value = strings.TrimSpace(value)
			if value == "" {
				continue
			}
			if _, ok := seen[value]; ok {
				continue
			}
			seen[value] = struct{}{}
			contentTypes = append(contentTypes, value)
		}
	}
	return contentTypes
}

func isSkipResponseBodyEncodeDecode(parent eval.Expression) bool {
	if parent == nil {
		return false
	}
	endpoint, ok := parent.(*expr.HTTPEndpointExpr)
	return ok && endpoint.SkipResponseBodyEncodeDecode
}

func buildResponseLinks(resp *expr.HTTPResponseExpr) []*ResponseLink {
	if resp == nil || len(resp.Links) == 0 {
		return nil
	}
	links := make([]*ResponseLink, 0, len(resp.Links))
	for _, link := range resp.Links {
		if link == nil {
			continue
		}
		links = append(links, &ResponseLink{
			Name:         link.Name,
			Operation:    link.Operation,
			OperationRef: link.OperationRef,
			Description:  link.Description,
			RequestBody:  link.RequestBody,
			Parameters:   cloneLinkParameters(link.Parameters),
		})
	}
	return links
}

func cloneLinkParameters(parameters map[string]string) map[string]string {
	if len(parameters) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(parameters))
	for key, value := range parameters {
		cloned[key] = value
	}
	return cloned
}

func buildError(httpError *expr.HTTPErrorExpr) *Error {
	if httpError == nil || httpError.ErrorExpr == nil {
		return nil
	}
	return &Error{
		Name:      httpError.Name,
		Attribute: httpError.AttributeExpr,
		Type:      httpError.Type,
		Remedy:    buildErrorRemedy(httpError.ErrorExpr.Remedy),
	}
}

func buildErrorRemedy(remedy *expr.ErrorRemedyExpr) *ErrorRemedy {
	if remedy == nil {
		return nil
	}
	return &ErrorRemedy{
		Code:        remedy.Code,
		SafeMessage: remedy.SafeMessage,
		RetryHint:   remedy.RetryHint,
	}
}

func buildHeaders(mapped *expr.MappedAttributeExpr) []*Header {
	if mapped == nil {
		return nil
	}
	headers := make([]*Header, 0)
	expr.WalkMappedAttr(mapped, func(name, element string, attr *expr.AttributeExpr) error { // nolint: errcheck
		headers = append(headers, &Header{
			Name:             name,
			HTTPName:         element,
			Attribute:        attr,
			Required:         mapped.IsRequired(name),
			PrimitivePointer: mapped.IsPrimitivePointer(name, true),
		})
		return nil
	})
	return headers
}

func buildCookies(cookiesExpr []*expr.HTTPResponseCookieExpr) []*Cookie {
	if len(cookiesExpr) == 0 {
		return nil
	}
	cookies := make([]*Cookie, 0, len(cookiesExpr))
	for _, cookieExpr := range cookiesExpr {
		if cookieExpr == nil {
			continue
		}
		name := cookieExpr.AttributeName()
		cookies = append(cookies, &Cookie{
			Name:             name,
			HTTPName:         cookieExpr.HTTPName(),
			Attribute:        cookieExpr.Attribute(),
			Required:         cookieExpr.IsRequired(name),
			PrimitivePointer: cookieExpr.IsPrimitivePointer(name, true),
			Path:             cookieExpr.Path,
			Domain:           cookieExpr.Domain,
			MaxAge:           cookieExpr.MaxAge,
			Secure:           cookieExpr.Secure,
			HTTPOnly:         cookieExpr.HTTPOnly,
			SameSite:         cookieExpr.SameSite,
		})
	}
	return cookies
}

func normalizeHTTPAttribute(attr *expr.AttributeExpr) *expr.AttributeExpr {
	if attr == nil {
		return nil
	}
	cloned := expr.DupAtt(attr)
	return normalizeHTTPAttributeRecursive(cloned, make(map[string]struct{}))
}

func normalizeHTTPAttributeRecursive(attr *expr.AttributeExpr, seen map[string]struct{}) *expr.AttributeExpr {
	switch actual := attr.Type.(type) {
	case expr.UserType:
		if _, ok := actual.(*expr.ResultTypeExpr); !ok && !expr.IsObject(actual) {
			attr.Type = actual.Attribute().Type
			if validation := actual.Attribute().Validation; validation != nil {
				if attr.Validation == nil {
					attr.Validation = validation
				} else {
					attr.Validation.Merge(validation)
				}
			}
			attr.DefaultValue = actual.Attribute().DefaultValue
			attr.UserExamples = actual.Attribute().UserExamples
		}
		if _, ok := seen[actual.ID()]; ok {
			return attr
		}
		seen[actual.ID()] = struct{}{}
		actual.SetAttribute(normalizeHTTPAttributeRecursive(actual.Attribute(), seen))
	case *expr.Array:
		actual.ElemType = normalizeHTTPAttributeRecursive(actual.ElemType, seen)
	case *expr.Map:
		actual.KeyType = normalizeHTTPAttributeRecursive(actual.KeyType, seen)
		actual.ElemType = normalizeHTTPAttributeRecursive(actual.ElemType, seen)
	case *expr.Object:
		object := make(expr.Object, len(*actual))
		for i, named := range *actual {
			object[i] = &expr.NamedAttributeExpr{
				Name:      named.Name,
				Attribute: normalizeHTTPAttributeRecursive(named.Attribute, seen),
			}
		}
		attr.Type = &object
	case *expr.Union:
		// Unions are represented as first-class sum types; HTTP uses the same
		// type for request and response bodies.
	}
	return attr
}
