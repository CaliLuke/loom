package ir

import (
	"fmt"
	"net/http"
	"reflect"
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

// BuildOperation analyzes HTTP body/content-related OpenAPI operation data.
func BuildOperation(endpoint *expr.HTTPEndpointExpr, bodies *EndpointBodies, rand *expr.ExampleGenerator, closeObjects bool) *Operation {
	return buildOperation(transportir.BuildEndpoint(endpoint), bodies, rand, closeObjects)
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
	if endpointIR == nil || endpointIR.Request == nil || endpointIR.Request.Body == nil || endpointIR.Request.Body.Type == expr.Empty {
		return nil
	}
	bodyAttr := attributeForSchemaUsage(endpointIR.Request.Body, schemaUsageRequest)
	contentType := "application/json"
	if endpointIR.Request.Multipart {
		contentType = "multipart/form-data"
	} else if endpointIR.Request.FormEncoded {
		contentType = "application/x-www-form-urlencoded"
	}
	return &RequestBody{
		Description:   bodyAttr.Description,
		Required:      endpointIR.Request.MustHaveBody,
		ComponentName: componentMetaValue(bodyAttr, "openapi:component:requestBody"),
		Content: map[string]*MediaType{
			contentType: buildMediaType(bodyAttr, bodies.RequestBody, rand, closeObjects),
		},
		Extensions: openapi.ExtensionsFromExpr(bodyAttr.Meta),
	}
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
		desc := errResp.Error.Name
		if resp.Description != "" {
			desc += ": " + resp.Description
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
	return responses
}

func buildResponse(resp *transportir.ResponseStatus, statusCode int, bodies map[int][]*Schema, rand *expr.ExampleGenerator, closeObjects bool, currentService string, websocketHandshake bool) *Response {
	body := attributeForSchemaUsage(resp.DocumentBody, schemaUsageResponse)
	contentTypes := resp.ContentTypes
	headers := headersFromAttr(resp.Headers, rand, closeObjects)
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
		Description: desc,
		Headers:     headers,
		Content:     content,
		Links:       buildResponseLinks(resp.Links, currentService),
		Extensions:  openapi.ExtensionsFromExpr(resp.Meta),
	}
}

func appendErrorRemedyDescription(desc string, errResp *expr.HTTPErrorExpr) string {
	if errResp == nil || errResp.ErrorExpr == nil || errResp.ErrorExpr.Remedy == nil {
		return desc
	}
	parts := []string{desc}
	if errResp.ErrorExpr.Remedy.Code != "" {
		parts = append(parts, "Remedy code: "+errResp.ErrorExpr.Remedy.Code+".")
	}
	if errResp.ErrorExpr.Remedy.SafeMessage != "" {
		parts = append(parts, "Safe message: "+trimSentence(errResp.ErrorExpr.Remedy.SafeMessage)+".")
	}
	if errResp.ErrorExpr.Remedy.RetryHint != "" {
		parts = append(parts, "Retry hint: "+trimSentence(errResp.ErrorExpr.Remedy.RetryHint)+".")
	}
	return strings.Join(parts, " ")
}

func trimSentence(text string) string {
	return strings.TrimRight(text, ". ")
}

func buildMediaType(attr *expr.AttributeExpr, schema *Schema, rand *expr.ExampleGenerator, closeObjects bool) *MediaType {
	mediaType := &MediaType{
		Schema:     schema,
		Extensions: openapi.ExtensionsFromExpr(attr.Meta),
	}
	initExamples(mediaType, attr, rand, closeObjects)
	return mediaType
}

func headersFromAttr(attr *expr.MappedAttributeExpr, rand *expr.ExampleGenerator, closeObjects bool) map[string]*HeaderRef {
	if attr == nil {
		return nil
	}
	object := expr.AsObject(attr.Type)
	if len(*object) == 0 {
		return nil
	}
	analyzer := NewAnalyzer(rand, closeObjects)
	headers := make(map[string]*HeaderRef, len(*object))
	expr.WalkMappedAttr(attr, func(name, elem string, child *expr.AttributeExpr) error { // nolint: errcheck
		header := &Header{
			Description: child.Description,
			Required:    child.IsRequiredNoDefault(name),
			Schema:      analyzer.AnalyzeSchema(child),
			Extensions:  openapi.ExtensionsFromExpr(child.Meta),
		}
		initExamples(header, child, rand, closeObjects)
		headers[elem] = &HeaderRef{Value: header}
		return nil
	})
	return headers
}

func responseCookieHeader(cookies []*expr.HTTPResponseCookieExpr, rand *expr.ExampleGenerator) *Header {
	if len(cookies) == 0 {
		return nil
	}
	header := &Header{
		Required: true,
		Schema: &Schema{
			Type: "string",
		},
	}
	if len(cookies) == 1 {
		cookie := cookies[0]
		header.Description = describeResponseCookie(cookie)
		header.Example = serializeResponseCookieExample(cookie, cookie.Attribute().Example(rand))
		return header
	}
	header.Description = describeResponseCookies(cookies)
	header.Examples = make(map[string]*ExampleRef, len(cookies))
	for _, cookie := range cookies {
		header.Examples[cookie.HTTPName()] = &ExampleRef{Value: &Example{
			Summary:     fmt.Sprintf("%s cookie", cookie.HTTPName()),
			Description: describeResponseCookie(cookie),
			Value:       serializeResponseCookieExample(cookie, cookie.Attribute().Example(rand)),
		}}
	}
	return header
}

func describeResponseCookie(cookie *expr.HTTPResponseCookieExpr) string {
	parts := []string{fmt.Sprintf("Sets the %q cookie.", cookie.HTTPName())}
	if attr := cookie.Attribute(); attr != nil && attr.Description != "" {
		parts = append(parts, attr.Description)
	}
	if policy := responseCookiePolicy(cookie); policy != "" {
		parts = append(parts, "Policy: "+policy+".")
	}
	return strings.Join(parts, " ")
}

func describeResponseCookies(cookies []*expr.HTTPResponseCookieExpr) string {
	lines := make([]string, 0, 1+len(cookies))
	lines = append(lines, "Set-Cookie headers issued by the server:")
	for _, cookie := range cookies {
		lines = append(lines, "- "+describeResponseCookie(cookie))
	}
	return strings.Join(lines, "\n")
}

func responseCookiePolicy(cookie *expr.HTTPResponseCookieExpr) string {
	parts := make([]string, 0, 6)
	httpCookie := buildResponseHTTPCookie(cookie, "")
	if httpCookie.Path != "" {
		parts = append(parts, "Path="+httpCookie.Path)
	}
	if httpCookie.Domain != "" {
		parts = append(parts, "Domain="+httpCookie.Domain)
	}
	if cookie.MaxAge != "" {
		parts = append(parts, "Max-Age="+strconv.Itoa(normalizeCookieMaxAge(httpCookie.MaxAge)))
	}
	if httpCookie.Secure {
		parts = append(parts, "Secure")
	}
	if httpCookie.HttpOnly {
		parts = append(parts, "HttpOnly")
	}
	if sameSite := sameSiteString(httpCookie.SameSite); sameSite != "" {
		parts = append(parts, "SameSite="+sameSite)
	}
	return strings.Join(parts, "; ")
}

func serializeResponseCookieExample(cookie *expr.HTTPResponseCookieExpr, value any) string {
	return buildResponseHTTPCookie(cookie, fmt.Sprintf("%v", value)).String()
}

func buildResponseHTTPCookie(cookie *expr.HTTPResponseCookieExpr, value string) *http.Cookie {
	httpCookie := &http.Cookie{
		Name:     cookie.HTTPName(),
		Value:    value,
		Path:     cookie.Path,
		Domain:   cookie.Domain,
		Secure:   cookie.Secure,
		HttpOnly: cookie.HTTPOnly,
	}
	if cookie.MaxAge != "" {
		if maxAge, err := strconv.Atoi(cookie.MaxAge); err == nil {
			httpCookie.MaxAge = maxAge
		}
	}
	switch cookie.SameSite {
	case expr.CookieSameSiteLax:
		httpCookie.SameSite = http.SameSiteLaxMode
	case expr.CookieSameSiteStrict:
		httpCookie.SameSite = http.SameSiteStrictMode
	case expr.CookieSameSiteNone:
		httpCookie.SameSite = http.SameSiteNoneMode
	case expr.CookieSameSiteDefault:
		httpCookie.SameSite = http.SameSiteDefaultMode
	}
	return httpCookie
}

func sameSiteString(mode http.SameSite) string {
	switch mode {
	case http.SameSiteDefaultMode:
		return "Default"
	case http.SameSiteLaxMode:
		return "Lax"
	case http.SameSiteStrictMode:
		return "Strict"
	case http.SameSiteNoneMode:
		return "None"
	default:
		return ""
	}
}

func normalizeCookieMaxAge(maxAge int) int {
	if maxAge < 0 {
		return 0
	}
	return maxAge
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
			refs[example.Summary] = &ExampleRef{Value: &Example{
				Summary:     example.Summary,
				Description: example.Description,
				Value:       val,
			}}
		}
		if len(refs) > 0 {
			target.setExamples(refs)
		}
	case len(examples) == 1:
		if val, ok := openAPIExampleValue(attr, examples[0].Value); ok {
			target.setExample(val)
		}
	default:
		if val, ok := openAPIExampleValue(attr, attr.Example(rand)); ok {
			target.setExample(val)
		}
	}
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

func objectContainsSuppressedOpenAPIExample(attr *expr.AttributeExpr, closeObjects bool, seenUT map[string]struct{}, seenDT map[expr.DataType]struct{}) bool {
	if attr == nil || attr.Type == nil {
		return false
	}
	if _, ok := seenDT[attr.Type]; ok {
		return false
	}
	seenDT[attr.Type] = struct{}{}

	switch actual := attr.Type.(type) {
	case expr.UserType:
		id := actual.ID()
		if _, ok := seenUT[id]; ok {
			return false
		}
		seenUT[id] = struct{}{}
		return objectContainsSuppressedOpenAPIExample(actual.Attribute(), closeObjects, seenUT, seenDT)
	case *expr.Array:
		return objectContainsSuppressedOpenAPIExample(actual.ElemType, closeObjects, seenUT, seenDT)
	case *expr.Map:
		return objectContainsSuppressedOpenAPIExample(actual.KeyType, closeObjects, seenUT, seenDT) ||
			objectContainsSuppressedOpenAPIExample(actual.ElemType, closeObjects, seenUT, seenDT)
	case *expr.Object:
		for _, nat := range *actual {
			if nat == nil || nat.Attribute == nil {
				continue
			}
			if disabled, ok := nat.Attribute.Meta.Last("openapi:example"); ok && disabled == "false" {
				return true
			}
			if isUnionWrapperObjectTypeSeen(nat.Attribute.Type, seenUT, seenDT) {
				return true
			}
			if closeObjects && isUnionTypeSeen(nat.Attribute.Type, seenUT) {
				return true
			}
			if objectContainsSuppressedOpenAPIExample(nat.Attribute, closeObjects, seenUT, seenDT) {
				return true
			}
		}
	}
	return false
}

func isCompleteOpenAPIExample(attr *expr.AttributeExpr, val any) bool {
	if attr == nil {
		return val != nil
	}
	if val == nil {
		return false
	}
	switch actual := attr.Type.(type) {
	case expr.UserType:
		return isCompleteOpenAPIExample(actual.Attribute(), val)
	case *expr.Object:
		return completeOpenAPIObjectExample(attr, val)
	case *expr.Array:
		return completeOpenAPIArrayExample(actual, val)
	case *expr.Map:
		return completeOpenAPIMapExample(actual, val)
	case *expr.Union:
		return completeOpenAPIUnionExample(actual, val)
	default:
		return true
	}
}

func completeOpenAPIObjectExample(attr *expr.AttributeExpr, val any) bool {
	obj, ok := val.(map[string]any)
	if !ok {
		return false
	}
	required := attr.AllRequired()
	if len(obj) == 0 && len(required) > 0 {
		return false
	}
	for _, name := range required {
		if !requiredOpenAPIFieldPresent(attr, obj, name) {
			return false
		}
	}
	return true
}

func requiredOpenAPIFieldPresent(attr *expr.AttributeExpr, obj map[string]any, name string) bool {
	child := attr.Find(name)
	if child == nil || !openapi.MustGenerate(child.Meta) {
		return true
	}
	fieldVal, ok := obj[name]
	return ok && isCompleteOpenAPIExample(child, fieldVal)
}

func completeOpenAPIArrayExample(actual *expr.Array, val any) bool {
	items, ok := val.([]any)
	if !ok {
		return true
	}
	for _, item := range items {
		if !isCompleteOpenAPIExample(actual.ElemType, item) {
			return false
		}
	}
	return true
}

func completeOpenAPIMapExample(actual *expr.Map, val any) bool {
	obj, ok := val.(map[string]any)
	if !ok {
		return true
	}
	for _, item := range obj {
		if !isCompleteOpenAPIExample(actual.ElemType, item) {
			return false
		}
	}
	return true
}

func completeOpenAPIUnionExample(actual *expr.Union, val any) bool {
	example, ok := val.(map[string]any)
	if !ok {
		return false
	}
	tag, rawValue, ok := openAPIUnionTagAndValue(actual, example)
	if !ok {
		return false
	}
	for _, branch := range actual.Values {
		if branch != nil && branch.Attribute != nil && expr.UnionVariantTag(branch) == tag {
			return isCompleteOpenAPIExample(branch.Attribute, rawValue)
		}
	}
	return false
}

func openAPIUnionTagAndValue(actual *expr.Union, example map[string]any) (string, any, bool) {
	rawTag, ok := example[actual.GetTypeKey()]
	if !ok {
		return "", nil, false
	}
	tag, ok := rawTag.(string)
	if !ok || tag == "" {
		return "", nil, false
	}
	rawValue, ok := example[actual.GetValueKey()]
	if !ok {
		return "", nil, false
	}
	return tag, rawValue, true
}

func normalizeOpenAPIExample(val any) any {
	switch actual := val.(type) {
	case []byte:
		return string(actual)
	case expr.Val:
		out := make(map[string]any, len(actual))
		for key, value := range actual {
			out[key] = normalizeOpenAPIExample(value)
		}
		return out
	case expr.ArrayVal:
		out := make([]any, len(actual))
		for i, value := range actual {
			out[i] = normalizeOpenAPIExample(value)
		}
		return out
	case expr.MapVal:
		out := make(map[string]any, len(actual))
		for key, value := range actual {
			stringKey, ok := key.(string)
			if !ok {
				return val
			}
			out[stringKey] = normalizeOpenAPIExample(value)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(actual))
		for key, value := range actual {
			out[key] = normalizeOpenAPIExample(value)
		}
		return out
	case []any:
		out := make([]any, len(actual))
		for i, value := range actual {
			out[i] = normalizeOpenAPIExample(value)
		}
		return out
	default:
		rv := reflect.ValueOf(val)
		switch rv.Kind() {
		case reflect.Map:
			out := make(map[string]any, rv.Len())
			iter := rv.MapRange()
			for iter.Next() {
				key := iter.Key()
				if key.Kind() != reflect.String {
					return val
				}
				out[key.String()] = normalizeOpenAPIExample(iter.Value().Interface())
			}
			return out
		case reflect.Slice, reflect.Array:
			out := make([]any, rv.Len())
			for i := 0; i < rv.Len(); i++ {
				out[i] = normalizeOpenAPIExample(rv.Index(i).Interface())
			}
			return out
		default:
			return val
		}
	}
}

func isUnionWrapperObjectType(dt expr.DataType) bool {
	return isUnionWrapperObjectTypeSeen(dt, map[string]struct{}{}, map[expr.DataType]struct{}{})
}

func isUnionWrapperObjectTypeSeen(dt expr.DataType, seenUT map[string]struct{}, seenDT map[expr.DataType]struct{}) bool {
	if dt == nil {
		return false
	}
	if _, ok := seenDT[dt]; ok {
		return false
	}
	seenDT[dt] = struct{}{}
	obj, ok := unwrapExampleDataType(dt, seenUT).(*expr.Object)
	if !ok || len(*obj) != 1 {
		return false
	}
	fieldType := (*obj)[0].Attribute.Type
	return isUnionTypeSeen(fieldType, seenUT) || isUnionWrapperObjectTypeSeen(fieldType, seenUT, seenDT)
}

func isUnionType(dt expr.DataType) bool {
	return isUnionTypeSeen(dt, map[string]struct{}{})
}

func isUnionTypeSeen(dt expr.DataType, seen map[string]struct{}) bool {
	_, ok := unwrapExampleDataType(dt, seen).(*expr.Union)
	return ok
}

func unwrapExampleDataType(dt expr.DataType, seen map[string]struct{}) expr.DataType {
	for {
		ut, ok := dt.(expr.UserType)
		if !ok {
			return dt
		}
		id := ut.ID()
		if _, ok := seen[id]; ok {
			return nil
		}
		seen[id] = struct{}{}
		attr := ut.Attribute()
		if attr == nil {
			return nil
		}
		dt = attr.Type
	}
}
