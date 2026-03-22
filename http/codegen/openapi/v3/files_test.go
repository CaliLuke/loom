package openapiv3_test

import (
	"bytes"
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"testing"
	"text/template"

	"github.com/pb33f/libopenapi"

	"goa.design/goa/v3/codegen/testutil"
	httpgen "goa.design/goa/v3/http/codegen"
	"goa.design/goa/v3/http/codegen/openapi"
	openapiv3 "goa.design/goa/v3/http/codegen/openapi/v3"
	"goa.design/goa/v3/http/codegen/testdata"
)

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
		{"parameter-components", testdata.OpenAPIParameterComponentsDSL},
		{"reusable-components", testdata.OpenAPIReusableComponentsDSL},
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
				s := o.SectionTemplates
				t.Run(tname, func(t *testing.T) {
					if len(s) != 1 {
						t.Fatalf("expected 1 section, got %d", len(s))
					}
					if s[0].Source == "" {
						t.Fatalf("empty section template")
					}
					if s[0].Data == nil {
						t.Fatalf("nil data")
					}
					var buf bytes.Buffer
					tmpl := template.Must(template.New("openapi").Funcs(s[0].FuncMap).Parse(s[0].Source))
					if err := tmpl.Execute(&buf, s[0].Data); err != nil {
						t.Fatalf("failed to render template: %s", err)
					}
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
			envelopes: []string{"StatusfailureEnvelope", "StatussuccessEnvelope"},
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
	requirePattern(`(?m)^\s+Session_d5d120e942d17641Status200Response:$`)
	requirePattern(`(?s)responses:\n\s+"200":\n\s+\$ref: '#/components/responses/Session_d5d120e942d17641Status200Response'`)
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
	requirePattern(`(?m)^\s+x-goa-async:$`)
	requirePattern(`(?m)^\s+transport: sse$`)
	requirePattern(`(?m)^\s+transport: websocket$`)
	requirePattern(`(?m)^\s+direction: server$`)
	requirePattern(`(?m)^\s+direction: bidirectional$`)
	requirePattern(`(?m)^\s+contentType: text/event-stream$`)
	requirePattern(`(?m)^\s+status: 101$`)
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
	parsed, err := libopenapi.NewDocument(b)
	if err != nil {
		t.Fatalf("libopenapi failed to parse generated spec: %s\nspec:\n%s", err, string(b))
	}
	if parsed.GetVersion() != openapiv3.OpenAPIVersion {
		t.Fatalf("libopenapi parsed version %q, expected %q", parsed.GetVersion(), openapiv3.OpenAPIVersion)
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
		if len(o.SectionTemplates) != 1 {
			t.Fatalf("expected 1 section for %s, got %d", o.Path, len(o.SectionTemplates))
		}
		var buf bytes.Buffer
		section := o.SectionTemplates[0]
		tmpl := template.Must(template.New("openapi").Funcs(section.FuncMap).Parse(section.Source))
		if err := tmpl.Execute(&buf, section.Data); err != nil {
			t.Fatalf("failed to render template: %s", err)
		}
		validateOpenAPI(t, buf.Bytes())
		return buf.String()
	}

	t.Fatal("missing YAML OpenAPI output")
	return ""
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
