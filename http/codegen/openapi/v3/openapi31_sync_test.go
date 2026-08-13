package openapiv3

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi/datamodel"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/openapi"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

type (
	openAPIModelSchema struct {
		owner  reflect.Type
		path31 string
		path32 string
	}

	openAPIVersionMember struct {
		owner    reflect.Type
		field    int
		jsonName string
	}

	openAPIReflectVisit struct {
		typ reflect.Type
		ptr uintptr
	}
)

func TestOpenAPI31FilterCoversEveryOpenAPI32ModelMember(t *testing.T) {
	members := deriveOpenAPI32ModelMembers(t)
	require.NotEmpty(t, members)

	openapi.Definitions = make(map[string]*openapi.Schema)
	root32 := expr.RunDSL(t, testdata.OpenAPI32FeaturesDSL)
	spec32, warnings := newForVersion(root32, openAPIVersion32)
	require.Empty(t, warnings)
	addOpenAPI32LinkServerCases(t, spec32)

	openapi.Definitions = make(map[string]*openapi.Schema)
	root31 := expr.RunDSL(t, testdata.OpenAPI32FeaturesDSL)
	spec31, _ := newForVersion(root31, openAPIVersion31)
	addOpenAPI32LinkServerCases(t, spec31)
	filterOpenAPI31(spec31)

	assertOpenAPIVersionMembers(t, "3.2 fixture", spec32, members, true)
	assertOpenAPIVersionMembers(t, "3.1 projection", spec31, members, false)
}

func deriveOpenAPI32ModelMembers(t *testing.T) []openAPIVersionMember {
	t.Helper()
	schema31 := decodeOpenAPISchema(t, datamodel.OpenAPI31SchemaData)
	schema32 := decodeOpenAPISchema(t, datamodel.OpenAPI32SchemaData)
	models := []openAPIModelSchema{
		{reflect.TypeFor[OpenAPI](), "", ""},
		{reflect.TypeFor[Info](), "info", "info"},
		{reflect.TypeFor[Contact](), "contact", "contact"},
		{reflect.TypeFor[License](), "license", "license"},
		{reflect.TypeFor[Server](), "server", "server"},
		{reflect.TypeFor[ServerVariable](), "server-variable", "server-variable"},
		{reflect.TypeFor[Components](), "components", "components"},
		{reflect.TypeFor[PathItem](), "path-item", "path-item"},
		{reflect.TypeFor[Operation](), "operation", "operation"},
		{reflect.TypeFor[openapi.ExternalDocs](), "external-documentation", "external-documentation"},
		{reflect.TypeFor[Parameter](), "parameter", "parameter"},
		{reflect.TypeFor[RequestBody](), "request-body", "request-body"},
		{reflect.TypeFor[MediaType](), "media-type", "media-type"},
		{reflect.TypeFor[Encoding](), "encoding", "encoding"},
		{reflect.TypeFor[Response](), "response", "response"},
		{reflect.TypeFor[Example](), "example", "example"},
		{reflect.TypeFor[Link](), "link", "link"},
		{reflect.TypeFor[Header](), "header", "header"},
		{reflect.TypeFor[openapi.Tag](), "tag", "tag"},
		{reflect.TypeFor[SecurityScheme](), "security-scheme", "security-scheme"},
		{reflect.TypeFor[OAuthFlows](), "oauth-flows", "oauth-flows"},
		{reflect.TypeFor[OAuthFlow](), "oauth-flow", "oauth-flow"},
	}

	var members []openAPIVersionMember
	for _, model := range models {
		properties31 := openAPISchemaProperties(schema31, model.path31)
		properties32 := openAPISchemaProperties(schema32, model.path32)
		for field := range model.owner.NumField() {
			jsonName := jsonFieldName(model.owner.Field(field))
			if jsonName == "" || properties31[jsonName] || !properties32[jsonName] {
				continue
			}
			members = append(members, openAPIVersionMember{
				owner:    model.owner,
				field:    field,
				jsonName: jsonName,
			})
		}
	}

	// The OAS document schema deliberately leaves Schema Object keywords open.
	// Keep the unavoidable vocabulary exceptions next to the mechanical inventory.
	members = append(members,
		versionMemberForField(t, reflect.TypeFor[openapi.Discriminator](), "DefaultMapping"),
		versionMemberForField(t, reflect.TypeFor[openapi.XML](), "NodeType"),
	)
	slices.SortFunc(members, func(a, b openAPIVersionMember) int {
		return strings.Compare(versionMemberName(a), versionMemberName(b))
	})
	return members
}

func decodeOpenAPISchema(t *testing.T, source string) map[string]any {
	t.Helper()
	var schema map[string]any
	require.NoError(t, json.Unmarshal([]byte(source), &schema))
	return schema
}

func openAPISchemaProperties(schema map[string]any, path string) map[string]bool {
	node := any(schema)
	if path != "" {
		if path == "oauth-flow" {
			node = openAPIOAuthFlowSchema(schema)
		} else {
			node = schema["$defs"].(map[string]any)[path]
		}
	}
	properties := make(map[string]bool)
	collectOpenAPISchemaProperties(schema, node, make(map[string]bool), properties)
	return properties
}

func openAPIOAuthFlowSchema(schema map[string]any) map[string]any {
	flows := schema["$defs"].(map[string]any)["oauth-flows"].(map[string]any)
	definitions := flows["$defs"].(map[string]any)
	variants := make([]string, 0, len(definitions))
	for name := range definitions {
		variants = append(variants, name)
	}
	slices.Sort(variants)
	anyOf := make([]any, 0, len(variants))
	for _, name := range variants {
		anyOf = append(anyOf, definitions[name])
	}
	return map[string]any{"anyOf": anyOf}
}

func collectOpenAPISchemaProperties(
	schema map[string]any,
	node any,
	seenRefs map[string]bool,
	properties map[string]bool,
) {
	object, ok := node.(map[string]any)
	if !ok {
		return
	}
	if fields, ok := object["properties"].(map[string]any); ok {
		for name := range fields {
			properties[name] = true
		}
	}
	if ref, ok := object["$ref"].(string); ok && strings.HasPrefix(ref, "#/") && !seenRefs[ref] {
		seenRefs[ref] = true
		collectOpenAPISchemaProperties(schema, resolveOpenAPISchemaRef(schema, ref), seenRefs, properties)
	}
	for _, keyword := range []string{"allOf", "anyOf", "oneOf"} {
		if entries, ok := object[keyword].([]any); ok {
			for _, entry := range entries {
				collectOpenAPISchemaProperties(schema, entry, seenRefs, properties)
			}
		}
	}
	for _, keyword := range []string{"if", "then", "else"} {
		collectOpenAPISchemaProperties(schema, object[keyword], seenRefs, properties)
	}
	if dependencies, ok := object["dependentSchemas"].(map[string]any); ok {
		for _, dependency := range dependencies {
			collectOpenAPISchemaProperties(schema, dependency, seenRefs, properties)
		}
	}
}

func resolveOpenAPISchemaRef(schema map[string]any, ref string) any {
	var node any = schema
	for _, raw := range strings.Split(strings.TrimPrefix(ref, "#/"), "/") {
		part := strings.ReplaceAll(strings.ReplaceAll(raw, "~1", "/"), "~0", "~")
		node = node.(map[string]any)[part]
	}
	return node
}

func versionMemberForField(t *testing.T, owner reflect.Type, fieldName string) openAPIVersionMember {
	t.Helper()
	field, ok := owner.FieldByName(fieldName)
	require.Truef(t, ok, "missing %s.%s", owner.Name(), fieldName)
	return openAPIVersionMember{
		owner:    owner,
		field:    field.Index[0],
		jsonName: jsonFieldName(field),
	}
}

func jsonFieldName(field reflect.StructField) string {
	name := strings.Split(field.Tag.Get("json"), ",")[0]
	if name == "-" {
		return ""
	}
	if name == "" {
		return field.Name
	}
	return name
}

func assertOpenAPIVersionMembers(
	t *testing.T,
	label string,
	spec *OpenAPI,
	members []openAPIVersionMember,
	wantPresent bool,
) {
	t.Helper()
	present := observeOpenAPIVersionMembers(spec, members)
	var mismatches []string
	for _, member := range members {
		if present[versionMemberName(member)] != wantPresent {
			mismatches = append(mismatches, versionMemberName(member))
		}
	}
	slices.Sort(mismatches)
	if wantPresent {
		require.Empty(t, mismatches, "%s does not exercise derived 3.2-only members", label)
		return
	}
	require.Empty(t, mismatches, "%s leaks derived 3.2-only members", label)
}

func observeOpenAPIVersionMembers(root any, members []openAPIVersionMember) map[string]bool {
	byOwner := make(map[reflect.Type][]openAPIVersionMember)
	for _, member := range members {
		byOwner[member.owner] = append(byOwner[member.owner], member)
	}
	present := make(map[string]bool, len(members))
	visitOpenAPIValue(reflect.ValueOf(root), byOwner, make(map[openAPIReflectVisit]bool), present)
	return present
}

func visitOpenAPIValue(
	value reflect.Value,
	members map[reflect.Type][]openAPIVersionMember,
	seen map[openAPIReflectVisit]bool,
	present map[string]bool,
) {
	if !value.IsValid() {
		return
	}
	for value.Kind() == reflect.Interface {
		if value.IsNil() {
			return
		}
		value = value.Elem()
	}
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return
		}
		visit := openAPIReflectVisit{typ: value.Type(), ptr: value.Pointer()}
		if seen[visit] {
			return
		}
		seen[visit] = true
		visitOpenAPIValue(value.Elem(), members, seen, present)
		return
	}
	switch value.Kind() {
	case reflect.Struct:
		for _, member := range members[value.Type()] {
			if !value.Field(member.field).IsZero() {
				present[versionMemberName(member)] = true
			}
		}
		for field := range value.NumField() {
			if value.Type().Field(field).IsExported() {
				visitOpenAPIValue(value.Field(field), members, seen, present)
			}
		}
	case reflect.Slice, reflect.Array:
		for index := range value.Len() {
			visitOpenAPIValue(value.Index(index), members, seen, present)
		}
	case reflect.Map:
		keys := value.MapKeys()
		slices.SortFunc(keys, func(a, b reflect.Value) int {
			return strings.Compare(fmt.Sprint(a.Interface()), fmt.Sprint(b.Interface()))
		})
		for _, key := range keys {
			visitOpenAPIValue(value.MapIndex(key), members, seen, present)
		}
	}
}

func versionMemberName(member openAPIVersionMember) string {
	return member.owner.PkgPath() + "." + member.owner.Name() + "." + member.jsonName
}

func addOpenAPI32LinkServerCases(t *testing.T, spec *OpenAPI) {
	t.Helper()
	require.NotNil(t, spec)
	require.NotNil(t, spec.Components)
	link := func() *LinkRef {
		return &LinkRef{Value: &Link{
			OperationID: "catalog.search",
			Server: &Server{
				Name: "link-server",
				URL:  "https://api.example.com",
			},
		}}
	}
	if spec.Components.Links == nil {
		spec.Components.Links = make(map[string]*LinkRef)
	}
	spec.Components.Links["CatalogSearch"] = link()
	path := spec.Paths["/parameters"]
	require.NotNil(t, path)
	require.NotNil(t, path.Get)
	response := path.Get.Responses["200"]
	require.NotNil(t, response)
	require.NotNil(t, response.Value)
	if response.Value.Links == nil {
		response.Value.Links = make(map[string]*LinkRef)
	}
	response.Value.Links["CatalogSearch"] = link()
}
