package ir

import (
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
	"github.com/CaliLuke/loom/http/codegen/openapi"
	"github.com/CaliLuke/loom/internal/enumvalue"
)

// BuildDocument analyzes HTTP operations, schemas and security bindings. It
// returns an error when OpenAPI cannot represent a credential location or
// reconcile response metadata for alternatives sharing a status.
func BuildDocument(api *expr.APIExpr, types []expr.UserType, resultTypes []*expr.ResultTypeExpr, options ...AnalyzerOption) (*Document, error) {
	if api == nil || api.HTTP == nil {
		return nil, nil
	}
	apiRequirements := apiSecurityRequirements(api)
	requirements := [][]*expr.SecurityExpr{apiRequirements}
	for _, svc := range api.HTTP.Services {
		if !openapi.MustGenerate(svc.Meta) || !openapi.MustGenerate(svc.ServiceExpr.Meta) {
			continue
		}
		for _, endpoint := range svc.HTTPEndpoints {
			if openapi.MustGenerate(endpoint.Meta) && openapi.MustGenerate(endpoint.MethodExpr.Meta) {
				requirements = append(requirements, endpoint.Requirements)
			}
		}
	}
	bindings, err := newSecurityBindings(requirements...)
	if err != nil {
		return nil, err
	}
	exampleGenerator := exampleGeneratorWithOptions(api.ExampleGenerator, options...)
	bodyTypes := BuildBodyTypes(api, types, resultTypes, options...)
	doc := &Document{
		Paths: make(map[string]*PathItem),
		Components: &Components{
			Schemas:         bodyTypes.Components,
			SecuritySchemes: bindings.schemes,
		},
		Security: bindings.requirements(apiRequirements),
	}
	closeObjects := openapi.ClosedObjectModeFromExpr(api.Meta)
	for _, svc := range api.HTTP.Services {
		if !openapi.MustGenerate(svc.Meta) || !openapi.MustGenerate(svc.ServiceExpr.Meta) {
			continue
		}
		irService := bodyTypes.prepared[svc.Name()]
		serviceBodies := bodyTypes.Services[svc.Name()]
		for _, endpoint := range irService.Endpoints {
			if !endpoint.Generate || !endpoint.MethodGenerate {
				continue
			}
			for _, route := range endpoint.Routes {
				key := expr.HTTPWildcardRegex.ReplaceAllString(route.Path, "/{$1}")
				operation, err := buildRouteOperationFromIR(endpoint, route, key, serviceBodies[endpoint.Name], exampleGenerator, api.Meta, closeObjects, bindings)
				if err != nil {
					return nil, err
				}
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
	return doc, nil
}

func buildOperation(endpointIR *transportir.Endpoint, bodies *EndpointBodies, rand *expr.ExampleGenerator, closeObjects bool) (*Operation, error) {
	if endpointIR == nil {
		return nil, nil
	}
	responses, err := buildResponses(endpointIR, bodies, rand, closeObjects)
	if err != nil {
		return nil, err
	}
	return &Operation{
		RequestBody: wrapRequestBody(buildRequestBody(endpointIR, bodies, closeObjects)),
		Responses:   wrapResponses(responses),
	}, nil
}

func buildRequestBody(endpointIR *transportir.Endpoint, bodies *EndpointBodies, closeObjects bool) *RequestBody {
	if endpointIR == nil || endpointIR.Request == nil {
		return nil
	}
	body := expr.UnwrapInlineHTTPBody(endpointIR.Request.Body)
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
	content := make(map[string]*MediaType, len(contentTypes))
	for _, contentType := range contentTypes {
		schema := bodies.RequestBody
		target := endpointIR.Request.BodyValue
		if prepared, exists := bodies.requestMedia[contentType]; exists {
			schema = prepared
			target = endpointIR.Request.DocumentValues[contentType]
		} else if endpointIR.Request.DocumentBody != nil {
			target = endpointIR.Request.DocumentValue
		}
		content[contentType] = buildMediaType(bodyAttr, schema, closeObjects,
			endpointIR.Request.DocumentBody != nil, target)
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

func buildResponses(endpointIR *transportir.Endpoint, bodies *EndpointBodies, rand *expr.ExampleGenerator, closeObjects bool) (map[string]*Response, error) {
	responses := make(map[string]*Response, len(endpointIR.Response.Responses)+len(endpointIR.Response.ErrorResponses))
	alternatives := make(map[string][]responseAlternative)
	statusBodies := cloneResponseBodies(bodies)
	for index, resp := range endpointIR.Response.Responses {
		statusCode := resp.StatusCode
		if endpointIR.Stream.IsStreaming && !endpointIR.Stream.IsSSE {
			if _, ok := responses[strconv.Itoa(expr.StatusSwitchingProtocols)]; !ok {
				statusBodies[expr.StatusSwitchingProtocols] = statusBodies[resp.StatusCode]
				delete(statusBodies, resp.StatusCode)
				statusCode = expr.StatusSwitchingProtocols
			}
		}
		websocketHandshake := endpointIR.Stream.IsStreaming && !endpointIR.Stream.IsSSE && statusCode == expr.StatusSwitchingProtocols
		response := buildResponse(resp, statusCode, statusBodies, rand, closeObjects, endpointServiceName(endpointIR), websocketHandshake, bodies.locations, bodies.responseMedia[resp])
		status := strconv.Itoa(statusCode)
		alternatives[status] = append(alternatives[status], responseAlternative{fmt.Sprintf("success-%d", index), response})
		responses[status] = response
	}
	for _, errResp := range endpointIR.Response.ErrorResponses {
		resp := buildResponse(errResp, errResp.StatusCode, statusBodies, rand, closeObjects, endpointServiceName(endpointIR), false, bodies.locations, bodies.responseMedia[errResp])
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
		status := strconv.Itoa(errResp.StatusCode)
		alternatives[status] = append(alternatives[status], responseAlternative{errResp.Error.Name, resp})
	}
	for _, status := range slices.Sorted(maps.Keys(alternatives)) {
		variants := alternatives[status]
		response, err := mergeResponseAlternatives(variants)
		if err != nil {
			return nil, fmt.Errorf("OpenAPI response %s for %s.%s: %w", status, endpointServiceName(endpointIR), endpointIR.Name, err)
		}
		responses[status] = response
	}
	addFileResponseProtocolResponses(endpointIR, responses)
	return responses, nil
}

func buildResponse(
	resp *transportir.ResponseStatus,
	statusCode int,
	bodies map[int][]*Schema,
	rand *expr.ExampleGenerator,
	closeObjects bool,
	currentService string,
	websocketHandshake bool,
	prepared map[*expr.AttributeExpr]*Schema,
	mediaSchemas map[string]*Schema,
) *Response {
	headers := headersFromIR(resp.Headers, rand, closeObjects, prepared)
	if cookieHeader := responseCookieHeader(resp.Cookies); cookieHeader != nil {
		if headers == nil {
			headers = make(map[string]*HeaderRef)
		}
		headers["Set-Cookie"] = &HeaderRef{Value: cookieHeader}
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
		Content:         buildResponseContent(resp, statusCode, bodies, closeObjects, websocketHandshake, mediaSchemas),
		Links:           buildResponseLinks(resp.Links, currentService),
		Extensions: openapi.MergeExtensions(
			openapi.ExtensionsFromExpr(resp.Meta),
			openapi.ScopedExtensionsFromExpr(resp.Meta, "response"),
		),
	}
}

func buildResponseContent(
	resp *transportir.ResponseStatus,
	statusCode int,
	bodies map[int][]*Schema,
	closeObjects bool,
	websocketHandshake bool,
	mediaSchemas map[string]*Schema,
) map[string]*MediaType {
	body := attributeForSchemaUsage(resp.DocumentBody, schemaUsageResponse)
	contentTypes := resp.ContentTypes
	var content map[string]*MediaType
	switch {
	case websocketHandshake || resp.IsWebSocket:
		content = nil
	case body != nil && body.Type != expr.Empty:
		content = make(map[string]*MediaType, len(contentTypes))
		for _, contentType := range contentTypes {
			schema := firstResponseBody(bodies[statusCode])
			target := resp.BodyValue
			if prepared, exists := mediaSchemas[contentType]; exists {
				schema = prepared
				target = resp.DocumentValues[contentType]
			} else if prepared, exists := resp.BodyValues[contentType]; exists {
				target = prepared
			} else if resp.DocumentBody != nil {
				target = resp.DocumentValue
			}
			content[contentType] = buildMediaType(
				body,
				schema,
				closeObjects,
				resp.BinaryBody,
				target,
			)
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
				Extensions: openapi.ScopedExtensionsFromExpr(resp.Meta, "mediaType"),
			}
		}
	}
	return content
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

func buildMediaType(
	attr *expr.AttributeExpr,
	schema *Schema,
	closeObjects bool,
	rawBody bool,
	prepared *transportir.ValueTarget,
) *MediaType {
	mediaType := &MediaType{
		Schema:        schema,
		ComponentName: componentMetaValue(attr, "openapi:component:mediaType"),
		Metadata:      mediaTypeMetadata(attr.Meta),
		Extensions:    openapi.ScopedExtensionsFromExpr(attr.Meta, "mediaType"),
	}
	initExamples(mediaType, attr, closeObjects, prepared)
	if rawBody {
		serializeBinaryMediaExamples(mediaType, attr)
	}
	return mediaType
}

// mediaTypeMetadata captures only annotations lowered on the media object.
// Body naming and schema annotations belong to the selected schema, not to
// compatibility checks between media objects that can contain several schemas.
func mediaTypeMetadata(meta expr.MetaExpr) map[string][]string {
	var selected map[string][]string
	for key, values := range meta {
		if key == "openapi:itemSchema" || strings.HasPrefix(key, "openapi:encoding:") ||
			strings.HasPrefix(key, "openapi:prefixEncoding:") || strings.HasPrefix(key, "openapi:itemEncoding:") {
			if selected == nil {
				selected = make(map[string][]string)
			}
			selected[key] = slices.Clone(values)
		}
	}
	return selected
}

func headersFromIR(
	headersIR []*transportir.Header,
	rand *expr.ExampleGenerator,
	closeObjects bool,
	prepared ...map[*expr.AttributeExpr]*Schema,
) map[string]*HeaderRef {
	if len(headersIR) == 0 {
		return nil
	}
	headers := make(map[string]*HeaderRef, len(headersIR))
	for _, headerIR := range headersIR {
		child := headerIR.Attribute
		if child == nil {
			continue
		}
		headerContext := attributeExampleContext(child, closeObjects, "response-header", headerIR.HTTPName)
		header := &Header{
			Description:   child.Description,
			Required:      child.IsRequiredNoDefault(headerIR.Name),
			AllowReserved: metaBool(child.Meta, "openapi:allowReserved"),
			Schema:        preparedLocationSchema(child, headerContext, rand, closeObjects, prepared),
			Extensions:    openapi.ScopedExtensionsFromExpr(child.Meta, "header"),
		}
		initExamples(header, child, closeObjects, headerIR.Value)
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
}, attr *expr.AttributeExpr, closeObjects bool, prepared *transportir.ValueTarget) {
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
	requirePreparedExampleTarget(prepared)
	if !initPreparedExamples(target, prepared) {
		panic("OpenAPI example surface did not consume its prepared value target")
	}
}

func requirePreparedExampleTarget(prepared *transportir.ValueTarget) {
	if prepared == nil {
		panic("OpenAPI example surface is missing its prepared value target")
	}
	if !prepared.ExamplesPrepared {
		panic("OpenAPI example surface value target has not prepared examples")
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

// OpenAPIExampleValue normalizes an example and reports whether it completely
// represents its attribute's OpenAPI schema.
func OpenAPIExampleValue(attr *expr.AttributeExpr, raw any) (any, bool) {
	if raw == nil {
		if examples := attr.ExtractUserExamples(); len(examples) > 0 &&
			examples[len(examples)-1].ExplicitNull && expr.AllowsNull(attr) {
			return NullExample{}, true
		}
		return nil, false
	}
	canonical := expr.CanonicalizeExample(attr, raw)
	if canonical != nil {
		canonical = enumvalue.Normalize(attr, raw)
	}
	return projectCompleteOpenAPIExample(attr, canonical)
}

func projectCompleteOpenAPIExample(attr *expr.AttributeExpr, value any) (any, bool) {
	val := normalizeOpenAPIExampleForAttribute(attr, projectOpenAPIExample(attr, value))
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

func preparedLocationSchema(attribute *expr.AttributeExpr, context string, random *expr.ExampleGenerator, closeObjects bool, prepared []map[*expr.AttributeExpr]*Schema) *Schema {
	if len(prepared) > 0 && prepared[0] != nil {
		schema, exists := prepared[0][attribute]
		if !exists {
			panic("OpenAPI location missing prepared representation")
		}
		if schema == nil {
			return nil
		}
		copy := *schema
		return &copy
	}
	return NewAnalyzer(random, closeObjects).AnalyzeSchemaWithContext(attribute, context)
}
