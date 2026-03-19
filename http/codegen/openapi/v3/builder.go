package openapiv3

import (
	"fmt"
	"maps"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"goa.design/goa/v3/codegen"
	"goa.design/goa/v3/expr"
	"goa.design/goa/v3/http/codegen/openapi"
)

const (
	// OpenAPIVersion is the OpenAPI specification version targeted by this package.
	OpenAPIVersion = "3.1.1"
	// JSONSchemaDialect is the JSON Schema dialect advertised by the generated spec.
	JSONSchemaDialect = "https://json-schema.org/draft/2020-12/schema"
)

var (
	routeIndexReplacementRegExp = regexp.MustCompile(`\((.*){routeIndex}\)`)
	operationIDSeparatorRegExp  = regexp.MustCompile(`_+`)
)

const (
	defaultOperationIDFormat = "{service}.{method}(.{routeIndex})"
)

// New returns the OpenAPI v3 specification for the given API.
// It returns nil if the design does not define HTTP endpoints.
func New(root *expr.RootExpr) *OpenAPI {
	if root == nil || root.API == nil || root.API.HTTP == nil || len(root.API.HTTP.Services) == 0 {
		// No HTTP transport
		return nil
	}

	m, ok := root.API.Meta.Last("openapi:example")
	if ok && m == "false" {
		root.API.ExampleGenerator.Randomizer = nil
	}

	var (
		bodies, types = buildBodyTypes(root.API, root.Types, root.ResultTypes)

		info     = buildInfo(root.API)
		servers  = buildServers(root.API.Servers)
		paths    = buildPaths(root.API.HTTP, bodies, root.API)
		comps    = buildComponents(root, pruneUnusedComponentSchemas(paths, types))
		security = buildSecurityRequirements(effectiveRequirements(root.API.Requirements, root.API.SessionAuths))
		tags     = buildTags(root.API)
	)

	return &OpenAPI{
		OpenAPI:           OpenAPIVersion,
		Info:              info,
		JSONSchemaDialect: JSONSchemaDialect,
		Components:        comps,
		Paths:             paths,
		Servers:           servers,
		Security:          security,
		Tags:              tags,
	}
}

func pruneUnusedComponentSchemas(paths map[string]*PathItem, schemas map[string]*openapi.Schema) map[string]*openapi.Schema {
	if len(schemas) == 0 {
		return schemas
	}

	reachable := make(map[string]struct{}, len(schemas))
	queue := make([]string, 0, len(schemas))
	enqueue := func(ref string) {
		name, ok := schemaNameFromRef(ref)
		if !ok {
			return
		}
		if _, seen := reachable[name]; seen {
			return
		}
		reachable[name] = struct{}{}
		queue = append(queue, name)
	}

	for _, pathItem := range paths {
		collectPathItemSchemaRefs(pathItem, enqueue)
	}

	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		schema := schemas[name]
		if schema == nil {
			continue
		}
		collectSchemaRefs(schema, enqueue)
	}

	pruned := make(map[string]*openapi.Schema, len(reachable))
	for name := range reachable {
		if schema := schemas[name]; schema != nil {
			pruned[name] = schema
		}
	}
	return pruned
}

func schemaNameFromRef(ref string) (string, bool) {
	const prefix = "#/components/schemas/"
	if !strings.HasPrefix(ref, prefix) {
		return "", false
	}
	name := strings.TrimPrefix(ref, prefix)
	if name == "" {
		return "", false
	}
	return name, true
}

func collectPathItemSchemaRefs(pathItem *PathItem, addRef func(string)) {
	if pathItem == nil {
		return
	}
	ops := []*Operation{
		pathItem.Get,
		pathItem.Put,
		pathItem.Post,
		pathItem.Delete,
		pathItem.Options,
		pathItem.Head,
		pathItem.Patch,
	}
	for _, op := range ops {
		collectOperationSchemaRefs(op, addRef)
	}
}

func collectOperationSchemaRefs(op *Operation, addRef func(string)) {
	if op == nil {
		return
	}
	for _, param := range op.Parameters {
		if param != nil && param.Value != nil {
			collectSchemaRefs(param.Value.Schema, addRef)
		}
	}
	if op.RequestBody != nil && op.RequestBody.Value != nil {
		for _, content := range op.RequestBody.Value.Content {
			if content != nil {
				collectSchemaRefs(content.Schema, addRef)
			}
		}
	}
	for _, response := range op.Responses {
		if response == nil || response.Value == nil {
			continue
		}
		for _, header := range response.Value.Headers {
			if header != nil && header.Value != nil {
				collectSchemaRefs(header.Value.Schema, addRef)
			}
		}
		for _, content := range response.Value.Content {
			if content != nil {
				collectSchemaRefs(content.Schema, addRef)
			}
		}
	}
}

func collectSchemaRefs(schema *openapi.Schema, addRef func(string)) {
	if schema == nil {
		return
	}
	if schema.Ref != "" {
		addRef(schema.Ref)
		return
	}
	collectSchemaRefs(schema.Items, addRef)
	for _, prop := range schema.Properties {
		collectSchemaRefs(prop, addRef)
	}
	for _, def := range schema.Defs {
		collectSchemaRefs(def, addRef)
	}
	for _, item := range schema.AnyOf {
		collectSchemaRefs(item, addRef)
	}
	for _, item := range schema.OneOf {
		collectSchemaRefs(item, addRef)
	}
	if nested, ok := schema.AdditionalProperties.(*openapi.Schema); ok {
		collectSchemaRefs(nested, addRef)
	}
	if nested, ok := schema.UnevaluatedProperties.(*openapi.Schema); ok {
		collectSchemaRefs(nested, addRef)
	}
	if schema.Media != nil {
		// no schema refs
	}
	for _, link := range schema.Links {
		if link == nil {
			continue
		}
		collectSchemaRefs(link.Schema, addRef)
		collectSchemaRefs(link.TargetSchema, addRef)
	}
}

// buildInfo builds the OpenAPI Info object.
func buildInfo(api *expr.APIExpr) *Info {
	title := api.Title
	if title == "" {
		title = "Goa API" // cannot be empty as per OpenAPI spec
	}
	info := &Info{
		Title:          title,
		Description:    api.Description,
		TermsOfService: api.TermsOfService,
		Version:        api.Version,
		Extensions:     openapi.ExtensionsFromExpr(api.Meta),
	}
	if c := api.Contact; c != nil {
		info.Contact = &Contact{
			Name:  c.Name,
			Email: c.Email,
			URL:   c.URL,
		}
	}
	if l := api.License; l != nil {
		info.License = &License{
			Name: l.Name,
			URL:  l.URL,
		}
	}
	return info
}

// buildComponents builds the OpenAPI Components object.
func buildComponents(root *expr.RootExpr, types map[string]*openapi.Schema) *Components {
	var schemesRef map[string]*SecuritySchemeRef
	{
		schemesRef = make(map[string]*SecuritySchemeRef)
		for _, s := range root.API.HTTP.Services {
			for _, e := range s.HTTPEndpoints {
				for _, r := range e.Requirements {
					for _, sch := range r.Schemes {
						schemesRef[sch.Hash()] = &SecuritySchemeRef{
							Value: buildSecurityScheme(sch),
						}
					}
				}
			}
		}
	}
	return &Components{
		SecuritySchemes: schemesRef,
		Schemas:         types,
	}
}

// buildPaths builds the OpenAPI Paths map with key as the HTTP path string and
// the value as the corresponding PathItem object.
func buildPaths(h *expr.HTTPExpr, bodies map[string]map[string]*EndpointBodies, api *expr.APIExpr) map[string]*PathItem {
	var paths = make(map[string]*PathItem)
	for _, svc := range h.Services {
		if !openapi.MustGenerate(svc.Meta) || !openapi.MustGenerate(svc.ServiceExpr.Meta) {
			continue
		}
		exts := openapi.ExtensionsFromExpr(svc.Meta)
		sbod := bodies[svc.Name()]

		// endpoints
		for _, e := range svc.HTTPEndpoints {
			if !openapi.MustGenerate(e.Meta) || !openapi.MustGenerate(e.MethodExpr.Meta) {
				continue
			}
			for _, r := range e.Routes {
				for _, key := range r.FullPaths() {
					// Remove any wildcards that is defined in path as a workaround to
					// https://github.com/OAI/OpenAPI-Specification/issues/291
					key = expr.HTTPWildcardRegex.ReplaceAllString(key, "/{$1}")
					operation := buildOperation(key, r, sbod[e.Name()], api.ExampleGenerator, api.Meta)
					path, ok := paths[key]
					if !ok {
						path = new(PathItem)
						paths[key] = path
					}
					switch r.Method {
					case "GET":
						path.Get = operation
					case "PUT":
						path.Put = operation
					case "POST":
						path.Post = operation
					case "DELETE":
						path.Delete = operation
					case "OPTIONS":
						path.Options = operation
					case "HEAD":
						path.Head = operation
					case "PATCH":
						path.Patch = operation
					}
					path.Extensions = openapi.ExtensionsFromExpr(r.Endpoint.Meta)
					if len(exts) > 0 {
						path.Extensions = make(map[string]any)
						maps.Copy(path.Extensions, exts)
					}
				}
			}
		}

		// file servers
		for _, f := range svc.FileServers {
			if !openapi.MustGenerate(f.Meta) || !openapi.MustGenerate(f.Service.Meta) {
				continue
			}

			for _, key := range f.RequestPaths {
				// Replace wildcards in the path to OpenAPI path parameter form
				// e.g. "/ui/{*filepath}" -> "/ui/{filepath}"
				key = expr.HTTPWildcardRegex.ReplaceAllString(key, "/{$1}")
				operation := buildFileServerOperation(key, f, api)
				path, ok := paths[key]
				if !ok {
					path = new(PathItem)
					paths[key] = path
				}
				path.Get = operation
			}
		}
	}
	return paths
}

// buildOperation builds the OpenAPI Operation object for the given path.
func buildOperation(key string, r *expr.RouteExpr, bodies *EndpointBodies, rand *expr.ExampleGenerator, meta expr.MetaExpr) *Operation {
	e := r.Endpoint
	m := e.MethodExpr
	svc := e.Service
	closeObjects := openapi.ClosedObjectModeFromExpr(meta)

	// OpenAPI summary
	var summary string
	setSummary := func(meta expr.MetaExpr) {
		for n, mdata := range meta {
			if n == "openapi:summary" && len(mdata) > 0 {
				if mdata[0] == "{path}" {
					summary = r.Path
				} else {
					summary = mdata[0]
				}
			}
		}
	}

	summary = fmt.Sprintf("%s %s", e.Name(), svc.Name())
	setSummary(meta)
	setSummary(svc.ServiceExpr.Meta)
	setSummary(e.Meta)
	setSummary(m.Meta)

	// OpenAPI operationId
	var operationIDFormat string
	setOperationIDFormat := func(meta expr.MetaExpr) {
		for n, mdata := range meta {
			if (n == "openapi:operationId") && len(mdata) > 0 {
				operationIDFormat = mdata[0]
			}
		}
	}

	operationIDFormat = defaultOperationIDFormat
	setOperationIDFormat(meta)
	setOperationIDFormat(m.Service.Meta)
	setOperationIDFormat(e.Meta)
	setOperationIDFormat(m.Meta)

	// request body
	var requestBody *RequestBodyRef
	if e.Body.Type != expr.Empty {
		ct := "application/json" // TBD: need a way to specify method media type in design...
		if e.MultipartRequest {
			ct = "multipart/form-data"
		} else if e.FormRequest {
			ct = "application/x-www-form-urlencoded"
		}
		mt := &MediaType{Schema: bodies.RequestBody}
		initExamples(mt, e.Body, rand)
		requestBody = &RequestBodyRef{Value: &RequestBody{
			Description: e.Body.Description,
			Required:    e.Body.Type != expr.Empty && !e.OptionalRequestBody,
			Content:     map[string]*MediaType{ct: mt},
			Extensions:  openapi.ExtensionsFromExpr(e.Body.Meta),
		}}
	}

	// parameters
	var params []*ParameterRef
	{
		ps := paramsFromPath(e, key, rand, closeObjects)
		ps = append(ps, paramsFromHeadersAndCookies(e, rand, closeObjects)...)
		if e.MapQueryParams != nil {
			name := *e.MapQueryParams
			if name == "" {
				name = "payload"
			}
			ps = append(ps, &Parameter{
				Name:        name,
				Description: "Query parameters",
				In:          "query",
				Required:    name == "payload" || e.MethodExpr.Payload.IsRequired(name),
				Schema: &openapi.Schema{
					Type:                 "object",
					AdditionalProperties: true,
				},
				Style: "deepObject",
			})
		}
		params = make([]*ParameterRef, len(ps))
		for i, p := range ps {
			params[i] = &ParameterRef{Value: p}
		}
	}

	// responses
	responses := make(map[string]*ResponseRef, len(e.Responses))
	for _, r := range e.Responses {
		if e.MethodExpr.IsStreaming() {
			// A streaming endpoint allows at most one successful response
			// definition. So it is okay to change the first successful
			// response to a HTTP 101 response for openapi docs.
			if _, ok := responses[strconv.Itoa(expr.StatusSwitchingProtocols)]; !ok {
				b := bodies.ResponseBodies[r.StatusCode]
				delete(bodies.ResponseBodies, r.StatusCode)
				r = r.Dup()
				r.StatusCode = expr.StatusSwitchingProtocols
				bodies.ResponseBodies[r.StatusCode] = b
			}
		}
		resp := responseFromExpr(r, bodies.ResponseBodies, rand, closeObjects)
		responses[strconv.Itoa(r.StatusCode)] = &ResponseRef{Value: resp}
	}
	for _, er := range e.HTTPErrors {
		if er.Description != "" && er.Response.Description == "" {
			er.Response.Description = er.Description
		}
		resp := responseFromExpr(er.Response, bodies.ResponseBodies, rand, closeObjects)
		desc := er.Name
		if resp.Description != nil {
			desc += ": " + *resp.Description
		}
		desc = appendErrorRemedyDescription(desc, er)
		resp.Description = &desc
		if er.Type == expr.ErrorResult && len(er.Response.Body.ExtractUserExamples()) == 0 {
			for _, content := range resp.Content {
				content.Example = nil
			}
		}
		responses[strconv.Itoa(er.Response.StatusCode)] = &ResponseRef{Value: resp}
	}

	// tag names
	var tagNames []string
	tagNames = openapi.TagNamesFromExpr(e.Meta)
	if len(tagNames) == 0 {
		// By default tag with service name
		tagNames = []string{e.Service.Name()}
	}

	// An endpoint can have multiple routes, so we need to be able to build a unique
	// operationId for each route.
	var routeIndex int
	for i, rt := range e.Routes {
		if rt == r {
			routeIndex = i
			break
		}
	}

	// An endpoint may be marked as deprecated. if the openapi:deprecated tag is present, we populate it to true
	_, deprecated := e.Meta.Last("openapi:deprecated")
	return &Operation{
		Tags:         tagNames,
		Summary:      summary,
		Description:  e.Description(),
		OperationID:  parseOperationIDTemplate(operationIDFormat, svc.Name(), e.Name(), routeIndex),
		Parameters:   params,
		RequestBody:  requestBody,
		Responses:    responses,
		Security:     buildOperationSecurity(e),
		Deprecated:   deprecated,
		ExternalDocs: openapi.DocsFromExpr(m.Docs, m.Meta),
		Extensions:   openapi.ExtensionsFromExpr(m.Meta),
	}
}

func buildOperationSecurity(e *expr.HTTPEndpointExpr) []map[string][]string {
	if e == nil || e.MethodExpr == nil {
		return nil
	}
	if _, ok := e.MethodExpr.Meta["security:no"]; ok {
		return []map[string][]string{}
	}
	if len(e.Requirements) == 0 {
		return nil
	}
	return buildSecurityRequirements(e.Requirements)
}

func appendErrorRemedyDescription(desc string, er *expr.HTTPErrorExpr) string {
	if er == nil || er.ErrorExpr == nil || er.ErrorExpr.Remedy == nil {
		return desc
	}
	parts := []string{desc}
	if er.ErrorExpr.Remedy.Code != "" {
		parts = append(parts, "Remedy code: "+er.ErrorExpr.Remedy.Code+".")
	}
	if er.ErrorExpr.Remedy.SafeMessage != "" {
		parts = append(parts, "Safe message: "+trimSentence(er.ErrorExpr.Remedy.SafeMessage)+".")
	}
	if er.ErrorExpr.Remedy.RetryHint != "" {
		parts = append(parts, "Retry hint: "+trimSentence(er.ErrorExpr.Remedy.RetryHint)+".")
	}
	return strings.Join(parts, " ")
}

func trimSentence(text string) string {
	return strings.TrimRight(text, ". ")
}

// buildFileServerOperation builds the OpenAPI Operation object for the given file server.
func buildFileServerOperation(key string, fs *expr.HTTPFileServerExpr, api *expr.APIExpr) *Operation {
	wildcards := expr.ExtractHTTPWildcards(key)
	svc := fs.Service

	// parameters
	var params []*ParameterRef
	if len(wildcards) > 0 {
		pref := ParameterRef{
			Value: &Parameter{
				// Use the literal wildcard (including leading '*') as name to match path if needed
				// Note: HTTPWildcardRegex already strips '*' in ExtractHTTPWildcards; however
				// the path key has been normalized to "/{name}" so the correct parameter name
				// is the bare wildcard identifier.
				Name:        wildcards[0],
				Description: "Relative file path",
				In:          "path",
				Required:    true,
				Schema: &openapi.Schema{ // string schema makes validators happy
					Type: openapi.String,
				},
			},
		}
		params = []*ParameterRef{&pref}
	}

	// responses
	var responses map[string]*ResponseRef
	{
		desc200 := "File downloaded"
		rref := ResponseRef{
			Value: &Response{
				Description: &desc200,
			},
		}
		responses = map[string]*ResponseRef{
			"200": &rref,
		}
		if len(wildcards) > 0 {
			desc404 := "File not found"
			responses["404"] = &ResponseRef{
				Value: &Response{
					Description: &desc404,
				},
			}
		}
	}

	// OpenAPI summary
	var summary string
	summary = fmt.Sprintf("Download %s", fs.FilePath)
	for n, mdata := range fs.Meta {
		if n == "openapi:summary" && len(mdata) > 0 {
			summary = mdata[0]
		}
	}

	// OpenAPI operationId
	var operationIDFormat string
	setOperationIDFormat := func(meta expr.MetaExpr) {
		for n, mdata := range meta {
			if n == "openapi:operationId" && len(mdata) > 0 {
				operationIDFormat = mdata[0]
			}
		}
	}

	operationIDFormat = defaultOperationIDFormat
	setOperationIDFormat(api.Meta)
	setOperationIDFormat(svc.Meta)
	setOperationIDFormat(fs.Meta)

	// tag names
	var tagNames []string
	tagNames = openapi.TagNamesFromExpr(fs.Meta)
	if len(tagNames) == 0 {
		// By default tag with service name
		tagNames = []string{svc.Name()}
	}

	return &Operation{
		OperationID:  parseOperationIDTemplate(operationIDFormat, svc.Name(), key, 0),
		Description:  fs.Description,
		Summary:      summary,
		Parameters:   params,
		Responses:    responses,
		Tags:         tagNames,
		Security:     buildSecurityRequirements(effectiveRequirements(api.Requirements, api.SessionAuths)),
		Deprecated:   false,
		ExternalDocs: openapi.DocsFromExpr(fs.Docs, fs.Meta),
		Extensions:   openapi.ExtensionsFromExpr(fs.Meta),
	}
}

func effectiveRequirements(requirements []*expr.SecurityExpr, sessionAuths []*expr.SessionAuthExpr) []*expr.SecurityExpr {
	merged := make([]*expr.SecurityExpr, len(requirements))
	copy(merged, requirements)
	for _, sessionAuth := range sessionAuths {
		for _, transport := range sessionAuth.Transports {
			if transport == nil || transport.Scheme == nil {
				continue
			}
			req := &expr.SecurityExpr{
				Schemes: []*expr.SchemeExpr{expr.DupScheme(transport.Scheme)},
			}
			if !containsRequirement(merged, req) {
				merged = append(merged, req)
			}
		}
	}
	return merged
}

func containsRequirement(requirements []*expr.SecurityExpr, candidate *expr.SecurityExpr) bool {
	for _, req := range requirements {
		if len(req.Scopes) != len(candidate.Scopes) || len(req.Schemes) != len(candidate.Schemes) {
			continue
		}
		matched := true
		for i, scope := range req.Scopes {
			if candidate.Scopes[i] != scope {
				matched = false
				break
			}
		}
		if !matched {
			continue
		}
		for i, scheme := range req.Schemes {
			other := candidate.Schemes[i]
			if scheme.Kind != other.Kind || scheme.SchemeName != other.SchemeName || scheme.In != other.In || scheme.Name != other.Name {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}

func parseOperationIDTemplate(template, service, method string, routeIndex int) string {
	// Early return if no replacement is needed for the template.
	if !strings.Contains(template, "{") && routeIndex == 0 {
		return template
	}

	// The template replacer
	repl := strings.NewReplacer(
		"{service}", canonicalOperationIDComponent(service),
		"{method}", canonicalOperationIDComponent(method),
	)

	operationID := repl.Replace(template)

	if routeIndex == 0 {
		return routeIndexReplacementRegExp.ReplaceAllString(operationID, "")
	}

	// If the routeIndex is greater than 0, we need to add the routeIndex to the operationId.
	if sep := routeIndexReplacementRegExp.FindStringSubmatch(template); sep != nil {
		return routeIndexReplacementRegExp.ReplaceAllString(operationID, fmt.Sprintf("%s%d", sep[1], routeIndex))
	}

	// Fallback in the event that the operationId doesn't contain the routeIndex placeholder.
	return fmt.Sprintf("%s#%d", operationID, routeIndex)
}

func canonicalOperationIDComponent(name string) string {
	component := codegen.SnakeCase(name)
	var b strings.Builder
	b.Grow(len(component))
	for _, r := range component {
		switch {
		case unicode.IsLower(r), unicode.IsDigit(r):
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	component = operationIDSeparatorRegExp.ReplaceAllString(b.String(), "_")
	component = strings.Trim(component, "_")
	if component == "" {
		return "operation"
	}
	return component
}

// buildServers builds the OpenAPI Server objects from the given server
// expressions.
func buildServers(servers []*expr.ServerExpr) []*Server {
	var svrs []*Server
	for _, svr := range servers {
		if !openapi.MustGenerate(svr.Meta) {
			continue
		}
		var server *Server
		for _, host := range svr.Hosts {
			if !openapi.MustGenerate(host.Meta) {
				continue
			}

			var (
				serverVariable   = make(map[string]*ServerVariable)
				defaultValue     any
				validationValues []any
			)

			// Get the first URL expression in the host by default.
			// Host expression must have at least one URI (validations would have failed
			// otherwise).
			uExpr := host.URIs[0]
			// attempt to find the first HTTP/HTTPS URL
			for _, ue := range host.URIs {
				s := ue.Scheme()
				if s == "http" || s == "https" {
					uExpr = ue
					break
				}
			}

			// retrieve host variables
			vars := expr.AsObject(host.Variables.Type)
			for _, v := range *vars {
				defaultValue = v.Attribute.DefaultValue

				if v.Attribute.Validation != nil && len(v.Attribute.Validation.Values) > 0 {
					validationValues = append(validationValues, v.Attribute.Validation.Values...)
					if defaultValue == nil {
						defaultValue = v.Attribute.Validation.Values[0]
					}
				}

				if defaultValue != nil {
					serverVariable[v.Name] = &ServerVariable{
						Enum:        validationValues,
						Default:     defaultValue,
						Description: host.Variables.Description,
					}
				}
			}

			server = &Server{
				URL:         string(uExpr),
				Description: svr.Description,
				Variables:   serverVariable,
			}
			svrs = append(svrs, server)
		}
	}
	return svrs
}

// buildSecurityRequirements builds the OpenAPI security requirements for the
// given security expressions.
func buildSecurityRequirements(reqs []*expr.SecurityExpr) []map[string][]string {
	if len(reqs) == 0 {
		return nil
	}
	srs := make([]map[string][]string, len(reqs))
	for i, req := range reqs {
		sr := make(map[string][]string, len(req.Schemes))
		for _, sch := range req.Schemes {
			scopes := make([]string, 0)
			switch sch.Kind {
			case expr.OAuth2Kind, expr.JWTKind:
				if len(req.Scopes) > 0 {
					scopes = req.Scopes
				}
			}
			sr[sch.Hash()] = scopes
		}
		srs[i] = sr
	}
	return srs
}

// buildSecurityScheme builds the OpenAPI SecurityScheme object from the
// top-level security scheme definition.
func buildSecurityScheme(se *expr.SchemeExpr) *SecurityScheme {
	var scheme *SecurityScheme
	switch se.Kind {
	case expr.BasicAuthKind:
		scheme = &SecurityScheme{
			Type:        "http",
			Scheme:      "basic",
			Description: se.Description,
			Extensions:  openapi.ExtensionsFromExpr(se.Meta),
		}
	case expr.APIKeyKind:
		scheme = &SecurityScheme{
			Type:        "apiKey",
			Description: se.Description,
			In:          se.In,
			Name:        se.Name,
			Extensions:  openapi.ExtensionsFromExpr(se.Meta),
		}
	case expr.JWTKind:
		scheme = &SecurityScheme{
			Type:        "http",
			Scheme:      "bearer",
			Description: se.Description,
			Extensions:  openapi.ExtensionsFromExpr(se.Meta),
		}
	case expr.OAuth2Kind:
		scopes := make(map[string]string, len(se.Scopes))
		for _, scope := range se.Scopes {
			scopes[scope.Name] = scope.Description
		}
		var flows OAuthFlows
		for _, f := range se.Flows {
			switch f.Kind {
			case expr.AuthorizationCodeFlowKind:
				flows.AuthorizationCode = &OAuthFlow{
					AuthorizationURL: f.AuthorizationURL,
					TokenURL:         f.TokenURL,
					RefreshURL:       f.RefreshURL,
					Scopes:           scopes,
				}
			case expr.ClientCredentialsFlowKind:
				flows.ClientCredentials = &OAuthFlow{
					TokenURL:   f.TokenURL,
					RefreshURL: f.RefreshURL,
					Scopes:     scopes,
				}
			case expr.ImplicitFlowKind:
				flows.Implicit = &OAuthFlow{
					AuthorizationURL: f.AuthorizationURL,
					RefreshURL:       f.RefreshURL,
					Scopes:           scopes,
				}
			case expr.PasswordFlowKind:
				flows.Password = &OAuthFlow{
					TokenURL:   f.TokenURL,
					RefreshURL: f.RefreshURL,
					Scopes:     scopes,
				}
			}
		}
		scheme = &SecurityScheme{
			Type:        "oauth2",
			Description: se.Description,
			Flows:       &flows,
			Extensions:  openapi.ExtensionsFromExpr(se.Meta),
		}
	}
	return scheme
}

// buildTags builds the OpenAPI Tag object from the API expression.
func buildTags(api *expr.APIExpr) []*openapi.Tag {
	m := make(map[string]*openapi.Tag)
	for _, t := range openapi.TagsFromExpr(api.Meta) {
		m[t.Name] = t
	}
	for _, s := range api.HTTP.Services {
		if !openapi.MustGenerate(s.Meta) || !openapi.MustGenerate(s.ServiceExpr.Meta) {
			continue
		}
		for _, t := range openapi.TagsFromExpr(s.Meta) {
			m[t.Name] = t
		}
	}

	// sort tag names alphabetically
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	tags := make([]*openapi.Tag, 0, len(keys))
	for _, k := range keys {
		tags = append(tags, m[k])
	}

	if len(tags) == 0 {
		// add service name and description to the tags since we tag every
		// operation with service name when no custom tag is defined
		for _, s := range api.HTTP.Services {
			if !openapi.MustGenerate(s.Meta) || !openapi.MustGenerate(s.ServiceExpr.Meta) {
				continue
			}
			tags = append(tags, &openapi.Tag{
				Name:        s.Name(),
				Description: s.Description(),
			})
		}
	}
	return tags
}
