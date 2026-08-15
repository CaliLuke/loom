package openapiv3_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/testutil"
	httpgen "github.com/CaliLuke/loom/http/codegen"
	"github.com/CaliLuke/loom/http/codegen/openapi"
	openapiv3 "github.com/CaliLuke/loom/http/codegen/openapi/v3"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestFilesEmitCanonicalOpenAPI32(t *testing.T) {
	openapi.Definitions = make(map[string]*openapi.Schema)
	root := httpgen.RunHTTPDSL(t, testdata.AsyncSessionSecurityDSL)
	root.API.Meta["openapi:self"] = []string{"https://api.example.com/openapi.json"}

	files, err := openapiv3.Files(root)
	require.NoError(t, err)
	require.Len(t, files, 2)

	artifacts := make(map[string][]byte, len(files))
	for _, file := range files {
		sections := file.AllSections()
		require.Len(t, sections, 1)
		buf := renderSection(t, sections[0])
		artifacts[file.Path] = append([]byte(nil), buf.Bytes()...)
	}

	jsonSpec := artifacts[filepath.Join(codegen.Gendir, "http", "openapi.json")]
	yamlSpec := artifacts[filepath.Join(codegen.Gendir, "http", "openapi.yaml")]
	require.NotEmpty(t, jsonSpec)
	require.NotEmpty(t, yamlSpec)
	validateOpenAPIVersion(t, jsonSpec, openapiv3.OpenAPIVersion)
	validateOpenAPIVersion(t, yamlSpec, openapiv3.OpenAPIVersion)

	spec := decodeOpenAPIJSON(t, jsonSpec)
	require.Equal(t, "https://api.example.com/openapi.json", requireString(t, spec["$self"], "OpenAPI document identity"))
	events := requireOperation(t, spec, "/events/{project_id}", "get")
	media := requireResponseMediaType(t, events, "text/event-stream")
	require.NotContains(t, media, "schema")
	itemSchema := requireMap(t, media["itemSchema"], "SSE item schema")
	properties := requireMap(t, itemSchema["properties"], "SSE item schema properties")
	data := requireMap(t, properties["data"], "SSE data schema")
	require.Equal(t, "string", requireString(t, data["type"], "SSE data type"))
	require.Equal(t, "application/json", requireString(t, data["contentMediaType"], "SSE data content type"))
	contentSchema := requireMap(t, data["contentSchema"], "SSE data content schema")
	require.Equal(t, "#/components/schemas/AsyncSessionRealtimeSSEEvent", requireString(t, contentSchema["$ref"], "SSE data schema ref"))
	require.Equal(t, "string", requireString(t, requireMap(t, properties["id"], "SSE id schema")["type"], "SSE id type"))
}

func TestFilesUseNormativeJSONSchemaDialectForAllTargets(t *testing.T) {
	const wantDialect = "https://spec.openapis.org/oas/3.1/dialect/base"

	tests := []struct {
		name        string
		target      string
		wantVersion string
	}{
		{name: "OpenAPI 3.1", target: "3.1", wantVersion: openapiv3.OpenAPICompatibilityVersion},
		{name: "OpenAPI 3.2", wantVersion: openapiv3.OpenAPIVersion},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			openapi.Definitions = make(map[string]*openapi.Schema)
			root := httpgen.RunHTTPDSL(t, testdata.SimpleDSL)
			if test.target != "" {
				if root.API.Meta == nil {
					root.API.Meta = make(map[string][]string)
				}
				root.API.Meta["openapi:version"] = []string{test.target}
			}

			files, err := openapiv3.Files(root)
			require.NoError(t, err)
			buf := renderSection(t, files[0].AllSections()[0])
			source := buf.Bytes()
			validateOpenAPIVersion(t, source, test.wantVersion)
			spec := decodeOpenAPIJSON(t, source)
			require.Equal(t, wantDialect, spec["jsonSchemaDialect"])
		})
	}
}

func TestRendererSkipsOpenAPI32OnlySectionsFor31Target(t *testing.T) {
	tests := []struct {
		name              string
		version           string
		wantVersion       string
		wantSelf          bool
		wantServerName    bool
		wantTagHierarchy  bool
		wantQuery         bool
		wantAdditionalOps bool
	}{
		{
			name:              "default 3.2",
			wantVersion:       openapiv3.OpenAPIVersion,
			wantSelf:          true,
			wantServerName:    true,
			wantTagHierarchy:  true,
			wantQuery:         true,
			wantAdditionalOps: true,
		},
		{
			name:        "3.1 compatibility target",
			version:     "3.1",
			wantVersion: openapiv3.OpenAPICompatibilityVersion,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			openapi.Definitions = make(map[string]*openapi.Schema)
			root := httpgen.RunHTTPDSL(t, testdata.OpenAPI32FeaturesDSL)
			if test.version != "" {
				root.API.Meta["openapi:version"] = []string{test.version}
			}

			files, err := openapiv3.Files(root)
			require.NoError(t, err)
			require.Len(t, files, 2)
			if test.version == "3.1" {
				require.Equal(t, []string{
					`OpenAPI 3.1 omits unsupported methods PURGE, QUERY from path "/books" and removes the path because no compatible operations remain`,
					`OpenAPI 3.1 omits unsupported method CONNECT from path "/tunnel" and removes the path because no compatible operations remain`,
				}, files[0].Warnings)
				require.Equal(t, files[0].Warnings, files[1].Warnings)
			} else {
				require.Empty(t, files[0].Warnings)
				require.Empty(t, files[1].Warnings)
			}
			buf := renderSection(t, files[0].AllSections()[0])
			jsonSpec := buf.Bytes()
			validateOpenAPIVersion(t, jsonSpec, test.wantVersion)
			spec := decodeOpenAPIJSON(t, jsonSpec)

			require.Equal(t, test.wantSelf, spec["$self"] != nil)
			servers := requireSlice(t, spec["servers"], "servers")
			server := requireMap(t, servers[0], "server")
			require.Equal(t, test.wantServerName, server["name"] != nil)
			assertOpenAPIMetadataPreserved(t, spec)
			tags := requireSlice(t, spec["tags"], "tags")
			var hierarchy bool
			for _, rawTag := range tags {
				tag := requireMap(t, rawTag, "tag")
				hierarchy = hierarchy || tag["summary"] != nil || tag["parent"] != nil || tag["kind"] != nil
			}
			require.Equal(t, test.wantTagHierarchy, hierarchy)

			paths := requireMap(t, spec["paths"], "paths")
			books, exists := paths["/books"]
			if !test.wantQuery && !test.wantAdditionalOps {
				require.False(t, exists)
				assertOpenAPI31CompatibilityProjection(t, spec)
				return
			}
			path := requireMap(t, books, "books path")
			require.Equal(t, test.wantQuery, path["query"] != nil)
			require.Equal(t, test.wantAdditionalOps, path["additionalOperations"] != nil)
		})
	}
}

func assertOpenAPIMetadataPreserved(t *testing.T, spec map[string]any) {
	t.Helper()
	servers := requireSlice(t, spec["servers"], "servers")
	server := requireMap(t, servers[0], "server")
	require.Equal(t, "Primary production environment.", server["description"])
	variables := requireMap(t, server["variables"], "server variables")
	region := requireMap(t, variables["region"], "region server variable")
	require.Equal(t, "Deployment region.", region["description"])

	operation := requireOperation(t, spec, "/parameters", "get")
	parameters := requireSlice(t, operation["parameters"], "parameter operation parameters")
	filter := findParameter(t, parameters, "header", "X-Filter")
	require.Equal(t, "Catalog filter.", filter["description"])
	require.Equal(t, "all", filter["example"])
	schema := requireMap(t, filter["schema"], "filter parameter schema")
	require.NotContains(t, schema, "description")
	require.NotContains(t, schema, "example")
}

func assertOpenAPI31CompatibilityProjection(t *testing.T, spec map[string]any) {
	t.Helper()
	require.Equal(t, "https://spec.openapis.org/oas/3.1/dialect/base", spec["jsonSchemaDialect"])
	paths := requireMap(t, spec["paths"], "paths")
	require.NotContains(t, paths, "/tunnel")
	events := requireMap(t, paths["/events"], "events path")
	get := requireMap(t, events["get"], "events GET")
	multipart := requireResponseMediaType(t, get, "multipart/mixed")
	require.NotContains(t, multipart, "$ref")
	require.NotNil(t, multipart["schema"])
	require.NotContains(t, multipart, "itemSchema")
	require.NotContains(t, multipart, "prefixEncoding")
	require.NotContains(t, multipart, "itemEncoding")

	components := requireMap(t, spec["components"], "components")
	require.NotContains(t, components, "mediaTypes")
	schemes := requireMap(t, components["securitySchemes"], "security schemes")
	require.Contains(t, schemes, "external")
	device := requireMap(t, schemes["device"], "device security scheme")
	require.NotContains(t, device, "deprecated")
	require.NotContains(t, device, "oauth2MetadataUrl")
	require.NotContains(t, requireMap(t, device["flows"], "OAuth flows"), "deviceAuthorization")

	schemas := requireMap(t, components["schemas"], "schemas")
	event := requireMap(t, schemas["CatalogEvent"], "CatalogEvent schema")
	require.NotContains(t, requireMap(t, event["xml"], "CatalogEvent XML"), "nodeType")
	change := requireMap(t, requireMap(t, event["properties"], "CatalogEvent properties")["change"], "change schema")
	discriminator := requireMap(t, change["discriminator"], "change discriminator")
	require.NotContains(t, discriminator, "defaultMapping")
	require.Contains(t, requireSlice(t, change["required"], "change required fields"), "type")

	parameters := requireOperation(t, spec, "/parameters", "get")
	requestParameters := requireSlice(t, parameters["parameters"], "parameter operation parameters")
	requestHeader := findParameter(t, requestParameters, "header", "X-Filter")
	require.NotContains(t, requestHeader, "allowReserved")
	requestCookie := findParameter(t, requestParameters, "cookie", "preferences")
	require.NotContains(t, requestCookie, "allowReserved")
	require.NotContains(t, requestCookie, "style")
	parameterResponse := requireMap(t, requireMap(t, parameters["responses"], "parameter responses")["200"], "parameter response")
	responseHeader := requireMap(t, requireMap(t, parameterResponse["headers"], "parameter response headers")["X-Cursor"], "response header")
	require.NotContains(t, responseHeader, "allowReserved")

	exampleOperation := requireOperation(t, spec, "/example", "get")
	exampleMedia := requireResponseMediaType(t, exampleOperation, "application/json")
	examples := requireMap(t, exampleMedia["examples"], "3.1 examples")
	structured := resolveExample(t, spec, requireMap(t, examples["structured"], "3.1 structured example"))
	require.NotNil(t, structured["value"])
	require.NotContains(t, structured, "dataValue")
	require.NotContains(t, structured, "serializedValue")
}

func TestFilesRejectUnsupportedOpenAPIVersion(t *testing.T) {
	openapi.Definitions = make(map[string]*openapi.Schema)
	root := httpgen.RunHTTPDSL(t, testdata.SimpleDSL)
	if root.API.Meta == nil {
		root.API.Meta = make(map[string][]string)
	}
	root.API.Meta["openapi:version"] = []string{"3.3"}

	_, err := openapiv3.Files(root)
	require.ErrorContains(t, err, `unsupported OpenAPI version "3.3"`)
}

func TestCanonicalOpenAPI32EmitsEveryNewFeatureFamily(t *testing.T) {
	openapi.Definitions = make(map[string]*openapi.Schema)
	root := httpgen.RunHTTPDSL(t, testdata.OpenAPI32FeaturesDSL)
	files, err := openapiv3.Files(root)
	require.NoError(t, err)
	buf := renderSection(t, files[0].AllSections()[0])
	validateOpenAPIVersion(t, buf.Bytes(), openapiv3.OpenAPIVersion)
	spec := decodeOpenAPIJSON(t, buf.Bytes())
	require.Equal(t, "https://spec.openapis.org/oas/3.1/dialect/base", spec["jsonSchemaDialect"])

	paths := requireMap(t, spec["paths"], "paths")
	books := requireMap(t, paths["/books"], "books path")
	query := requireMap(t, books["query"], "QUERY operation")
	additional := requireMap(t, books["additionalOperations"], "books additional operations")
	purge := requireMap(t, additional["PURGE"], "PURGE operation")
	require.NotEmpty(t, requireMap(t, purge["responses"], "PURGE responses"))
	tunnel := requireMap(t, paths["/tunnel"], "tunnel path")
	require.Contains(t, requireMap(t, tunnel["additionalOperations"], "tunnel additional operations"), "CONNECT")
	parameters := requireSlice(t, query["parameters"], "QUERY parameters")
	queryString := requireMap(t, parameters[0], "querystring parameter")
	require.Equal(t, "querystring", queryString["in"])
	require.NotNil(t, queryString["content"])

	responses := requireMap(t, query["responses"], "QUERY responses")
	response := requireMap(t, responses["200"], "QUERY response")
	require.Equal(t, "Event stream", response["summary"])
	require.NotContains(t, response, "description")
	jsonl := requireMap(t, requireMap(t, response["content"], "QUERY response content")["application/jsonl"], "JSONL media type")
	require.NotNil(t, jsonl["itemSchema"])
	require.NotContains(t, jsonl, "schema")
	examples := requireMap(t, jsonl["examples"], "JSONL examples")
	structured := resolveExample(t, spec, requireMap(t, examples["structured"], "structured example"))
	require.NotNil(t, structured["dataValue"])
	require.Equal(t, `{"id":"evt-1"}`, structured["serializedValue"])

	security := requireSlice(t, query["security"], "QUERY security")
	require.NotEmpty(t, security)
	require.Contains(t, requireMap(t, security[0], "security requirement"), "https://auth.example.com/security/external")
	components := requireMap(t, spec["components"], "components")
	schemes := requireMap(t, components["securitySchemes"], "security schemes")
	require.NotContains(t, schemes, "external")
	device := requireMap(t, schemes["device"], "device security scheme")
	require.Equal(t, true, device["deprecated"])
	require.Equal(t, "https://auth.example.com/.well-known/oauth-authorization-server", device["oauth2MetadataUrl"])
	flows := requireMap(t, device["flows"], "OAuth flows")
	deviceFlow := requireMap(t, flows["deviceAuthorization"], "device authorization flow")
	require.Equal(t, "https://auth.example.com/device", deviceFlow["deviceAuthorizationUrl"])

	schemas := requireMap(t, components["schemas"], "schemas")
	event := requireMap(t, schemas["CatalogEvent"], "CatalogEvent schema")
	xml := requireMap(t, event["xml"], "CatalogEvent XML")
	require.Equal(t, "element", xml["nodeType"])
	change := requireMap(t, requireMap(t, event["properties"], "CatalogEvent properties")["change"], "change schema")
	discriminator := requireMap(t, change["discriminator"], "change discriminator")
	require.Equal(t, "#/components/schemas/CatalogCreated", discriminator["defaultMapping"])
	required, _ := change["required"].([]any)
	require.NotContains(t, required, "type")

	events := requireMap(t, paths["/events"], "events path")
	get := requireMap(t, events["get"], "events GET")
	multipart := requireResponseMediaType(t, get, "multipart/mixed")
	require.Equal(t, "#/components/mediaTypes/CatalogEventStream", multipart["$ref"])
	mediaTypes := requireMap(t, components["mediaTypes"], "media type components")
	stream := requireMap(t, mediaTypes["CatalogEventStream"], "CatalogEventStream component")
	require.NotNil(t, stream["itemSchema"])
	require.NotNil(t, stream["prefixEncoding"])
	itemEncoding := requireMap(t, stream["itemEncoding"], "item encoding")
	nested := requireMap(t, itemEncoding["encoding"], "nested encoding")
	require.NotNil(t, nested["id"])

	parameterOperation := requireOperation(t, spec, "/parameters", "get")
	requestParameters := requireSlice(t, parameterOperation["parameters"], "parameter operation parameters")
	requestHeader := findParameter(t, requestParameters, "header", "X-Filter")
	require.Equal(t, true, requestHeader["allowReserved"])
	requestCookie := findParameter(t, requestParameters, "cookie", "preferences")
	require.Equal(t, true, requestCookie["allowReserved"])
	require.Equal(t, "cookie", requestCookie["style"])
	parameterResponse := requireMap(t, requireMap(t, parameterOperation["responses"], "parameter responses")["200"], "parameter response")
	responseHeader := requireMap(t, requireMap(t, parameterResponse["headers"], "parameter response headers")["X-Cursor"], "response header")
	require.Equal(t, true, responseHeader["allowReserved"])
}

func resolveExample(t *testing.T, spec, example map[string]any) map[string]any {
	t.Helper()
	ref, _ := example["$ref"].(string)
	if ref == "" {
		return example
	}
	const prefix = "#/components/examples/"
	require.True(t, strings.HasPrefix(ref, prefix), "unexpected example reference %q", ref)
	components := requireMap(t, spec["components"], "components")
	examples := requireMap(t, components["examples"], "component examples")
	return requireMap(t, examples[strings.TrimPrefix(ref, prefix)], "component example")
}

func findParameter(t *testing.T, parameters []any, in, name string) map[string]any {
	t.Helper()
	for _, raw := range parameters {
		parameter := requireMap(t, raw, "parameter")
		if parameter["in"] == in && parameter["name"] == name {
			return parameter
		}
	}
	t.Fatalf("parameter %s %q not found", in, name)
	return nil
}

func requireResponseMediaType(t *testing.T, operation map[string]any, contentType string) map[string]any {
	t.Helper()
	responses := requireMap(t, operation["responses"], "operation responses")
	response := requireMap(t, responses["200"], "operation response")
	content := requireMap(t, response["content"], "response content")
	return requireMap(t, content[contentType], "response media type")
}

func TestFiles(t *testing.T) {
	var (
		goldenPath = filepath.Join("testdata", "golden")
	)
	cases := []struct {
		Name string
		DSL  func()
	}{
		// TestSections
		{"file-service", testdata.FileServiceDSL},
		{"file-service-wildcard", testdata.FileServiceWildcardDSL},
		{"valid", testdata.SimpleDSL},
		{"multiple-services", testdata.MultipleServicesDSL},
		{"multiple-views", testdata.MultipleViewsDSL},
		{"explicit-view", testdata.ExplicitViewDSL},
		{"security", testdata.SecurityDSL},
		{"server-host-with-variables", testdata.ServerHostWithVariablesDSL},
		{"with-spaces", testdata.WithSpacesDSL},
		{"with-map", testdata.WithMapDSL},
		{"with-any", testdata.WithAnyDSL},
		{"path-with-wildcards", testdata.PathWithWildcardDSL},
		{"path-with-multiple-wildcards", testdata.PathWithMultipleWildcardDSL},
		{"path-with-multiple-explicit-wildcards", testdata.PathWithMultipleExplicitWildcardDSL},
		{"headers", testdata.HeadersDSL},
		{"with-tags", testdata.WithTagsDSL},
		{"meal-planner", testdata.MealPlannerDSL},
		{"collab-streams", testdata.CollabStreamsDSL},
		{"ops-socket", testdata.OpsSocketDSL},
		{"activity-feed", testdata.ActivityFeedDSL},
		{"streaming-partial-examples", testdata.StreamingPartialExamplesDSL},
		{"async-session-security", testdata.AsyncSessionSecurityDSL},
		{"raw-request-bodies", testdata.RawRequestBodyOpenAPIDSL},
		{"parameter-components", testdata.OpenAPIParameterComponentsDSL},
		{"reusable-components", testdata.OpenAPIReusableComponentsDSL},
		{"fingerprint-collisions", testdata.OpenAPIFingerprintCollisionsDSL},
		{"explicit-reusable-component-names", testdata.OpenAPIExplicitReusableComponentNamesDSL},
		{"request-response-split", testdata.OpenAPIRequestResponseSplitDSL},
		{"problem-links-async", testdata.OpenAPIProblemLinksAsyncDSL},
		{"typename", testdata.TypenameDSL},
		{"schema-dedup", testdata.OpenAPISchemaDedupDSL},
		{"not-generate-server", testdata.NotGenerateServerDSL},
		{"not-generate-host", testdata.NotGenerateHostDSL},
		{"not-generate-attribute", testdata.NotGenerateAttributeDSL},
		{"json-prefix", testdata.JSONPrefixDSL},
		{"json-indent", testdata.JSONIndentDSL},
		{"json-prefix-indent", testdata.JSONPrefixIndentDSL},
		// TestEndpoints
		{"endpoint", testdata.ExtensionDSL},
		{"skip-response-body-encode-decode", testdata.SkipResponseBodyEncodeDecodeDSL},
		// TestValidations
		{"string", testdata.StringValidationDSL},
		{"integer", testdata.IntValidationDSL},
		{"array", testdata.ArrayValidationDSL},
		// Error examples
		{"error-examples", testdata.ErrorExamplesDSL},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			// Reset global variables
			openapi.Definitions = make(map[string]*openapi.Schema)
			root := httpgen.RunHTTPDSL(t, c.DSL)
			oFiles, err := openapiv3.Files(root)
			if err != nil {
				t.Fatalf("OpenAPI failed with %s", err)
			}
			for i, o := range oFiles {
				tname := fmt.Sprintf("file%d", i)
				sections := o.AllSections()
				t.Run(tname, func(t *testing.T) {
					if len(sections) != 1 {
						t.Fatalf("expected 1 section, got %d", len(sections))
					}
					buf := renderSection(t, sections[0])
					validateOpenAPI(t, buf.Bytes())

					golden := filepath.Join(goldenPath, fmt.Sprintf("%s_%s.golden", c.Name, tname))
					if filepath.Ext(o.Path) == ".json" {
						testutil.AssertJSON(t, golden, buf.Bytes())
					} else {
						testutil.AssertString(t, golden, buf.String())
					}
				})
			}
		})
	}
}

func TestRenderedJSONFileEndsWithFinalLineFeed(t *testing.T) {
	openapi.Definitions = make(map[string]*openapi.Schema)
	root := httpgen.RunHTTPDSL(t, testdata.SimpleDSL)
	files, err := openapiv3.Files(root)
	if err != nil {
		t.Fatalf("OpenAPI failed with %s", err)
	}

	var jsonFile *codegen.File
	for _, file := range files {
		if filepath.Ext(file.Path) == ".json" {
			jsonFile = file
			break
		}
	}
	if jsonFile == nil {
		t.Fatal("OpenAPI did not produce a JSON file")
	}

	path, err := jsonFile.Render(t.TempDir())
	if err != nil {
		t.Fatalf("failed rendering OpenAPI JSON: %s", err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed reading OpenAPI JSON: %s", err)
	}
	if !bytes.HasSuffix(content, []byte("}\n")) {
		t.Fatalf("OpenAPI JSON must end with exactly one LF, got final bytes %q", content[max(0, len(content)-2):])
	}
}

func TestRenderedUnionSchemasIncludeDiscriminatorMappingsAndEnvelopeRefs(t *testing.T) {
	cases := []struct {
		name      string
		dsl       func()
		typeKey   string
		valueKey  string
		wantTags  []string
		envelopes []string
	}{
		{
			name:      "payload-custom-keys",
			dsl:       testdata.PayloadBodyUnionCustomKeysDSL,
			typeKey:   "kind",
			valueKey:  "data",
			wantTags:  []string{"Int", "String"},
			envelopes: []string{"ValuesIntEnvelope", "ValuesStringEnvelope"},
		},
		{
			name:      "result-custom-keys-multi",
			dsl:       testdata.ResultBodyUnionCustomKeysMultiDSL,
			typeKey:   "statusType",
			valueKey:  "statusDetails",
			wantTags:  []string{"failure", "success"},
			envelopes: []string{"StatusFailureEnvelope", "StatusSuccessEnvelope"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			spec := renderYAMLOpenAPI(t, tc.dsl)
			assertRenderedUnionContract(t, spec, tc.typeKey, tc.valueKey, tc.wantTags, tc.envelopes)
		})
	}
}

func TestRenderedSpecDeduplicatesGeneratedRequestBodiesAndUnionEnvelopes(t *testing.T) {
	spec := renderYAMLOpenAPI(t, testdata.OpenAPISchemaDedupDSL)

	requestRefs := regexp.MustCompile(`(?m)^\s+\$ref: '#/components/schemas/([^']+RequestBody)'$`).FindAllStringSubmatch(spec, -1)
	if len(requestRefs) != 2 {
		t.Fatalf("expected 2 request body refs for the duplicated JSON payload methods, got %d\nspec:\n%s", len(requestRefs), spec)
	}

	counts := make(map[string]int)
	for _, match := range requestRefs {
		counts[match[1]]++
	}
	if counts["FirstRequestBody"] < 2 {
		t.Fatalf("expected FirstRequestBody to be reused across duplicate request payloads, got counts %#v\nspec:\n%s", counts, spec)
	}
	if _, ok := counts["SecondRequestBody"]; ok {
		t.Fatalf("expected duplicate request body component to be removed, got counts %#v\nspec:\n%s", counts, spec)
	}

	envelopes := regexp.MustCompile(`(?m)^\s+([A-Za-z0-9]+Envelope):$`).FindAllStringSubmatch(spec, -1)
	if len(envelopes) != 2 {
		t.Fatalf("expected exactly 2 deduplicated union envelope components, got %d\nspec:\n%s", len(envelopes), spec)
	}
}

func TestRenderedSpecDeduplicatesRepeatedParameters(t *testing.T) {
	spec := renderYAMLOpenAPI(t, testdata.OpenAPIParameterComponentsDSL)

	if count := len(regexp.MustCompile(`(?m)^components:$`).FindAllString(spec, -1)); count != 1 {
		t.Fatalf("expected exactly one components block, got %d\nspec:\n%s", count, spec)
	}
	if count := len(regexp.MustCompile(`(?m)^    parameters:$`).FindAllString(spec, -1)); count != 1 {
		t.Fatalf("expected exactly one components.parameters block, got %d\nspec:\n%s", count, spec)
	}
	if count := len(regexp.MustCompile(`(?m)^\s+PathWidgetID:$`).FindAllString(spec, -1)); count != 1 {
		t.Fatalf("expected PathWidgetID parameter component once, got %d\nspec:\n%s", count, spec)
	}
	if count := len(regexp.MustCompile(`(?m)^\s+QueryLimit:$`).FindAllString(spec, -1)); count != 1 {
		t.Fatalf("expected QueryLimit parameter component once, got %d\nspec:\n%s", count, spec)
	}
	if count := len(regexp.MustCompile(`(?m)^\s+- \$ref: '#/components/parameters/PathWidgetID'$`).FindAllString(spec, -1)); count != 2 {
		t.Fatalf("expected PathWidgetID to be referenced twice, got %d\nspec:\n%s", count, spec)
	}
	if count := len(regexp.MustCompile(`(?m)^\s+- \$ref: '#/components/parameters/QueryLimit'$`).FindAllString(spec, -1)); count != 2 {
		t.Fatalf("expected QueryLimit to be referenced twice, got %d\nspec:\n%s", count, spec)
	}
}

func TestRenderedSpecReusesRequestBodiesResponsesHeadersExamplesAndServiceTags(t *testing.T) {
	spec := renderYAMLOpenAPI(t, testdata.OpenAPIReusableComponentsDSL)

	requirePattern := func(pattern string) {
		t.Helper()
		re := regexp.MustCompile(pattern)
		if !re.MatchString(spec) {
			t.Fatalf("spec did not match pattern %q\nspec:\n%s", pattern, spec)
		}
	}

	requirePattern(`(?m)^    requestBodies:$`)
	requirePattern(`(?m)^    responses:$`)
	requirePattern(`(?m)^    headers:$`)
	requirePattern(`(?m)^    examples:$`)
	requirePattern(`(?m)^\s+- name: Auth$`)
	requirePattern(`(?s)post:\n\s+tags:\n\s+- Auth`)
	requirePattern(`(?m)^\s+\$ref: '#/components/requestBodies/CredentialsRequestBody'$`)
	requirePattern(`(?m)^\s+CredentialsRequestBody:$`)
	requirePattern(`(?m)^\s+Session_f01e4e234bbb10ecStatus200Response:$`)
	requirePattern(`(?s)responses:\n\s+"200":\n\s+\$ref: '#/components/responses/Session_f01e4e234bbb10ecStatus200Response'`)
	requirePattern(`(?m)^\s+NoContentResponse:$`)
	requirePattern(`(?s)responses:\n\s+"204":\n\s+\$ref: '#/components/responses/NoContentResponse'`)
	requirePattern(`(?m)^\s+X-Trace-ID:\n\s+\$ref: '#/components/headers/XTraceIDHeader'$`)
	requirePattern(`(?m)^\s+AuthRefreshRequestBodyPrimaryExample:$`)
	requirePattern(`(?m)^\s+AuthRefreshStatus200ResponseCurrentExample:$`)
	requirePattern(`(?m)^\s+XTraceIDHeaderTraceAExample:$`)
	requirePattern(`(?m)password:$`)
	requirePattern(`(?m)writeOnly: true`)
	requirePattern(`(?m)legacyToken:$`)
	requirePattern(`(?m)deprecated: true`)
	requirePattern(`(?m)profile:$`)
	requirePattern(`(?m)contentEncoding: base64`)
	requirePattern(`(?m)contentMediaType: application/json`)

	if count := len(regexp.MustCompile(`(?m)^\s+\$ref: '#/components/requestBodies/CredentialsRequestBody'$`).FindAllString(spec, -1)); count != 2 {
		t.Fatalf("expected CredentialsRequestBody to be reused exactly twice, got %d\nspec:\n%s", count, spec)
	}
	if count := len(regexp.MustCompile(`(?m)^\s+\$ref: '#/components/responses/Session_f01e4e234bbb10ecStatus200Response'$`).FindAllString(spec, -1)); count != 2 {
		t.Fatalf("expected shared session response component to be reused exactly twice, got %d\nspec:\n%s", count, spec)
	}
}

func TestRenderedSpecUsesExplicitReusableRequestBodyAndParameterNames(t *testing.T) {
	spec := renderYAMLOpenAPI(t, testdata.OpenAPIExplicitReusableComponentNamesDSL)

	requirePattern := func(pattern string) {
		t.Helper()
		re := regexp.MustCompile(pattern)
		if !re.MatchString(spec) {
			t.Fatalf("spec did not match pattern %q\nspec:\n%s", pattern, spec)
		}
	}

	requirePattern(`(?m)^\s+WidgetIDParam:$`)
	requirePattern(`(?m)^\s+SearchFiltersRequest:$`)
	requirePattern(`(?m)^\s+- \$ref: '#/components/parameters/WidgetIDParam'$`)
	requirePattern(`(?m)^\s+\$ref: '#/components/requestBodies/SearchFiltersRequest'$`)
}

func TestRenderedSpecPreservesNamedRequestBodyExamples(t *testing.T) {
	spec := renderYAMLOpenAPI(t, testdata.OpenAPINamedRequestBodyExamplesDSL)

	requirePattern := func(pattern string) {
		t.Helper()
		re := regexp.MustCompile(pattern)
		if !re.MatchString(spec) {
			t.Fatalf("spec did not match pattern %q\nspec:\n%s", pattern, spec)
		}
	}

	requirePattern(`(?m)^\s+SearchFiltersRequest:$`)
	requirePattern(`(?m)^\s+examples:$`)
	requirePattern(`(?m)^\s+simple:$`)
	requirePattern(`(?m)^\s+advanced:$`)
	requirePattern(`(?m)^\s+query: soup$`)
	requirePattern(`(?m)^\s+query: stew$`)
}

func TestRenderedSpecPreservesNamedExamplesForExplicitBodyWrapperTypes(t *testing.T) {
	spec := renderYAMLOpenAPI(t, testdata.OpenAPIExplicitBodyWrapperExamplesDSL)

	requirePattern := func(pattern string) {
		t.Helper()
		re := regexp.MustCompile(pattern)
		if !re.MatchString(spec) {
			t.Fatalf("spec did not match pattern %q\nspec:\n%s", pattern, spec)
		}
	}

	requirePattern(`(?m)^\s+SearchFiltersRequestBody:$`)
	requirePattern(`(?m)^\s+examples:$`)
	requirePattern(`(?m)^\s+simple:$`)
	requirePattern(`(?m)^\s+advanced:$`)
	requirePattern(`(?m)^\s+query: soup$`)
	requirePattern(`(?m)^\s+query: stew$`)
}

func TestRenderedSpecSplitsRequestAndResponseSchemasFromDirectionalMetadata(t *testing.T) {
	spec := renderYAMLOpenAPI(t, testdata.OpenAPIRequestResponseSplitDSL)

	requirePattern := func(pattern string) {
		t.Helper()
		re := regexp.MustCompile(pattern)
		if !re.MatchString(spec) {
			t.Fatalf("spec did not match pattern %q\nspec:\n%s", pattern, spec)
		}
	}

	requirePattern(`(?m)^\s+CreateRequestBodyRequest:$`)
	requirePattern(`(?m)^\s+CreateResponseBodyResponse:$`)
	requirePattern(`(?s)CreateRequestBodyRequest:.*email:.*password:`)
	requirePattern(`(?s)CreateResponseBodyResponse:.*id:.*email:`)
}

func TestRenderedSpecPublishesProblemLinksAndAsyncContracts(t *testing.T) {
	spec := renderYAMLOpenAPI(t, testdata.OpenAPIProblemLinksAsyncDSL)

	requirePattern := func(pattern string) {
		t.Helper()
		re := regexp.MustCompile(pattern)
		if !re.MatchString(spec) {
			t.Fatalf("spec did not match pattern %q\nspec:\n%s", pattern, spec)
		}
	}

	requirePattern(`(?m)^    responses:$`)
	requirePattern(`(?m)^\s+Problem:$`)
	requirePattern(`(?m)^\s+application/problem\+json:$`)
	requirePattern(`(?m)^\s+OpenAPIThreadAcceptedStatus202Response:$`)
	requirePattern(`(?m)^\s+links:$`)
	requirePattern(`(?m)^\s+thread:$`)
	requirePattern(`(?m)^\s+watch:$`)
	requirePattern(`(?m)^\s+operationId: thread_ops\.get_thread$`)
	requirePattern(`(?m)^\s+operationId: thread_ops\.watch_thread$`)
	requirePattern(`(?m)^\s+x-loom-async:$`)
	requirePattern(`(?m)^\s+transport: sse$`)
	requirePattern(`(?m)^\s+transport: websocket$`)
	requirePattern(`(?m)^\s+direction: server$`)
	requirePattern(`(?m)^\s+direction: bidirectional$`)
	requirePattern(`(?m)^\s+contentType: text/event-stream$`)
	requirePattern(`(?m)^\s+status: 101$`)
	requirePattern(`(?m)^\s+messages:$`)
	requirePattern(`(?m)^\s+schema:$`)
	requirePattern(`(?m)^\s+\$ref: '#/components/schemas/OpenAPIThreadEvent'$`)
}

func TestRenderedSpecPublishesSecuredAsyncSessionContracts(t *testing.T) {
	spec := renderYAMLOpenAPI(t, testdata.AsyncSessionSecurityDSL)

	requirePattern := func(pattern string) {
		t.Helper()
		re := regexp.MustCompile(pattern)
		if !re.MatchString(spec) {
			t.Fatalf("spec did not match pattern %q\nspec:\n%s", pattern, spec)
		}
	}

	requirePattern(`(?m)^\s+browser_session_cookie:$`)
	requirePattern(`(?m)^\s+in: cookie$`)
	requirePattern(`(?m)^\s+name: __Host-ak_session$`)
	requirePattern(`(?m)^    /ws/projects/\{project_id\}:$`)
	requirePattern(`(?m)^    /events/\{project_id\}:$`)
	requirePattern(`(?m)^\s+PathProjectID:$`)
	requirePattern(`(?m)^\s+name: project_id$`)
	requirePattern(`(?m)^\s+name: last_event_id$`)
	requirePattern(`(?m)^\s+security:$`)
	requirePattern(`(?m)^\s+- browser_session_cookie: \[\]$`)
	requirePattern(`(?m)^\s+x-loom-async:$`)
	requirePattern(`(?m)^\s+transport: websocket$`)
	requirePattern(`(?m)^\s+transport: sse$`)
	requirePattern(`(?m)^\s+direction: bidirectional$`)
	requirePattern(`(?m)^\s+direction: server$`)
	requirePattern(`(?m)^\s+status: 101$`)
	requirePattern(`(?m)^\s+contentType: text/event-stream$`)
}

func TestRenderedSpecClosedObjectModeClosesObjectsAndUsesUnevaluatedPropertiesForUnions(t *testing.T) {
	spec := renderYAMLOpenAPI(t, testdata.OpenAPIClosedObjectsDSL)

	requirePattern := func(pattern string) {
		t.Helper()
		re := regexp.MustCompile(pattern)
		if !re.MatchString(spec) {
			t.Fatalf("spec did not match pattern %q\nspec:\n%s", pattern, spec)
		}
	}

	requirePattern(`(?m)ObjectRequestBody:$`)
	requirePattern(`(?m)address:$`)
	requirePattern(`(?m)\$ref: '#/components/schemas/ClosedObjectsNested'$`)
	requirePattern(`(?s)ObjectRequestBody:.*additionalProperties: false`)
	requirePattern(`(?s)ClosedObjectsNested:.*additionalProperties: false`)
	requirePattern(`(?m)MapObjectRequestBody:$`)
	requirePattern(`(?m)labels:$`)
	requirePattern(`(?m)additionalProperties:$`)
	requirePattern(`(?m)type: string`)
	requirePattern(`(?m)unevaluatedProperties: false`)
}

func validateOpenAPI(t *testing.T, b []byte) {
	validateOpenAPIVersion(t, b, openapiv3.OpenAPIVersion)
}

func validateOpenAPIVersion(t *testing.T, b []byte, wantVersion string) {
	t.Helper()
	parsed, err := libopenapi.NewDocument(b)
	if err != nil {
		t.Fatalf("libopenapi failed to parse generated spec: %s\nspec:\n%s", err, string(b))
	}
	if parsed.GetVersion() != wantVersion {
		t.Fatalf("libopenapi parsed version %q, expected %q", parsed.GetVersion(), wantVersion)
	}
	if _, err := parsed.BuildV3Model(); err != nil {
		t.Fatalf("libopenapi failed to build 3.x model: %s\nspec:\n%s", err, string(b))
	}
}

func renderYAMLOpenAPI(t *testing.T, dsl func()) string {
	t.Helper()

	openapi.Definitions = make(map[string]*openapi.Schema)
	root := httpgen.RunHTTPDSL(t, dsl)
	oFiles, err := openapiv3.Files(root)
	if err != nil {
		t.Fatalf("OpenAPI failed with %s", err)
	}

	for _, o := range oFiles {
		if filepath.Ext(o.Path) != ".yaml" {
			continue
		}
		sections := o.AllSections()
		if len(sections) != 1 {
			t.Fatalf("expected 1 section for %s, got %d", o.Path, len(sections))
		}
		buf := renderSection(t, sections[0])
		validateOpenAPI(t, buf.Bytes())
		return buf.String()
	}

	t.Fatal("missing YAML OpenAPI output")
	return ""
}

func renderSection(t *testing.T, section codegen.Section) bytes.Buffer {
	t.Helper()

	var buf bytes.Buffer
	if err := section.Write(&buf); err != nil {
		t.Fatalf("failed to render section %q: %s", section.SectionName(), err)
	}
	return buf
}

func assertRenderedUnionContract(t *testing.T, spec, typeKey, valueKey string, wantTags, envelopePrefixes []string) {
	t.Helper()

	requirePattern := func(pattern string) {
		t.Helper()
		re := regexp.MustCompile(pattern)
		if !re.MatchString(spec) {
			t.Fatalf("spec did not match pattern %q\nspec:\n%s", pattern, spec)
		}
	}

	requirePattern(fmt.Sprintf(`(?m)^\s+propertyName: %s$`, regexp.QuoteMeta(typeKey)))
	requirePattern(`(?m)^\s+mapping:$`)
	requirePattern(`(?m)^\s+oneOf:$`)

	oneOfMatches := regexp.MustCompile(`(?m)^\s+- \$ref: '#/components/schemas/([^']+Envelope[^']*)'$`).FindAllStringSubmatch(spec, -1)
	if len(oneOfMatches) < len(wantTags) {
		t.Fatalf("got %d oneOf envelope refs, expected at least %d\nspec:\n%s", len(oneOfMatches), len(wantTags), spec)
	}
	oneOfRefs := make([]string, 0, len(oneOfMatches))
	for _, match := range oneOfMatches {
		oneOfRefs = append(oneOfRefs, match[1])
	}

	for i, tag := range wantTags {
		prefix := envelopePrefixes[i]
		re := regexp.MustCompile(fmt.Sprintf(`(?m)^\s+%s: '#/components/schemas/((?:%s)(?:_[0-9a-f]+(?:_\d+)?)?)'$`, regexp.QuoteMeta(tag), regexp.QuoteMeta(prefix)))
		match := re.FindStringSubmatch(spec)
		if len(match) != 2 {
			t.Fatalf("missing mapping for tag %q with envelope prefix %q\nspec:\n%s", tag, prefix, spec)
		}
		if !slices.Contains(oneOfRefs, match[1]) {
			t.Fatalf("mapping ref %q for tag %q is not present in oneOf refs %#v", match[1], tag, oneOfRefs)
		}
		requirePattern(fmt.Sprintf(`(?m)^\s+%s:$`, regexp.QuoteMeta(match[1])))
		requirePattern(fmt.Sprintf(`(?m)^\s+- %s$`, regexp.QuoteMeta(typeKey)))
		requirePattern(fmt.Sprintf(`(?m)^\s+- %s$`, regexp.QuoteMeta(valueKey)))
	}
}
