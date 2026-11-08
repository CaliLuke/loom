package transportir

import (
	"fmt"
	"net/textproto"
	"slices"
	"strings"

	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/httpencoding"
)

func buildRequest(endpoint *expr.HTTPEndpointExpr) *Request {
	if endpoint == nil {
		return nil
	}
	payload := endpoint.MethodExpr.Payload
	body := normalizeHTTPAttribute(endpoint.Body)
	streamingBody := normalizeHTTPAttribute(endpoint.StreamingBody)
	bodyOrigin := attributeOrigin(body)
	var bodyOriginKey string
	if bodyOrigin != "" && payload != nil {
		bodyOriginKey = originKey(payload, bodyOrigin)
	}
	mustHaveBody := body != nil && body.Type != expr.Empty
	if endpoint.OptionalRequestBody {
		mustHaveBody = false
	}
	if bodyOriginKey != "" && !payload.IsRequired(bodyOriginKey) {
		mustHaveBody = false
	}
	var contentType string
	if endpoint.RequestContentType != nil {
		contentType = *endpoint.RequestContentType
	}
	return &Request{
		MediaVariants:        buildMediaVariants(body, endpoint.MediaTypes),
		ContentType:          contentType,
		JSONRPCID:            endpoint.JSONRPCID,
		Payload:              payload,
		Body:                 body,
		DocumentBody:         normalizeHTTPAttribute(endpoint.OpenAPIRequestBody),
		DocumentContentTypes: append([]string(nil), endpoint.OpenAPIRequestBodyContentTypes...),
		DocumentRequired:     endpoint.OpenAPIRequestBodyRequired,
		StreamingBody:        streamingBody,
		BodyOrigin:           bodyOrigin,
		BodyOriginKey:        bodyOriginKey,
		PathParams:           buildPathParameters(endpoint),
		QueryParams:          buildQueryParameters(endpoint),
		Headers:              buildHeaderParameters(endpoint),
		Cookies:              buildCookieParameters(endpoint),
		MapQueryParams:       endpoint.MapQueryParams,
		Multipart:            endpoint.MultipartRequest,
		MultipartEncoding:    multipartBodyEncoding(body, endpoint.MultipartRequest),
		FormEncoded:          endpoint.FormRequest,
		Form:                 formBodyEncoding(body, endpoint.FormRequest),
		OptionalBody:         endpoint.OptionalRequestBody,
		MustHaveBody:         mustHaveBody,
		SkipBodyEncode:       endpoint.SkipRequestBodyEncodeDecode,
	}
}

func buildResponse(endpoint *expr.HTTPEndpointExpr) *Response {
	if endpoint == nil {
		return nil
	}
	result := endpoint.MethodExpr.Result
	response := &Response{
		Result:              result,
		FinalKeys:           endpoint.FinalStatusKeys(),
		InformationalKeys:   endpoint.InformationalStatusKeys(),
		StatusSelected:      endpoint.StatusSelected(),
		NoSuccess:           endpoint.NoSuccess,
		StreamingResult:     endpoint.MethodExpr.StreamingResult,
		HasMixedResults:     endpoint.MethodExpr.HasMixedResults(),
		SkipBodyEncode:      endpoint.SkipResponseBodyEncodeDecode,
		FileResponse:        endpoint.FileResponse,
		IDAttribute:         endpoint.ResultIDAttribute,
		IDAttributeRequired: result != nil && endpoint.ResultIDAttribute != "" && result.IsRequired(endpoint.ResultIDAttribute),
	}
	for _, status := range endpoint.OrderedResponses() {
		responseStatus := buildResponseStatus(status, nil, result)
		if endpoint.FileResponse && status.ContentType == "" && len(responseContentTypeHeaderEnums(status)) == 0 {
			responseStatus.ContentTypes = []string{"*/*"}
		}
		response.Responses = append(response.Responses, responseStatus)
	}
	for _, httpError := range endpoint.HTTPErrors {
		response.ErrorResponses = append(response.ErrorResponses, buildResponseStatus(httpError.Response, httpError, httpError.AttributeExpr))
	}
	for _, info := range endpoint.Informational {
		response.Informational = append(response.Informational, &InformationalResponse{
			Name: info.Name, Response: buildResponseStatus(info.HTTPResponseExpr, nil, info.Value),
		})
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

func buildResponseStatus(status *expr.HTTPResponseExpr, httpErrorExpr *expr.HTTPErrorExpr, carrier *expr.AttributeExpr) *ResponseStatus {
	if status == nil {
		return nil
	}
	if status.JSONRPCError != nil && (httpErrorExpr == nil || status.Status != (expr.HTTPStatusSelector{})) {
		panic("JSON-RPC code requires an error response without an HTTP status")
	}
	if status.Handshake != nil {
		carrier = status.Handshake
	}
	var statusField *StatusBinding
	if status.StatusField != "" && carrier != nil {
		name, attribute := carrier.FindAttribute(status.StatusField)
		statusField = &StatusBinding{Name: name, Attribute: attribute}
	}
	required := map[string][]string{"": status.RequiredFields("")}
	if carrier != nil {
		if result, ok := carrier.Type.(*expr.ResultTypeExpr); ok {
			for _, view := range result.Views {
				required[view.Name] = status.RequiredFields(view.Name)
			}
		}
	}
	documentBody := responseDocumentBody(status)
	body := normalizeHTTPAttribute(status.Body)
	httpError := buildError(httpErrorExpr)
	var unary *ResponseBody
	if status.UnaryBody != nil {
		unaryBody := normalizeHTTPAttribute(status.UnaryBody)
		unary = &ResponseBody{
			Body: unaryBody, ContentType: status.ContentType,
			ContentTypes:  responseContentTypes(status, false),
			MediaVariants: buildMediaVariants(unaryBody, status.MediaTypes),
		}
	}
	return &ResponseStatus{
		Occurrence:              status,
		Unary:                   unary,
		Carrier:                 carrier,
		Handshake:               status.Handshake != nil,
		StatusField:             statusField,
		Required:                required,
		AuthoredBody:            status.AuthoredBody(),
		MediaVariants:           buildMediaVariants(body, status.MediaTypes),
		Error:                   httpError,
		ErrorSelector:           httpErrorExpr.JSONRPCErrorSelector(),
		StatusErrorName:         status.StatusErrorName,
		Selector:                status.Status,
		Domain:                  status.StatusDomain(),
		Description:             status.Description,
		ContentType:             status.ContentType,
		ContentTypes:            responseContentTypes(status, isSSEResponse(status)),
		Headers:                 buildHeaders(status.Headers),
		Cookies:                 buildCookies(status.Cookies),
		Body:                    body,
		DocumentBody:            documentBody,
		IndependentDocumentBody: status.OpenAPIBody != nil,
		BodyOrigin:              attributeOrigin(status.Body),
		Tag:                     status.TagBinding,
		IsError:                 httpError != nil,
		EmitExamples:            shouldEmitResponseExamples(status),
		BinaryBody:              status.Status != (expr.HTTPStatusSelector{Code: expr.StatusNoContent}) && isRawResponseBody(status.Parent),
		Meta:                    status.Meta,
		Links:                   buildResponseLinks(status),
	}
}

// originKey returns the object key of the attribute of parent that a body
// selected with Body(origin) holds, such as "n:m" for the origin "n".
func originKey(parent *expr.AttributeExpr, origin string) string {
	if key, att := parent.FindAttribute(origin); att != nil {
		return key
	}
	return origin
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

func responseContentTypes(resp *expr.HTTPResponseExpr, sse bool) []string {
	if contentTypes := responseContentTypeHeaderEnums(resp); len(contentTypes) > 0 {
		return contentTypes
	}
	contentType := resp.ContentType
	if contentType == "" && sse {
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

func responseContentTypeHeaderEnums(resp *expr.HTTPResponseExpr) []string {
	if resp == nil {
		return nil
	}
	var contentTypes []string
	seen := map[string]struct{}{}
	for _, header := range buildHeaders(resp.Headers) {
		if textproto.CanonicalMIMEHeaderKey(header.HTTPName) != "Content-Type" || header.Attribute == nil {
			continue
		}
		constraints, err := expr.EffectiveConstraintsFor(header.Attribute)
		if err != nil {
			panic(fmt.Sprintf("invalid response header %q constraints: %v", header.Name, err))
		}
		values, _ := constraints.EnumCandidates()
		for _, raw := range values {
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

func isRawResponseBody(parent eval.Expression) bool {
	if parent == nil {
		return false
	}
	endpoint, ok := parent.(*expr.HTTPEndpointExpr)
	return ok && (endpoint.SkipResponseBodyEncodeDecode || endpoint.FileResponse)
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
	var rpcCode *int
	if httpError.Response.JSONRPCError != nil {
		code := httpError.Response.JSONRPCError.Code
		rpcCode = &code
	}
	return &Error{
		JSONRPCCode: rpcCode,
		Name:        httpError.Name,
		Attribute:   httpError.AttributeExpr,
		Type:        httpError.Type,
		Remedy:      buildErrorRemedy(httpError.ErrorExpr.Remedy),
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
		codec, _ := parameterCodec(httpencoding.Header, element, attr, expr.HTTPParameterSerialization(attr, "openapi:"))
		headers = append(headers, &Header{
			Codec:            codec,
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
	// Capture the input contract once before lowering mutates the owned copy.
	// Re-reading constraints from each output node recaptures whole subtrees.
	occurrence, err := expr.NewValueContext().NewOccurrence(cloned)
	if err != nil {
		panic(fmt.Sprintf("invalid HTTP attribute constraints: %v", err))
	}
	return normalizeHTTPAttributeRecursive(cloned, occurrence, make(map[expr.UserType]struct{}))
}

func normalizeHTTPAttributeRecursive(
	attr *expr.AttributeExpr,
	occurrence expr.ValueOccurrence,
	seen map[expr.UserType]struct{},
) *expr.AttributeExpr {
	if occurrence == (expr.ValueOccurrence{}) {
		panic("invalid HTTP normalization: missing captured occurrence")
	}
	constraints := occurrence.EffectiveConstraints()
	defaultValue, hasDefault := constraints.Default()
	switch actual := attr.Type.(type) {
	case expr.UserType:
		if len(attr.UserExamples) == 0 {
			attr.UserExamples = actual.Attribute().ExtractUserExamples()
		}
		_, result := actual.(*expr.ResultTypeExpr)
		flattened := !result && !expr.IsObject(actual) && !hasCanonicalOpenAPITypeName(attr, actual)
		if flattened {
			// Capture the contract owned by the removed edge before flattening.
			attr.Meta = expr.HTTPBodyProjectionMetadata(attr)
			// Preserve inherited nullability before removing the named type.
			attr.Nullable = expr.IsNullable(attr)
			attr.Type = actual.Attribute().Type
			attr.Validation = constraints.Validation().Lowered()
			if attr.Validation.HasRequiredOnly() && len(attr.Validation.Required) == 0 {
				attr.Validation = nil
			}
			if hasDefault {
				attr.DefaultValue = defaultValue
			}
		}
		if _, ok := seen[actual]; !ok {
			seen[actual] = struct{}{}
			actual.SetAttribute(normalizeHTTPAttributeRecursive(actual.Attribute(), occurrence.Underlying(), seen))
		}
		if flattened {
			attr.Type = actual.Attribute().Type
			if _, union := attr.Type.(*expr.Union); !union {
				// A union keeps its declaration as a first-class named sum type.
				captureRemovedEdgeAnnotations(attr, actual.Attribute())
			}
		}
	case *expr.Array:
		actual.ElemType = normalizeHTTPAttributeRecursive(actual.ElemType, occurrence.Element(), seen)
	case *expr.Map:
		actual.KeyType = normalizeHTTPAttributeRecursive(actual.KeyType, occurrence.Key(), seen)
		actual.ElemType = normalizeHTTPAttributeRecursive(actual.ElemType, occurrence.Element(), seen)
	case *expr.Object:
		normalizeHTTPObject(attr, actual, occurrence, seen)
	case *expr.Union:
		// Unions are represented as first-class sum types; HTTP uses the same
		// type for request and response bodies.
	}
	return attr
}

// normalizeHTTPObject normalizes the members of an object attribute, and its
// typed AdditionalAttributes, against the captured occurrence.
func normalizeHTTPObject(attr *expr.AttributeExpr, actual *expr.Object, occurrence expr.ValueOccurrence, seen map[expr.UserType]struct{}) {
	members := occurrence.Members()
	if len(members) != len(*actual) {
		panic("invalid HTTP normalization: captured object shape differs")
	}
	object := make(expr.Object, len(*actual))
	for i, named := range *actual {
		if members[i].Name != named.Name {
			panic("invalid HTTP normalization: captured object member differs")
		}
		object[i] = &expr.NamedAttributeExpr{
			Name:      named.Name,
			Attribute: normalizeHTTPAttributeRecursive(named.Attribute, members[i].Occurrence, seen),
		}
	}
	attr.Type = &object
	if additional, ok := occurrence.Additional(); ok && attr.AdditionalAttributes != nil {
		attr.AdditionalAttributes = normalizeHTTPAttributeRecursive(attr.AdditionalAttributes, additional, seen)
	}
}

// captureRemovedEdgeAnnotations keeps the annotations owned by a flattened
// declaration on the occurrence that replaces it. The declaration was itself
// normalized first, so a chain composes inner to outer and the nearer authored
// value, including an explicit false, wins. Identity, discriminator and media
// encoding metadata stay with their owners; the shared declaration is not
// modified.
func captureRemovedEdgeAnnotations(occurrence, declaration *expr.AttributeExpr) {
	if occurrence.Title == "" {
		occurrence.Title = declaration.Title
	}
	if occurrence.Description == "" {
		occurrence.Description = declaration.Description
	}
	if occurrence.Docs == nil && declaration.Docs != nil {
		docs := *declaration.Docs
		occurrence.Docs = &docs
	}
	for key, values := range declaration.Meta {
		if _, authored := occurrence.Meta[key]; authored || !removedEdgeAnnotationMeta(key) {
			continue
		}
		occurrence.Meta[key] = slices.Clone(values)
	}
}

// removedEdgeAnnotationMeta reports whether a declaration metadata key is an
// OpenAPI annotation that a flattened occurrence inherits.
func removedEdgeAnnotationMeta(key string) bool {
	if !strings.HasPrefix(key, "openapi:") || key == "openapi:typename" || key == "openapi:typename:canonical" {
		return false
	}
	for _, owned := range []string{"openapi:discriminator:", "openapi:encoding:", "openapi:itemEncoding:", "openapi:prefixEncoding:"} {
		if strings.HasPrefix(key, owned) {
			return false
		}
	}
	return true
}

func hasCanonicalOpenAPITypeName(attr *expr.AttributeExpr, userType expr.UserType) bool {
	for _, meta := range []expr.MetaExpr{attr.Meta, userType.Attribute().Meta} {
		value, ok := meta.Last("openapi:typename:canonical")
		if ok && value == "true" {
			return true
		}
	}
	return false
}
