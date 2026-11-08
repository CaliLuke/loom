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
	bindings, err := newSecurityBindings(documentSecurityRequirements(api, apiRequirements)...)
	if err != nil {
		return nil, err
	}
	exampleGenerator := exampleGeneratorWithOptions(api.ExampleGenerator, options...)
	bodyTypes := BuildBodyTypes(api, types, resultTypes, options...)
	if bodyTypes.Err != nil {
		return nil, bodyTypes.Err
	}
	doc := &Document{
		Paths: make(map[string]*PathItem),
		Components: &Components{
			Schemas:               bodyTypes.Components,
			SecuritySchemes:       bindings.schemes,
			DiscriminatorChildren: bodyTypes.DiscriminatorChildren,
		},
		Security: bindings.requirements(apiRequirements),
	}
	closeObjects := openapi.ClosedObjectModeFromExpr(api.Meta)
	if err := buildDocumentPaths(doc, api, bodyTypes, exampleGenerator, closeObjects, bindings); err != nil {
		return nil, err
	}
	if err := componentizeDocument(doc); err != nil {
		return nil, err
	}
	return doc, nil
}

func documentSecurityRequirements(api *expr.APIExpr, apiRequirements []*expr.SecurityExpr) [][]*expr.SecurityExpr {
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
	return requirements
}

func buildDocumentPaths(
	doc *Document,
	api *expr.APIExpr,
	bodyTypes *BodyTypes,
	exampleGenerator *expr.ExampleGenerator,
	closeObjects bool,
	bindings *securityBindings,
) error {
	var identities []OperationIdentity
	for _, svc := range api.HTTP.Services {
		if !openapi.MustGenerate(svc.Meta) || !openapi.MustGenerate(svc.ServiceExpr.Meta) {
			continue
		}
		irService := bodyTypes.prepared[svc.Name()]
		serviceBodies := bodyTypes.Services[svc.Name()]
		if err := appendServiceOperations(doc, svc, irService, serviceBodies, exampleGenerator, api.Meta, closeObjects, bindings, &identities); err != nil {
			return err
		}
	}
	return ValidateOperationIdentities(identities)
}

func appendServiceOperations(
	doc *Document,
	svc *expr.HTTPServiceExpr,
	irService *transportir.Service,
	serviceBodies map[string]*EndpointBodies,
	exampleGenerator *expr.ExampleGenerator,
	apiMeta expr.MetaExpr,
	closeObjects bool,
	bindings *securityBindings,
	identities *[]OperationIdentity,
) error {
	for _, endpoint := range irService.Endpoints {
		if !endpoint.Generate || !endpoint.MethodGenerate {
			continue
		}
		if err := appendEndpointOperations(doc, svc.Name(), endpoint, serviceBodies[endpoint.Name], exampleGenerator, apiMeta, closeObjects, bindings, identities); err != nil {
			return err
		}
	}
	return nil
}

func appendEndpointOperations(
	doc *Document,
	serviceName string,
	endpoint *transportir.Endpoint,
	bodies *EndpointBodies,
	exampleGenerator *expr.ExampleGenerator,
	apiMeta expr.MetaExpr,
	closeObjects bool,
	bindings *securityBindings,
	identities *[]OperationIdentity,
) error {
	for _, route := range endpoint.Routes {
		key := expr.HTTPWildcardRegex.ReplaceAllString(route.Path, "/{$1}")
		operation, err := buildRouteOperationFromIR(endpoint, route, key, bodies, exampleGenerator, apiMeta, closeObjects, bindings)
		if err != nil {
			return err
		}
		*identities = append(*identities, OperationIdentity{
			Path:        key,
			Method:      route.Method,
			OperationID: operation.OperationID,
			Source:      serviceName + "." + endpoint.Name,
		})
		pathItem := doc.Paths[key]
		if pathItem == nil {
			pathItem = &PathItem{Operations: make(map[string]*Operation)}
			doc.Paths[key] = pathItem
		}
		pathItem.Operations[route.Method] = operation
	}
	return nil
}

func buildOperation(endpointIR *transportir.Endpoint, bodies *EndpointBodies, rand *expr.ExampleGenerator, closeObjects bool) (*Operation, error) {
	if endpointIR == nil {
		return nil, nil
	}
	responses, err := buildResponses(endpointIR, bodies, rand, closeObjects)
	if err != nil {
		return nil, err
	}
	request, err := buildRequestBody(endpointIR, bodies, closeObjects)
	if err != nil {
		return nil, err
	}
	return &Operation{
		RequestBody: wrapRequestBody(request),
		Responses:   wrapResponses(responses),
	}, nil
}

func buildRequestBody(endpointIR *transportir.Endpoint, bodies *EndpointBodies, closeObjects bool) (*RequestBody, error) {
	if endpointIR == nil || endpointIR.Request == nil {
		return nil, nil
	}
	body := expr.UnwrapInlineHTTPBody(endpointIR.Request.Body)
	contentTypes := []string{"application/json"}
	if endpointIR.Request.ContentType != "" {
		contentTypes = []string{endpointIR.Request.ContentType}
	}
	required := endpointIR.Request.MustHaveBody
	if endpointIR.Request.DocumentBody != nil {
		body = endpointIR.Request.DocumentBody
		contentTypes = endpointIR.Request.DocumentContentTypes
		required = endpointIR.Request.DocumentRequired
	}
	if body == nil || body.Type == expr.Empty {
		return nil, nil
	}
	bodyAttr := attributeForSchemaUsage(body, schemaUsageRequest)
	if endpointIR.Request.Multipart {
		contentTypes = []string{"multipart/form-data"}
	} else if endpointIR.Request.FormEncoded {
		contentTypes = []string{"application/x-www-form-urlencoded"}
	}
	content := make(map[string]*MediaType, len(contentTypes))
	if len(endpointIR.Request.MediaVariants) > 0 {
		var err error
		content, err = buildMediaVariantContent(endpointIR.Request.MediaVariants, bodies.requestMedia, closeObjects, schemaUsageRequest)
		if err != nil {
			return nil, err
		}
	} else {
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
			if err := applyMediaFieldEncoding(endpointIR.Request, content[contentType]); err != nil {
				return nil, fmt.Errorf("media type %q: %w", contentType, err)
			}
		}
	}
	return &RequestBody{
		ReferenceMetadata: componentMetadata(bodyAttr, "openapi:reference:requestBody"),
		Description:       requestBodyDescription(bodyAttr),
		Required:          required,
		ComponentName:     componentMetaValue(bodyAttr, "openapi:component:requestBody"),
		Content:           content,
		Extensions: openapi.MergeExtensions(
			openapi.ExtensionsFromExpr(bodyAttr.Meta),
			openapi.ScopedExtensionsFromExpr(bodyAttr.Meta, "requestBody"),
		),
	}, nil
}

func requestBodyDescription(bodyAttr *expr.AttributeExpr) string {
	if bodyAttr == nil {
		return ""
	}
	if len(componentMetadata(bodyAttr, "openapi:reference:requestBody")) > 0 {
		if values := componentMetadata(bodyAttr, "openapi:description:requestBody"); len(values) > 0 {
			return values[len(values)-1]
		}
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
		selector := resp.Selector
		if endpointIR.Stream.IsStreaming && !endpointIR.Stream.IsSSE {
			if _, ok := responses[strconv.Itoa(expr.StatusSwitchingProtocols)]; !ok {
				statusBodies["101"] = statusBodies[resp.Selector.OpenAPIKey()]
				delete(statusBodies, resp.Selector.OpenAPIKey())
				selector = expr.HTTPStatusSelector{Code: expr.StatusSwitchingProtocols}
			}
		}
		websocketHandshake := endpointIR.Stream.IsStreaming && !endpointIR.Stream.IsSSE && selector == (expr.HTTPStatusSelector{Code: expr.StatusSwitchingProtocols})
		response, err := buildResponse(resp, selector, statusBodies, rand, closeObjects, endpointServiceName(endpointIR), websocketHandshake, bodies.locations, bodies.responseMedia[resp], bodies.eventFor(resp), bodies.unaryMedia[resp])
		if err != nil {
			return nil, err
		}
		status := selector.OpenAPIKey()
		alternatives[status] = append(alternatives[status], responseAlternative{fmt.Sprintf("success-%d", index), response})
		responses[status] = response
	}
	for _, errResp := range endpointIR.Response.ErrorResponses {
		resp, err := buildResponse(errResp, errResp.Selector, statusBodies, rand, closeObjects, endpointServiceName(endpointIR), false, bodies.locations, bodies.responseMedia[errResp], nil, nil)
		if err != nil {
			return nil, err
		}
		describeErrorResponse(errResp, resp)
		status := errResp.Selector.OpenAPIKey()
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
	if err := addInformationalResponses(endpointIR, bodies, rand, closeObjects, responses); err != nil {
		return nil, err
	}
	addFileResponseProtocolResponses(endpointIR, responses)
	return responses, nil
}

// describeErrorResponse prefixes the error name and remedy and drops synthesized
// examples of the built-in error result.
func describeErrorResponse(errResp *transportir.ResponseStatus, resp *Response) {
	desc := resp.Description
	if value, ok := errResp.Meta.Last("openapi:description:errorName"); !ok || value != "false" {
		desc = errResp.Error.Name
		if resp.Description != "" {
			desc += ": " + resp.Description
		}
	}
	resp.Description = appendErrorRemedyDescription(desc, errResp.Error)
	if errResp.Error.Type == expr.ErrorResult && len(errResp.Body.ExtractUserExamples()) == 0 {
		for _, content := range resp.Content {
			content.Example = nil
			content.Examples = nil
		}
	}
}

// addInformationalResponses documents each interim declaration under its own
// informational key: headers and description, never content.
func addInformationalResponses(endpointIR *transportir.Endpoint, bodies *EndpointBodies, rand *expr.ExampleGenerator, closeObjects bool, responses map[string]*Response) error {
	for _, info := range endpointIR.Response.Informational {
		response, err := buildResponse(info.Response, info.Response.Selector, nil, rand, closeObjects, endpointServiceName(endpointIR), false, bodies.locations, nil, nil, nil)
		if err != nil {
			return err
		}
		responses[info.Response.Selector.OpenAPIKey()] = response
	}
	return nil
}

func buildResponse(
	resp *transportir.ResponseStatus,
	selector expr.HTTPStatusSelector,
	bodies map[string][]*Schema,
	rand *expr.ExampleGenerator,
	closeObjects bool,
	currentService string,
	websocketHandshake bool,
	prepared map[*expr.AttributeExpr]*Schema,
	mediaSchemas map[string]*Schema,
	event *sseEventOutcome,
	unaryMedia map[string]*Schema,
) (*Response, error) {
	headers := headersFromIR(resp.Headers, rand, closeObjects, prepared)
	if len(resp.MediaVariants) > 0 {
		for name := range headers {
			if strings.EqualFold(name, "Content-Type") {
				delete(headers, name) // The content map owns representation selection.
			}
		}
	}
	if cookieHeader := responseCookieHeader(resp.Cookies); cookieHeader != nil {
		if headers == nil {
			headers = make(map[string]*HeaderRef)
		}
		headers["Set-Cookie"] = &HeaderRef{Value: cookieHeader}
	}

	desc := resp.Description
	if desc == "" && len(resp.Meta["openapi:reference:response"]) == 0 {
		if code, exact := selector.Exact(); exact {
			desc = fmt.Sprintf("%s response.", http.StatusText(code))
		} else {
			desc = responseComponentDescription(selector.OpenAPIKey())
		}
	}
	content, err := buildResponseContent(resp, selector, bodies, closeObjects, websocketHandshake, mediaSchemas, event)
	if err != nil {
		return nil, err
	}
	if content, err = addUnaryContent(content, resp, unaryMedia, closeObjects); err != nil {
		return nil, err
	}
	return &Response{
		ReferenceMetadata: resp.Meta["openapi:reference:response"],
		Description:       desc,
		Summary:           metaValue(resp.Meta, "openapi:summary"),
		OmitDescription:   metaBool(resp.Meta, "openapi:description:omit"),
		ComponentName:     metaValue(resp.Meta, "openapi:component:response"),
		Headers:           headers,
		Content:           content,
		Links:             buildResponseLinks(resp.Links, currentService),
		Extensions: openapi.MergeExtensions(
			openapi.ExtensionsFromExpr(resp.Meta),
			openapi.ScopedExtensionsFromExpr(resp.Meta, "response"),
		),
	}, nil
}

func buildResponseContent(
	resp *transportir.ResponseStatus,
	selector expr.HTTPStatusSelector,
	bodies map[string][]*Schema,
	closeObjects bool,
	websocketHandshake bool,
	mediaSchemas map[string]*Schema,
	event *sseEventOutcome,
) (map[string]*MediaType, error) {
	if len(resp.MediaVariants) > 0 {
		return buildMediaVariantContent(resp.MediaVariants, mediaSchemas, closeObjects, schemaUsageResponse)
	}
	body := attributeForSchemaUsage(resp.DocumentBody, schemaUsageResponse)
	if event != nil {
		body = event.attribute
	}
	contentTypes := resp.ContentTypes
	var content map[string]*MediaType
	switch {
	case websocketHandshake:
		content = nil
	case body != nil && body.Type != expr.Empty:
		content = make(map[string]*MediaType, len(contentTypes))
		for _, contentType := range contentTypes {
			schema := firstResponseBody(bodies[selector.OpenAPIKey()])
			target := resp.BodyValue
			if event != nil {
				target = event.target
			} else if prepared, exists := mediaSchemas[contentType]; exists {
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
	return content, nil
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
		Metadata:      mediaTypeMetadata(expr.HTTPMediaMetadata(attr)),
		Extensions:    openapi.ScopedExtensionsFromExpr(attr.Meta, "mediaType"),
	}
	initExamples(mediaType, attr, closeObjects, prepared)
	if rawBody {
		serializeBinaryMediaExamples(mediaType, attr)
	}
	return mediaType
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
			ReferenceMetadata: componentMetadata(child, "openapi:reference:header"),
			Description:       child.Description,
			Required:          child.IsRequiredNoDefault(headerIR.Name),
			AllowReserved:     metaBool(expr.HTTPMediaMetadata(child), "openapi:allowReserved"),
			Schema:            preparedLocationSchema(child, headerContext, rand, closeObjects, prepared),
			Extensions:        openapi.ScopedExtensionsFromExpr(child.Meta, "header"),
		}
		header.Style, header.Explode = parameterCodecFields("header", headerIR.Codec)
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
		if expr.AllowsNull(attr) {
			for _, example := range attr.ExtractUserExamples() {
				if example.ExplicitNull {
					return NullExample{}, true
				}
			}
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

func cloneResponseBodies(bodies *EndpointBodies) map[string][]*Schema {
	if bodies == nil || len(bodies.ResponseBodies) == 0 {
		return nil
	}
	cloned := make(map[string][]*Schema, len(bodies.ResponseBodies))
	for status, schemas := range bodies.ResponseBodies {
		cloned[status] = append([]*Schema(nil), schemas...)
	}
	return cloned
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
	var schema *Schema
	if len(prepared) > 0 && prepared[0] != nil {
		var exists bool
		schema, exists = prepared[0][attribute]
		if !exists {
			panic("OpenAPI location missing prepared representation")
		}
		if schema == nil {
			return nil
		}
		copy := *schema
		schema = &copy
	} else {
		analyzer := NewAnalyzer(random, closeObjects)
		schema = analyzer.AnalyzeSchemaWithContext(attribute, context)
		if err := analyzer.Err(); err != nil {
			panic(err)
		}
	}
	if description, explicit := attribute.Meta.Last("openapi:description:schema"); explicit {
		schema.Description = description
	}
	return schema
}

// componentMetadata preserves local presence, including empty values, before
// consulting a named declaration. Reference edge metadata is never combined.
func componentMetadata(attr *expr.AttributeExpr, key string) []string {
	if attr == nil {
		return nil
	}
	if values, present := attr.Meta[key]; present {
		return values
	}
	if userType, ok := attr.Type.(expr.UserType); ok {
		return userType.Attribute().Meta[key]
	}
	return nil
}
