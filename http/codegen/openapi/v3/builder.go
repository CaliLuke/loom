package openapiv3

import (
	"fmt"
	"maps"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/openapi"
	openapiir "github.com/CaliLuke/loom/http/codegen/openapi/internal/ir"
	"github.com/CaliLuke/loom/internal/securityreq"
)

const (
	// JSONSchemaDialect is the JSON Schema dialect advertised by the generated spec.
	JSONSchemaDialect = "https://spec.openapis.org/oas/3.1/dialect/base"
)

var (
	routeIndexReplacementRegExp = regexp.MustCompile(`\((.*){routeIndex}\)`)
	operationIDSeparatorRegExp  = regexp.MustCompile(`_+`)
)

const (
	defaultOperationIDFormat = "{service}.{method}(.{routeIndex})"
)

// New returns the OpenAPI v3 specification for the given API. It returns nil
// if the design does not define HTTP endpoints or configures an unsupported
// openapi:version value. Callers that need the validation error should evaluate
// the design before calling New or use Files.
func New(root *expr.RootExpr) *OpenAPI {
	target := openAPIVersion32
	if root != nil && root.API != nil {
		configured, err := targetOpenAPIVersion(root.API.Meta)
		if err != nil {
			return nil
		}
		target = configured
	}
	return newForVersion(root, target)
}

func newForVersion(root *expr.RootExpr, target openAPIVersion) *OpenAPI {
	if root == nil || root.API == nil || root.API.HTTP == nil || len(root.API.HTTP.Services) == 0 {
		// No HTTP transport
		return nil
	}

	disableOpenAPIExamples(root.API)
	spec := buildDocument(root)
	renderOpenAPI(root, spec, target)
	return spec
}

// buildInfo builds the OpenAPI Info object.
func buildInfo(api *expr.APIExpr) *Info {
	title := api.Title
	if title == "" {
		title = "Loom API" // cannot be empty as per OpenAPI spec
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
func buildComponents(root *expr.RootExpr, types map[string]*openapi.Schema, reusable reusableComponents) *Components {
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
		Parameters:      reusable.Parameters,
		Headers:         reusable.Headers,
		RequestBodies:   reusable.RequestBodies,
		Responses:       reusable.Responses,
		Examples:        reusable.Examples,
	}
}

// buildPaths builds the OpenAPI Paths map with key as the HTTP path string and
// the value as the corresponding PathItem object.
func buildPaths(h *expr.HTTPExpr, doc *openapiir.Document, api *expr.APIExpr) map[string]*PathItem {
	var paths = make(map[string]*PathItem)
	for _, svc := range h.Services {
		if !openapi.MustGenerate(svc.Meta) || !openapi.MustGenerate(svc.ServiceExpr.Meta) {
			continue
		}
		buildServiceEndpointPaths(paths, doc, svc, openapi.ExtensionsFromExpr(svc.Meta))
		buildServiceFileServerPaths(paths, api, svc)
	}
	return paths
}

// buildOperation builds the OpenAPI Operation object for the given path.
func buildOperation(key string, r *expr.RouteExpr, bodies *EndpointBodies, rand *expr.ExampleGenerator, meta expr.MetaExpr) *Operation {
	closeObjects := openapi.ClosedObjectModeFromExpr(meta)
	operationIR := openapiir.BuildRouteOperation(r, key, endpointBodiesToIR(bodies), rand, meta, closeObjects)
	return buildOperationFromIR(operationIR)
}

func buildOperationFromIR(operationIR *openapiir.Operation) *Operation {
	if operationIR == nil {
		operationIR = &openapiir.Operation{}
	}
	return &Operation{
		Tags:         append([]string(nil), operationIR.Tags...),
		Summary:      operationIR.Summary,
		Description:  operationIR.Description,
		OperationID:  operationIR.OperationID,
		Parameters:   parametersFromIR(operationIR.Parameters),
		RequestBody:  requestBodyRefFromIR(operationIR.RequestBody),
		Responses:    responsesFromIR(operationIR.Responses),
		Security:     cloneOperationSecurity(operationIR.Security),
		Deprecated:   operationIR.Deprecated,
		ExternalDocs: externalDocsFromIR(operationIR.ExternalDocs),
		Extensions:   cloneStringAnyMap(operationIR.Extensions),
	}
}

func irOperation(doc *openapiir.Document, path, method string) *openapiir.Operation {
	if doc == nil || doc.Paths == nil {
		return nil
	}
	pathItem := doc.Paths[path]
	if pathItem == nil || pathItem.Operations == nil {
		return nil
	}
	return pathItem.Operations[method]
}

// buildFileServerOperation builds the OpenAPI Operation object for the given file server.
func buildFileServerOperation(key string, fs *expr.HTTPFileServerExpr, api *expr.APIExpr) *Operation {
	wildcards := expr.ExtractHTTPWildcards(key)
	svc := fs.Service

	return &Operation{
		OperationID:  parseOperationIDTemplate(fileServerOperationIDFormat(api, svc, fs), svc.Name(), key, 0),
		Description:  fs.Description,
		Summary:      fileServerSummary(fs),
		Parameters:   fileServerParameters(wildcards),
		Responses:    fileServerResponses(wildcards),
		Tags:         operationTagNames(fs.Meta, svc.Meta, svc.Name()),
		Security:     securityreq.OpenAPI(securityreq.Effective(api.Requirements, api.SessionAuths)),
		Deprecated:   false,
		ExternalDocs: openapi.DocsFromExpr(fs.Docs, fs.Meta),
		Extensions:   openapi.ExtensionsFromExpr(fs.Meta),
	}
}

func buildServiceEndpointPaths(paths map[string]*PathItem, doc *openapiir.Document, svc *expr.HTTPServiceExpr, exts map[string]any) {
	corsExt := corsExtension(effectiveCORS(svc))
	for _, endpoint := range svc.HTTPEndpoints {
		if !openapi.MustGenerate(endpoint.Meta) || !openapi.MustGenerate(endpoint.MethodExpr.Meta) {
			continue
		}
		for _, route := range endpoint.Routes {
			for _, key := range route.FullPaths() {
				normalizedKey := normalizeOpenAPIPath(key)
				assignPathOperation(paths, normalizedKey, route.Method, buildOperationFromIR(irOperation(doc, normalizedKey, route.Method)))
				assignPathExtensions(paths[normalizedKey], route.Endpoint.Meta, exts)
				assignCORSExtension(paths[normalizedKey], corsExt)
			}
		}
	}
}

func effectiveCORS(svc *expr.HTTPServiceExpr) *expr.HTTPCORSExpr {
	if svc.CORS != nil {
		return svc.CORS
	}
	return svc.Root.CORS
}

func corsExtension(cors *expr.HTTPCORSExpr) map[string]any {
	if cors == nil {
		return nil
	}
	if cors.Runtime {
		return map[string]any{"runtime": true}
	}
	origins := make([]map[string]any, 0, len(cors.Origins))
	for _, origin := range cors.Origins {
		item := map[string]any{"origin": origin.Pattern}
		if origin.Regex {
			item["regex"] = true
		}
		if len(origin.Methods) > 0 {
			item["methods"] = append([]string(nil), origin.Methods...)
		}
		if len(origin.Headers) > 0 {
			item["headers"] = append([]string(nil), origin.Headers...)
		}
		if len(origin.Expose) > 0 {
			item["expose"] = append([]string(nil), origin.Expose...)
		}
		if origin.MaxAge > 0 {
			item["maxAge"] = origin.MaxAge
		}
		if origin.Credentials {
			item["credentials"] = true
		}
		origins = append(origins, item)
	}
	return map[string]any{"origins": origins}
}

func assignCORSExtension(path *PathItem, cors map[string]any) {
	if path == nil || len(cors) == 0 {
		return
	}
	if path.Extensions == nil {
		path.Extensions = make(map[string]any)
	}
	path.Extensions["x-loom-cors"] = cors
}

func buildServiceFileServerPaths(paths map[string]*PathItem, api *expr.APIExpr, svc *expr.HTTPServiceExpr) {
	for _, fileServer := range svc.FileServers {
		if !openapi.MustGenerate(fileServer.Meta) || !openapi.MustGenerate(fileServer.Service.Meta) {
			continue
		}
		for _, key := range fileServer.RequestPaths {
			normalizedKey := normalizeOpenAPIPath(key)
			assignPathOperation(paths, normalizedKey, "GET", buildFileServerOperation(normalizedKey, fileServer, api))
		}
	}
}

func normalizeOpenAPIPath(key string) string {
	return expr.HTTPWildcardRegex.ReplaceAllString(key, "/{$1}")
}

func assignPathOperation(paths map[string]*PathItem, key, method string, operation *Operation) {
	path := paths[key]
	if path == nil {
		path = new(PathItem)
		paths[key] = path
	}
	switch method {
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
	case "TRACE":
		path.Trace = operation
	case "QUERY":
		path.Query = operation
	default:
		if path.AdditionalOperations == nil {
			path.AdditionalOperations = make(map[string]*Operation)
		}
		path.AdditionalOperations[method] = operation
	}
}

func assignPathExtensions(path *PathItem, endpointMeta expr.MetaExpr, serviceExts map[string]any) {
	path.Extensions = openapi.ExtensionsFromExpr(endpointMeta)
	if len(serviceExts) > 0 {
		path.Extensions = make(map[string]any)
		maps.Copy(path.Extensions, serviceExts)
	}
}

func fileServerParameters(wildcards []string) []*ParameterRef {
	if len(wildcards) == 0 {
		return nil
	}
	pref := ParameterRef{
		Value: &Parameter{
			Name:        wildcards[0],
			Description: "Relative file path",
			In:          "path",
			Required:    true,
			Schema: &openapi.Schema{
				Type: openapi.String,
			},
		},
	}
	return []*ParameterRef{&pref}
}

func fileServerResponses(wildcards []string) map[string]*ResponseRef {
	desc200 := "File downloaded"
	responses := map[string]*ResponseRef{
		"200": &ResponseRef{
			Value: &Response{Description: &desc200},
		},
	}
	if len(wildcards) > 0 {
		desc404 := "File not found"
		responses["404"] = &ResponseRef{
			Value: &Response{Description: &desc404},
		}
	}
	return responses
}

func fileServerSummary(fs *expr.HTTPFileServerExpr) string {
	summary := fmt.Sprintf("Download %s", fs.FilePath)
	if override := metaFirst(fs.Meta, "openapi:summary"); override != "" {
		return override
	}
	return summary
}

func fileServerOperationIDFormat(api *expr.APIExpr, svc *expr.HTTPServiceExpr, fs *expr.HTTPFileServerExpr) string {
	operationIDFormat := defaultOperationIDFormat
	for _, meta := range []expr.MetaExpr{api.Meta, svc.Meta, fs.Meta} {
		if override := metaFirst(meta, "openapi:operationId"); override != "" {
			operationIDFormat = override
		}
	}
	return operationIDFormat
}

func metaFirst(meta expr.MetaExpr, key string) string {
	values := meta[key]
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func cloneOperationSecurity(requirements []map[string][]string) []map[string][]string {
	if requirements == nil {
		return nil
	}
	if len(requirements) == 0 {
		return []map[string][]string{}
	}
	cloned := make([]map[string][]string, len(requirements))
	for i, requirement := range requirements {
		current := make(map[string][]string, len(requirement))
		for name, scopes := range requirement {
			current[name] = append(make([]string, 0, len(scopes)), scopes...)
		}
		cloned[i] = current
	}
	return cloned
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
				Name:        svr.Name,
				URL:         string(uExpr),
				Description: svr.Description,
				Variables:   serverVariable,
			}
			svrs = append(svrs, server)
		}
	}
	return svrs
}

// buildSecurityScheme builds the OpenAPI SecurityScheme object from the
// top-level security scheme definition.
func buildSecurityScheme(se *expr.SchemeExpr) *SecurityScheme {
	extensions := openapi.ExtensionsFromExpr(se.Meta)
	var scheme *SecurityScheme
	switch se.Kind {
	case expr.BasicAuthKind:
		scheme = &SecurityScheme{
			Type:        "http",
			Scheme:      "basic",
			Description: se.Description,
			Extensions:  extensions,
		}
	case expr.APIKeyKind:
		scheme = &SecurityScheme{
			Type:        "apiKey",
			Description: se.Description,
			In:          se.In,
			Name:        se.Name,
			Extensions:  extensions,
		}
	case expr.JWTKind:
		scheme = &SecurityScheme{
			Type:        "http",
			Scheme:      "bearer",
			Description: se.Description,
			Extensions:  extensions,
		}
	case expr.OAuth2Kind:
		scheme = &SecurityScheme{
			Type:        "oauth2",
			Description: se.Description,
			Flows:       buildOAuthFlows(se),
			Extensions:  extensions,
		}
	}
	if scheme != nil {
		scheme.OAuth2MetadataURL = metaFirst(se.Meta, "openapi:oauth2MetadataUrl")
		scheme.Deprecated = metaBool(se.Meta, "openapi:deprecated")
	}
	return scheme
}

func buildOAuthFlows(se *expr.SchemeExpr) *OAuthFlows {
	scopes := make(map[string]string, len(se.Scopes))
	for _, scope := range se.Scopes {
		scopes[scope.Name] = scope.Description
	}
	flows := new(OAuthFlows)
	for _, flow := range se.Flows {
		assignOAuthFlow(flows, flow, scopes)
	}
	return flows
}

func assignOAuthFlow(flows *OAuthFlows, flow *expr.FlowExpr, scopes map[string]string) {
	value := &OAuthFlow{
		AuthorizationURL:       flow.AuthorizationURL,
		TokenURL:               flow.TokenURL,
		RefreshURL:             flow.RefreshURL,
		DeviceAuthorizationURL: flow.DeviceAuthorizationURL,
		Scopes:                 scopes,
	}
	switch flow.Kind {
	case expr.AuthorizationCodeFlowKind:
		flows.AuthorizationCode = value
	case expr.ClientCredentialsFlowKind:
		flows.ClientCredentials = value
	case expr.ImplicitFlowKind:
		flows.Implicit = value
	case expr.PasswordFlowKind:
		flows.Password = value
	case expr.DeviceAuthorizationFlowKind:
		flows.DeviceAuthorization = value
	}
}

func metaBool(meta expr.MetaExpr, key string) bool {
	value, ok := meta.Last(key)
	return ok && value != "false"
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
