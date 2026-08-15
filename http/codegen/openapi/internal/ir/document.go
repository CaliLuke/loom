package ir

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
	"github.com/CaliLuke/loom/http/codegen/openapi"
)

// BuildDocument analyzes HTTP body/schema-related OpenAPI document data.
func BuildDocument(api *expr.APIExpr, types []expr.UserType, resultTypes []*expr.ResultTypeExpr, options ...AnalyzerOption) *Document {
	if api == nil || api.HTTP == nil {
		return nil
	}
	bodyTypes := BuildBodyTypes(api, types, resultTypes, options...)
	doc := &Document{
		Paths: make(map[string]*PathItem),
		Components: &Components{
			Schemas: bodyTypes.Components,
		},
	}
	closeObjects := openapi.ClosedObjectModeFromExpr(api.Meta)
	for _, svc := range api.HTTP.Services {
		if !openapi.MustGenerate(svc.Meta) || !openapi.MustGenerate(svc.ServiceExpr.Meta) {
			continue
		}
		irService := transportir.BuildService(svc)
		serviceBodies := bodyTypes.Services[svc.Name()]
		for _, endpoint := range irService.Endpoints {
			if !endpoint.Generate || !endpoint.MethodGenerate {
				continue
			}
			for _, route := range endpoint.Routes {
				key := expr.HTTPWildcardRegex.ReplaceAllString(route.Path, "/{$1}")
				operation := buildRouteOperationFromIR(endpoint, route, key, serviceBodies[endpoint.Name], api.ExampleGenerator, api.Meta, closeObjects)
				pathItem := doc.Paths[key]
				if pathItem == nil {
					pathItem = &PathItem{Operations: make(map[string]*Operation)}
					doc.Paths[key] = pathItem
				}
				pathItem.Operations[route.Method] = operation
			}
		}
	}
	componentizeDocument(doc)
	return doc
}

func buildOperation(endpointIR *transportir.Endpoint, bodies *EndpointBodies, rand *expr.ExampleGenerator, closeObjects bool) *Operation {
	if endpointIR == nil {
		return nil
	}
	return &Operation{
		RequestBody: wrapRequestBody(buildRequestBody(endpointIR, bodies, rand, closeObjects)),
		Responses:   wrapResponses(buildResponses(endpointIR, bodies, rand, closeObjects)),
	}
}

func buildRequestBody(endpointIR *transportir.Endpoint, bodies *EndpointBodies, rand *expr.ExampleGenerator, closeObjects bool) *RequestBody {
	if endpointIR == nil || endpointIR.Request == nil {
		return nil
	}
	body := endpointIR.Request.Body
	contentTypes := []string{"application/json"}
	required := endpointIR.Request.MustHaveBody
	if endpointIR.Request.DocumentBody != nil {
		body = endpointIR.Request.DocumentBody
		contentTypes = endpointIR.Request.DocumentContentTypes
		required = endpointIR.Request.DocumentRequired
	}
	if body == nil || body.Type == expr.Empty {
		return nil
	}
	bodyAttr := attributeForSchemaUsage(body, schemaUsageRequest)
	if endpointIR.Request.Multipart {
		contentTypes = []string{"multipart/form-data"}
	} else if endpointIR.Request.FormEncoded {
		contentTypes = []string{"application/x-www-form-urlencoded"}
	}
	mediaType := buildMediaType(bodyAttr, bodies.RequestBody, rand, closeObjects)
	content := make(map[string]*MediaType, len(contentTypes))
	for _, contentType := range contentTypes {
		content[contentType] = mediaType
	}
	return &RequestBody{
		Description:   requestBodyDescription(bodyAttr),
		Required:      required,
		ComponentName: componentMetaValue(bodyAttr, "openapi:component:requestBody"),
		Content:       content,
		Extensions: openapi.MergeExtensions(
			openapi.ExtensionsFromExpr(bodyAttr.Meta),
			openapi.ScopedExtensionsFromExpr(bodyAttr.Meta, "requestBody"),
		),
	}
}

func requestBodyDescription(bodyAttr *expr.AttributeExpr) string {
	if bodyAttr == nil {
		return ""
	}
	if desc := componentMetaValue(bodyAttr, "openapi:description:requestBody"); desc != "" {
		return strings.TrimSpace(desc)
	}
	if desc := strings.TrimSpace(bodyAttr.Description); desc != "" {
		return desc
	}
	return ""
}

func buildResponses(endpointIR *transportir.Endpoint, bodies *EndpointBodies, rand *expr.ExampleGenerator, closeObjects bool) map[string]*Response {
	responses := make(map[string]*Response, len(endpointIR.Response.Responses)+len(endpointIR.Response.ErrorResponses))
	statusBodies := cloneResponseBodies(bodies)
	for _, resp := range endpointIR.Response.Responses {
		statusCode := resp.StatusCode
		if endpointIR.Stream.IsStreaming && !endpointIR.Stream.IsSSE {
			if _, ok := responses[strconv.Itoa(expr.StatusSwitchingProtocols)]; !ok {
				statusBodies[expr.StatusSwitchingProtocols] = statusBodies[resp.StatusCode]
				delete(statusBodies, resp.StatusCode)
				statusCode = expr.StatusSwitchingProtocols
			}
		}
		websocketHandshake := endpointIR.Stream.IsStreaming && !endpointIR.Stream.IsSSE && statusCode == expr.StatusSwitchingProtocols
		responses[strconv.Itoa(statusCode)] = buildResponse(resp, statusCode, statusBodies, rand, closeObjects, endpointServiceName(endpointIR), websocketHandshake)
	}
	for _, errResp := range endpointIR.Response.ErrorResponses {
		resp := buildResponse(errResp, errResp.StatusCode, statusBodies, rand, closeObjects, endpointServiceName(endpointIR), false)
		desc := resp.Description
		if value, ok := errResp.Meta.Last("openapi:description:errorName"); !ok || value != "false" {
			desc = errResp.Error.Name
			if resp.Description != "" {
				desc += ": " + resp.Description
			}
		}
		desc = appendErrorRemedyDescription(desc, errResp.Error)
		resp.Description = desc
		if errResp.Error.Type == expr.ErrorResult && len(errResp.Body.ExtractUserExamples()) == 0 {
			for _, content := range resp.Content {
				content.Example = nil
				content.Examples = nil
			}
		}
		responses[strconv.Itoa(errResp.StatusCode)] = resp
	}
	addFileResponseProtocolResponses(endpointIR, responses)
	return responses
}

func buildResponse(resp *transportir.ResponseStatus, statusCode int, bodies map[int][]*Schema, rand *expr.ExampleGenerator, closeObjects bool, currentService string, websocketHandshake bool) *Response {
	body := attributeForSchemaUsage(resp.DocumentBody, schemaUsageResponse)
	contentTypes := resp.ContentTypes
	headers := headersFromIR(resp.Headers, rand, closeObjects)
	if cookieHeader := responseCookieHeader(resp.Cookies, rand); cookieHeader != nil {
		if headers == nil {
			headers = make(map[string]*HeaderRef)
		}
		headers["Set-Cookie"] = &HeaderRef{Value: cookieHeader}
	}

	var content map[string]*MediaType
	switch {
	case websocketHandshake || resp.IsWebSocket:
		content = nil
	case body != nil && body.Type != expr.Empty:
		content = make(map[string]*MediaType, len(contentTypes))
		for _, contentType := range contentTypes {
			content[contentType] = buildMediaType(body, firstResponseBody(bodies[statusCode]), rand, closeObjects)
		}
		if !resp.EmitExamples {
			for _, mediaType := range content {
				mediaType.Example = nil
				mediaType.Examples = nil
			}
		}
	case resp.BinaryBody:
		content = make(map[string]*MediaType, len(contentTypes))
		for _, contentType := range contentTypes {
			content[contentType] = &MediaType{
				Schema: &Schema{
					Type:   "string",
					Format: "binary",
				},
				Extensions: openapi.ExtensionsFromExpr(resp.Meta),
			}
		}
	}

	desc := resp.Description
	if desc == "" {
		desc = fmt.Sprintf("%s response.", http.StatusText(statusCode))
	}
	return &Response{
		Description:     desc,
		Summary:         metaValue(resp.Meta, "openapi:summary"),
		OmitDescription: metaBool(resp.Meta, "openapi:description:omit"),
		ComponentName:   metaValue(resp.Meta, "openapi:component:response"),
		Headers:         headers,
		Content:         content,
		Links:           buildResponseLinks(resp.Links, currentService),
		Extensions: openapi.MergeExtensions(
			openapi.ExtensionsFromExpr(resp.Meta),
			openapi.ScopedExtensionsFromExpr(resp.Meta, "response"),
		),
	}
}

func appendErrorRemedyDescription(desc string, errResp *transportir.Error) string {
	if errResp == nil || errResp.Remedy == nil {
		return desc
	}
	parts := []string{desc}
	if errResp.Remedy.Code != "" {
		parts = append(parts, "Remedy code: "+errResp.Remedy.Code+".")
	}
	if errResp.Remedy.SafeMessage != "" {
		parts = append(parts, "Safe message: "+trimSentence(errResp.Remedy.SafeMessage)+".")
	}
	if errResp.Remedy.RetryHint != "" {
		parts = append(parts, "Retry hint: "+trimSentence(errResp.Remedy.RetryHint)+".")
	}
	return strings.Join(parts, " ")
}

func trimSentence(text string) string {
	return strings.TrimRight(text, ". ")
}

func buildMediaType(attr *expr.AttributeExpr, schema *Schema, rand *expr.ExampleGenerator, closeObjects bool) *MediaType {
	mediaType := &MediaType{
		Schema:        schema,
		ComponentName: componentMetaValue(attr, "openapi:component:mediaType"),
		Metadata:      cloneMeta(attr.Meta),
		Extensions:    openapi.ExtensionsFromExpr(attr.Meta),
	}
	initExamples(mediaType, attr, rand, closeObjects)
	return mediaType
}

func cloneMeta(meta expr.MetaExpr) map[string][]string {
	if len(meta) == 0 {
		return nil
	}
	cloned := make(map[string][]string, len(meta))
	for key, values := range meta {
		cloned[key] = append([]string(nil), values...)
	}
	return cloned
}

func headersFromIR(headersIR []*transportir.Header, rand *expr.ExampleGenerator, closeObjects bool) map[string]*HeaderRef {
	if len(headersIR) == 0 {
		return nil
	}
	analyzer := NewAnalyzer(rand, closeObjects)
	headers := make(map[string]*HeaderRef, len(headersIR))
	for _, headerIR := range headersIR {
		child := headerIR.Attribute
		if child == nil {
			continue
		}
		header := &Header{
			Description:   child.Description,
			Required:      child.IsRequiredNoDefault(headerIR.Name),
			AllowReserved: metaBool(child.Meta, "openapi:allowReserved"),
			Schema:        analyzer.AnalyzeSchema(child),
			Extensions:    openapi.ExtensionsFromExpr(child.Meta),
		}
		initExamples(header, child, rand, closeObjects)
		headers[headerIR.HTTPName] = &HeaderRef{Value: header}
	}
	return headers
}

func endpointServiceName(endpointIR *transportir.Endpoint) string {
	if endpointIR == nil || endpointIR.Service == nil {
		return ""
	}
	return endpointIR.Service.Name
}

func initExamples(target interface {
	setExample(any)
	setExamples(map[string]*ExampleRef)
}, attr *expr.AttributeExpr, rand *expr.ExampleGenerator, closeObjects bool) {
	if attr == nil {
		return
	}
	if disabled, ok := attr.Meta.Last("openapi:example"); ok && disabled == "false" {
		return
	}
	if objectContainsSuppressedOpenAPIExample(attr, closeObjects, map[string]struct{}{}, map[expr.DataType]struct{}{}) {
		return
	}
	if isUnionWrapperObjectType(attr.Type) {
		return
	}
	if closeObjects && isUnionType(attr.Type) {
		return
	}
	examples := attr.ExtractUserExamples()
	switch {
	case len(examples) > 1:
		refs := make(map[string]*ExampleRef, len(examples))
		for _, example := range examples {
			val, ok := openAPIExampleValue(attr, example.Value)
			if !ok {
				continue
			}
			refs[example.Summary] = &ExampleRef{Value: buildExample(example, val)}
		}
		if len(refs) > 0 {
			target.setExamples(refs)
		}
	case len(examples) == 1:
		if val, ok := openAPIExampleValue(attr, examples[0].Value); ok {
			if componentName := metaValue(examples[0].Meta, "openapi:component:example"); componentName != "" || hasStructuredExampleMetadata(examples[0].Meta) {
				name := examples[0].Summary
				if name == "" {
					name = "default"
				}
				target.setExamples(map[string]*ExampleRef{
					name: {Value: buildExample(examples[0], val)},
				})
				return
			}
			target.setExample(val)
		}
	default:
		if val, ok := openAPIExampleValue(attr, attr.Example(rand)); ok {
			target.setExample(val)
		}
	}
}

func buildExample(example *expr.ExampleExpr, value any) *Example {
	summary := example.Summary
	if authored, ok := example.Meta.Last("openapi:example:summary"); ok {
		summary = authored
	}
	out := &Example{
		Summary:       summary,
		Description:   example.Description,
		ComponentName: metaValue(example.Meta, "openapi:component:example"),
		Value:         value,
	}
	if _, ok := example.Meta["openapi:example:dataValue"]; ok {
		out.DataValue = value
		out.Value = nil
	}
	if serialized, ok := example.Meta.Last("openapi:example:serializedValue"); ok {
		out.SerializedValue = serialized
	}
	return out
}

func hasStructuredExampleMetadata(meta expr.MetaExpr) bool {
	_, dataValue := meta["openapi:example:dataValue"]
	_, serialized := meta["openapi:example:serializedValue"]
	return dataValue || serialized
}

func openAPIExampleValue(attr *expr.AttributeExpr, raw any) (any, bool) {
	if raw == nil {
		return nil, false
	}
	val := normalizeOpenAPIExample(expr.CanonicalizeExample(attr, raw))
	if !isCompleteOpenAPIExample(attr, val) {
		return nil, false
	}
	return val, true
}

func componentMetaValue(attr *expr.AttributeExpr, key string) string {
	if attr == nil {
		return ""
	}
	if value, ok := attr.Meta.Last(key); ok && strings.TrimSpace(value) != "" {
		return value
	}
	if userType, ok := attr.Type.(expr.UserType); ok {
		if value, ok := userType.Attribute().Meta.Last(key); ok && strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func metaValue(meta expr.MetaExpr, key string) string {
	if value, ok := meta.Last(key); ok && strings.TrimSpace(value) != "" {
		return value
	}
	return ""
}

func firstResponseBody(schemas []*Schema) *Schema {
	if len(schemas) == 0 {
		return nil
	}
	return schemas[0]
}

func cloneResponseBodies(bodies *EndpointBodies) map[int][]*Schema {
	if bodies == nil || len(bodies.ResponseBodies) == 0 {
		return nil
	}
	cloned := make(map[int][]*Schema, len(bodies.ResponseBodies))
	for status, schemas := range bodies.ResponseBodies {
		cloned[status] = append([]*Schema(nil), schemas...)
	}
	return cloned
}

func (m *MediaType) setExample(value any) {
	m.Example = value
}

func (m *MediaType) setExamples(value map[string]*ExampleRef) {
	m.Examples = value
}

func (h *Header) setExample(value any) {
	h.Example = value
}

func (h *Header) setExamples(value map[string]*ExampleRef) {
	h.Examples = value
}

func (p *Parameter) setExample(value any) {
	p.Example = value
}

func (p *Parameter) setExamples(value map[string]*ExampleRef) {
	p.Examples = value
}

func wrapResponses(responses map[string]*Response) map[string]*ResponseRef {
	if len(responses) == 0 {
		return nil
	}
	wrapped := make(map[string]*ResponseRef, len(responses))
	for status, response := range responses {
		wrapped[status] = &ResponseRef{Value: response}
	}
	return wrapped
}
