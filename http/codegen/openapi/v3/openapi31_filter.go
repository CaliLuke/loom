package openapiv3

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/CaliLuke/loom/http/codegen/openapi"
)

// filterOpenAPI31 omits sections whose minimum OpenAPI version is 3.2.
// The shared DSL analysis and document model remain identical for all targets.
func filterOpenAPI31(spec *OpenAPI) []string {
	spec.Self = ""
	filterOpenAPI31Servers(spec.Servers)
	filterOpenAPI31Tags(spec.Tags)
	warnings := filterOpenAPI31Paths(spec.Paths)
	filterOpenAPI31Components(spec.Components)
	return warnings
}

func filterOpenAPI31Servers(servers []*Server) {
	for _, server := range servers {
		if server != nil {
			server.Name = ""
		}
	}
}

func filterOpenAPI31Tags(tags []*openapi.Tag) {
	for _, tag := range tags {
		if tag != nil {
			tag.Summary = ""
			tag.Parent = ""
			tag.Kind = ""
		}
	}
}

func filterOpenAPI31Paths(paths map[string]*PathItem) []string {
	pathNames := make([]string, 0, len(paths))
	for pathName := range paths {
		pathNames = append(pathNames, pathName)
	}
	sort.Strings(pathNames)

	var warnings []string
	for _, pathName := range pathNames {
		path := paths[pathName]
		if path == nil {
			continue
		}
		droppedMethods := droppedOpenAPI31Methods(path)
		path.Query = nil
		path.AdditionalOperations = nil
		path.Connect = nil
		filterParameterCompatibility(path.Parameters)
		for _, operation := range pathItemOperations(path) {
			if operation != nil {
				filterParameterCompatibility(operation.Parameters)
				if operation.RequestBody != nil && operation.RequestBody.Value != nil {
					filterMediaTypeCompatibility(operation.RequestBody.Value.Content)
				}
				for _, response := range operation.Responses {
					if response != nil && response.Value != nil {
						filterResponseCompatibility(response.Value)
						filterMediaTypeCompatibility(response.Value.Content)
					}
				}
			}
		}
		removePath := path.Delete == nil && path.Get == nil && path.Head == nil &&
			path.Options == nil && path.Patch == nil && path.Post == nil && path.Put == nil && path.Trace == nil
		if removePath {
			delete(paths, pathName)
		}
		if len(droppedMethods) > 0 {
			warnings = append(warnings, openAPI31PathWarning(pathName, droppedMethods, removePath))
		}
	}
	return warnings
}

func droppedOpenAPI31Methods(path *PathItem) []string {
	methods := make(map[string]struct{})
	if path.Connect != nil {
		methods["CONNECT"] = struct{}{}
	}
	if path.Query != nil {
		methods["QUERY"] = struct{}{}
	}
	for method, operation := range path.AdditionalOperations {
		if operation != nil {
			methods[method] = struct{}{}
		}
	}
	ordered := make([]string, 0, len(methods))
	for method := range methods {
		ordered = append(ordered, method)
	}
	sort.Strings(ordered)
	return ordered
}

func openAPI31PathWarning(pathName string, methods []string, removePath bool) string {
	noun := "method"
	if len(methods) != 1 {
		noun = "methods"
	}
	warning := fmt.Sprintf(
		"OpenAPI 3.1 omits unsupported %s %s from path %q",
		noun,
		strings.Join(methods, ", "),
		pathName,
	)
	if removePath {
		warning += " and removes the path because no compatible operations remain"
	}
	return warning
}

func filterOpenAPI31Components(components *Components) {
	if components == nil {
		return
	}
	components.MediaTypes = nil
	for _, parameter := range components.Parameters {
		filterParameterCompatibility([]*ParameterRef{parameter})
	}
	for _, schema := range components.Schemas {
		filterSchemaCompatibility(schema, make(map[*openapi.Schema]struct{}))
	}
	for _, header := range components.Headers {
		filterHeaderCompatibility(header)
	}
	for _, example := range components.Examples {
		filterExampleCompatibility(example)
	}
	for _, link := range components.Links {
		filterLinkCompatibility(link)
	}
	for _, scheme := range components.SecuritySchemes {
		filterSecurityCompatibility(scheme)
	}
	for _, body := range components.RequestBodies {
		if body != nil && body.Value != nil {
			filterMediaTypeCompatibility(body.Value.Content)
		}
	}
	for _, response := range components.Responses {
		if response != nil && response.Value != nil {
			filterResponseCompatibility(response.Value)
			filterMediaTypeCompatibility(response.Value.Content)
		}
	}
}

func filterResponseCompatibility(response *Response) {
	response.Summary = ""
	if response.Description == nil {
		description := response.CompatibilityDescription
		response.Description = &description
	}
	for _, header := range response.Headers {
		filterHeaderCompatibility(header)
	}
	for _, link := range response.Links {
		filterLinkCompatibility(link)
	}
}

func filterParameterCompatibility(parameters []*ParameterRef) {
	for _, ref := range parameters {
		if ref == nil || ref.Value == nil {
			continue
		}
		parameter := ref.Value
		if parameter.In != "query" {
			parameter.AllowReserved = false
		}
		if parameter.In == "cookie" && parameter.Style == "cookie" {
			parameter.Style = ""
		}
		filterSchemaCompatibility(parameter.Schema, make(map[*openapi.Schema]struct{}))
		filterMediaTypeCompatibility(parameter.Content)
		for _, example := range parameter.Examples {
			filterExampleCompatibility(example)
		}
	}
}

func filterMediaTypeCompatibility(mediaTypes map[string]*MediaType) {
	for _, mediaType := range mediaTypes {
		if mediaType == nil {
			continue
		}
		mediaType.ItemSchema = nil
		mediaType.PrefixEncoding = nil
		mediaType.ItemEncoding = nil
		filterSchemaCompatibility(mediaType.Schema, make(map[*openapi.Schema]struct{}))
		for _, example := range mediaType.Examples {
			filterExampleCompatibility(example)
		}
		for _, encoding := range mediaType.Encoding {
			filterEncodingCompatibility(encoding)
		}
	}
}

func filterHeaderCompatibility(ref *HeaderRef) {
	if ref == nil || ref.Value == nil {
		return
	}
	ref.Value.AllowReserved = false
	filterSchemaCompatibility(ref.Value.Schema, make(map[*openapi.Schema]struct{}))
	filterMediaTypeCompatibility(ref.Value.Content)
	for _, example := range ref.Value.Examples {
		filterExampleCompatibility(example)
	}
}

func filterExampleCompatibility(ref *ExampleRef) {
	if ref == nil || ref.Value == nil {
		return
	}
	ref.Value.DataValue = nil
	ref.Value.SerializedValue = ""
	ref.Value.Value = ref.Value.CompatibilityValue
}

func filterLinkCompatibility(ref *LinkRef) {
	if ref == nil || ref.Value == nil {
		return
	}
	ref.Value.Server = nil
}

func filterSecurityCompatibility(ref *SecuritySchemeRef) {
	if ref == nil || ref.Value == nil {
		return
	}
	ref.Value.OAuth2MetadataURL = ""
	ref.Value.Deprecated = false
	if ref.Value.Flows != nil {
		ref.Value.Flows.DeviceAuthorization = nil
	}
}

func filterSchemaCompatibility(schema *openapi.Schema, seen map[*openapi.Schema]struct{}) {
	if schema == nil {
		return
	}
	if _, ok := seen[schema]; ok {
		return
	}
	seen[schema] = struct{}{}
	if schema.Discriminator != nil {
		if schema.Discriminator.Optional && !slices.Contains(schema.Required, schema.Discriminator.PropertyName) {
			schema.Required = append(schema.Required, schema.Discriminator.PropertyName)
		}
		schema.Discriminator.DefaultMapping = ""
		schema.Discriminator.Optional = false
	}
	if schema.XML != nil {
		schema.XML.NodeType = ""
	}
	filterSchemaCompatibility(schema.Items, seen)
	filterSchemaCompatibility(schema.ContentSchema, seen)
	for _, child := range schema.Properties {
		filterSchemaCompatibility(child, seen)
	}
	for _, child := range schema.Defs {
		filterSchemaCompatibility(child, seen)
	}
	for _, child := range schema.AnyOf {
		filterSchemaCompatibility(child, seen)
	}
	for _, child := range schema.OneOf {
		filterSchemaCompatibility(child, seen)
	}
	if child, ok := schema.AdditionalProperties.(*openapi.Schema); ok {
		filterSchemaCompatibility(child, seen)
	}
	if child, ok := schema.UnevaluatedProperties.(*openapi.Schema); ok {
		filterSchemaCompatibility(child, seen)
	}
}

func filterEncodingCompatibility(encoding *Encoding) {
	if encoding == nil {
		return
	}
	encoding.Encoding = nil
	encoding.PrefixEncoding = nil
	encoding.ItemEncoding = nil
}
