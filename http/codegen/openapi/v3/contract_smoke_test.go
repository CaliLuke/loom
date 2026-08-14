package openapiv3_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	httpgen "github.com/CaliLuke/loom/http/codegen"
	"github.com/CaliLuke/loom/http/codegen/openapi"
	openapiv3 "github.com/CaliLuke/loom/http/codegen/openapi/v3"
	"github.com/CaliLuke/loom/http/codegen/testdata"
	"github.com/CaliLuke/loom/internal/testingx"
)

const (
	redoclyCLIVersion        = "2.24.1"
	heyAPIOpenAPIVersion     = "0.99.0"
	heyAPIClientFetchVersion = "0.13.1"
	openAPITypescriptVersion = "7.13.0"
	typeScriptVersion        = "5.9.3"
	tanStackQueryVersion     = "5.101.4"
	zodVersion               = "4.4.3"
	reactVersion             = "19.2.8"
	oapiCodegenVersion       = "v2.6.0"
	openAPIAsyncExtension    = "x-loom-async"
)

type renderedOpenAPIArtifacts struct {
	JSON []byte
	YAML []byte
}

func TestRenderedSpecsPassContractLint(t *testing.T) {
	cases := []struct {
		name  string
		dsl   func()
		extra func(*testing.T, map[string]any)
	}{
		{
			name: "meal-planner",
			dsl:  testdata.MealPlannerDSL,
		},
		{
			name: "file-response",
			dsl:  fileResponseOpenAPIDSL,
		},
		{
			name: "raw-request-bodies",
			dsl:  testdata.RawRequestBodyOpenAPIDSL,
			extra: func(t *testing.T, spec map[string]any) {
				binary := requireOperation(t, spec, "/uploads/{id}", "post")
				binaryBody := requireMap(t, binary["requestBody"], "binary request body")
				require.Equal(t, true, binaryBody["required"])
				binaryContent := requireMap(t, binaryBody["content"], "binary request content")
				require.Contains(t, binaryContent, "application/octet-stream")

				text := requireOperation(t, spec, "/imports", "post")
				textBody := requireMap(t, text["requestBody"], "text request body")
				require.NotContains(t, textBody, "required")
				textContent := requireMap(t, textBody["content"], "text request content")
				require.Contains(t, textContent, "text/plain; charset=utf-8")
			},
		},
		{
			name: "request-response-split",
			dsl:  testdata.OpenAPIRequestResponseSplitDSL,
			extra: func(t *testing.T, spec map[string]any) {
				requestSchema := requireComponentSchema(t, spec, "CreateRequestBodyRequest")
				responseSchema := requireComponentSchema(t, spec, "CreateResponseBodyResponse")
				require.Contains(t, requestSchema["properties"].(map[string]any), "password")
				require.NotContains(t, responseSchema["properties"].(map[string]any), "password")
				require.Contains(t, responseSchema["properties"].(map[string]any), "id")
				require.NotContains(t, requestSchema["properties"].(map[string]any), "id")
			},
		},
		{
			name: "problem-links-async",
			dsl:  testdata.OpenAPIProblemLinksAsyncDSL,
			extra: func(t *testing.T, spec map[string]any) {
				requireComponentSchema(t, spec, "Problem")
				response := requireComponentResponse(t, spec, "OpenAPIThreadAcceptedStatus202Response")
				links := requireMap(t, response["links"], "response links")
				thread := requireMap(t, links["thread"], "thread link")
				watch := requireMap(t, links["watch"], "watch link")
				require.Equal(t, "thread_ops.get_thread", requireString(t, thread["operationId"], "thread link operationId"))
				require.Equal(t, "thread_ops.watch_thread", requireString(t, watch["operationId"], "watch link operationId"))
			},
		},
		{
			name: "shared-error-responses",
			dsl:  testdata.OpenAPISharedErrorHeaderDSL,
			extra: func(t *testing.T, spec map[string]any) {
				components := requireMap(t, spec["components"], "components")
				schemas := requireMap(t, components["schemas"], "components.schemas")
				variantCount := 0
				for name := range schemas {
					if strings.HasPrefix(name, "ExceptionResponse_") {
						variantCount++
					}
				}
				require.Equal(t, 1, variantCount)
			},
		},
		{
			name: "async-session-security",
			dsl:  testdata.AsyncSessionSecurityDSL,
			extra: func(t *testing.T, spec map[string]any) {
				components := requireMap(t, spec["components"], "components")
				securitySchemes := requireMap(t, components["securitySchemes"], "components.securitySchemes")
				cookieScheme := requireMap(t, securitySchemes["browser_session_cookie"], "browser session security scheme")
				require.Equal(t, "apiKey", requireString(t, cookieScheme["type"], "cookie scheme type"))
				require.Equal(t, "cookie", requireString(t, cookieScheme["in"], "cookie scheme in"))
				require.Equal(t, "__Host-ak_session", requireString(t, cookieScheme["name"], "cookie scheme name"))

				socketOperation := requireOperation(t, spec, "/ws/projects/{project_id}", "get")
				require.Nil(t, socketOperation["requestBody"])
				requireOperationSecurity(t, socketOperation, "browser_session_cookie")
				requireOperationParameter(t, spec, socketOperation, "project_id", "path")
				socketAsync := requireMap(t, socketOperation[openAPIAsyncExtension], "websocket async contract")
				require.Equal(t, "websocket", requireString(t, socketAsync["transport"], "websocket async transport"))
				require.Equal(t, "bidirectional", requireString(t, socketAsync["direction"], "websocket async direction"))
				socketHandshake := requireMap(t, socketAsync["handshake"], "websocket handshake")
				socketRequest := requireMap(t, socketHandshake["request"], "websocket handshake request")
				require.Equal(t, "GET", requireString(t, socketRequest["method"], "websocket handshake method"))
				socketMessages := requireMap(t, socketAsync["messages"], "websocket messages")
				require.Contains(t, socketMessages, "inbound")
				require.Contains(t, socketMessages, "outbound")

				eventsOperation := requireOperation(t, spec, "/events/{project_id}", "get")
				require.Nil(t, eventsOperation["requestBody"])
				requireOperationSecurity(t, eventsOperation, "browser_session_cookie")
				requireOperationParameter(t, spec, eventsOperation, "project_id", "path")
				requireOperationParameter(t, spec, eventsOperation, "last_event_id", "query")
				eventsAsync := requireMap(t, eventsOperation[openAPIAsyncExtension], "sse async contract")
				require.Equal(t, "sse", requireString(t, eventsAsync["transport"], "sse async transport"))
				require.Equal(t, "server", requireString(t, eventsAsync["direction"], "sse async direction"))
				eventsHandshake := requireMap(t, eventsAsync["handshake"], "sse handshake")
				eventsResponse := requireMap(t, eventsHandshake["response"], "sse handshake response")
				require.Equal(t, 200, requireInt(t, eventsResponse["status"], "sse handshake status"))
				require.Equal(t, "text/event-stream", requireString(t, eventsResponse["contentType"], "sse handshake content type"))
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			artifacts := renderOpenAPIArtifacts(t, tc.dsl)
			spec := decodeOpenAPIJSON(t, artifacts.JSON)
			lintRenderedContract(t, spec)
			if tc.extra != nil {
				tc.extra(t, spec)
			}
		})
	}
}

func TestRepresentativeSpecsPassRedoclyLintAndConsumerSmoke(t *testing.T) {
	if os.Getenv("LOOM_OPENAPI_CONTRACT") == "" {
		t.Skip("consumer smoke needs node/npx and network access; run via 'make openapi-contract'")
	}
	lintCases := []struct {
		name                       string
		dsl                        func()
		unsupportedOAS32Diagnostic string
	}{
		{name: "meal-planner", dsl: testdata.MealPlannerDSL},
		{
			name:                       "openapi-3.2-features",
			dsl:                        testdata.OpenAPI32FeaturesDSL,
			unsupportedOAS32Diagnostic: "Unknown type name found: string",
		},
		{name: "file-response", dsl: fileResponseOpenAPIDSL},
		{name: "raw-request-bodies", dsl: testdata.RawRequestBodyOpenAPIDSL},
		{name: "problem-links-async", dsl: testdata.OpenAPIProblemLinksAsyncDSL},
		{name: "shared-error-responses", dsl: testdata.OpenAPISharedErrorHeaderDSL},
	}
	for _, tc := range lintCases {
		t.Run("redocly-"+tc.name, func(t *testing.T) {
			artifacts := renderOpenAPIArtifacts(t, tc.dsl)
			workDir := filepath.Join(t.TempDir(), tc.name)
			require.NoError(t, os.MkdirAll(workDir, 0o750))
			yamlPath := filepath.Join(workDir, "openapi.yaml")
			require.NoError(t, os.WriteFile(yamlPath, artifacts.YAML, 0o600))
			_, err := testingx.RunCmd(workDir, "npx", "--yes", "@redocly/cli@"+redoclyCLIVersion, "lint", yamlPath)
			if tc.unsupportedOAS32Diagnostic != "" {
				// renderOpenAPIArtifacts has already parsed and built both valid
				// artifacts with libopenapi. This diagnostic is Redocly's current
				// lack of OpenAPI 3.2 support, not a Loom rendering failure.
				require.ErrorContains(t, err, tc.unsupportedOAS32Diagnostic)
				return
			}
			require.NoError(t, err)
		})
	}

	// The pinned Redocly CLI above rejects the canonical 3.2 feature specimen
	// before linting it; keep asserting that exact limitation so the specimen is
	// not silently omitted. The pinned downstream generators below likewise
	// reject the 3.2 version string before processing the otherwise compatible
	// document, so exercise the renderer's 3.1 compatibility target with the
	// existing non-trivial raw-body specimen.
	artifacts := renderOpenAPIArtifactsForVersion(
		t,
		testdata.RawRequestBodyOpenAPIDSL,
		"3.1",
		openapiv3.OpenAPICompatibilityVersion,
	)
	workDir := filepath.Join(t.TempDir(), "contract-smoke")
	require.NoError(t, os.MkdirAll(workDir, 0o750))

	yamlPath := filepath.Join(workDir, "openapi.yaml")
	jsonPath := filepath.Join(workDir, "openapi.json")
	require.NoError(t, os.WriteFile(yamlPath, artifacts.YAML, 0o600))
	require.NoError(t, os.WriteFile(jsonPath, artifacts.JSON, 0o600))

	_, err := testingx.RunCmd(workDir, "npx", "--yes", "--package=openapi-typescript@"+openAPITypescriptVersion, "--package=typescript@"+typeScriptVersion, "openapi-typescript", yamlPath, "-o", "openapi.d.ts")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(workDir, "index.ts"), []byte("import type { paths } from \"./openapi\";\n\ntype SmokePaths = paths;\nconst smoke: SmokePaths | undefined = undefined;\nexport default smoke;\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(workDir, "tsconfig.json"), []byte("{\n  \"compilerOptions\": {\n    \"target\": \"ES2020\",\n    \"module\": \"ESNext\",\n    \"moduleResolution\": \"Bundler\",\n    \"strict\": true,\n    \"skipLibCheck\": true,\n    \"lib\": [\"ES2020\", \"DOM\"]\n  },\n  \"include\": [\"index.ts\", \"openapi.d.ts\"]\n}\n"), 0o600))
	_, err = testingx.RunCmd(workDir, "npx", "--yes", "--package=typescript@"+typeScriptVersion, "tsc", "--noEmit", "--project", "tsconfig.json")
	require.NoError(t, err)
	smokeHeyAPITypeScriptClient(t, workDir, jsonPath)

	goDir := filepath.Join(workDir, "go-smoke")
	require.NoError(t, os.MkdirAll(goDir, 0o750))
	_, err = testingx.RunCmd(goDir, "go", "mod", "init", "example.com/openapi-smoke")
	require.NoError(t, err)
	_, err = testingx.RunCmd(goDir, "go", "run", "github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@"+oapiCodegenVersion, "-generate", "types,client", "-package", "smoke", "-o", "client.gen.go", yamlPath)
	require.NoError(t, err)
	_, err = testingx.RunCmd(goDir, "go", "mod", "tidy")
	require.NoError(t, err)
	_, err = testingx.RunCmd(goDir, "go", "build", "./...")
	require.NoError(t, err)
}

func smokeHeyAPITypeScriptClient(t *testing.T, workDir, jsonPath string) {
	t.Helper()
	heyDir := filepath.Join(workDir, "hey-api-smoke")
	require.NoError(t, os.MkdirAll(heyDir, 0o750))
	_, err := testingx.RunCmd(
		heyDir,
		"npm",
		"install",
		"--ignore-scripts",
		"--save-exact",
		"typescript@"+typeScriptVersion,
		"@hey-api/openapi-ts@"+heyAPIOpenAPIVersion,
		"@hey-api/client-fetch@"+heyAPIClientFetchVersion,
		"@tanstack/react-query@"+tanStackQueryVersion,
		"zod@"+zodVersion,
		"react@"+reactVersion,
	)
	require.NoError(t, err)

	config := `export default {
  input: ` + strconv.Quote(jsonPath) + `,
  output: "src/client",
  plugins: [
    { name: "@hey-api/client-fetch", baseUrl: false, throwOnError: true },
    { name: "@hey-api/typescript" },
    { name: "@hey-api/sdk", validator: { response: "zod" } },
    { name: "zod" },
    {
      name: "@tanstack/react-query",
      queryOptions: true,
      mutationOptions: true,
      queryKeys: { tags: true },
    },
  ],
};
`
	require.NoError(t, os.WriteFile(filepath.Join(heyDir, "openapi-ts.config.mjs"), []byte(config), 0o600))
	_, err = testingx.RunCmd(heyDir, filepath.Join(heyDir, "node_modules", ".bin", "openapi-ts"))
	require.NoError(t, err)

	tsconfig := `{
  "compilerOptions": {
    "target": "ES2022",
    "module": "ESNext",
    "moduleResolution": "Bundler",
    "strict": true,
    "skipLibCheck": true,
    "lib": ["ES2022", "DOM", "DOM.Iterable"]
  },
  "include": ["src/**/*.ts"]
}
`
	require.NoError(t, os.WriteFile(filepath.Join(heyDir, "tsconfig.json"), []byte(tsconfig), 0o600))
	_, err = testingx.RunCmd(heyDir, filepath.Join(heyDir, "node_modules", ".bin", "tsc"), "--noEmit", "--project", "tsconfig.json")
	require.NoError(t, err)
}

func renderOpenAPIArtifacts(t *testing.T, dsl func()) renderedOpenAPIArtifacts {
	t.Helper()
	return renderOpenAPIArtifactsForVersion(t, dsl, "", openapiv3.OpenAPIVersion)
}

func renderOpenAPIArtifactsForVersion(t *testing.T, dsl func(), target, want string) renderedOpenAPIArtifacts {
	t.Helper()

	openapi.Definitions = make(map[string]*openapi.Schema)
	root := httpgen.RunHTTPDSL(t, dsl)
	if target != "" {
		if root.API.Meta == nil {
			root.API.Meta = make(map[string][]string)
		}
		root.API.Meta["openapi:version"] = []string{target}
	}
	oFiles, err := openapiv3.Files(root)
	require.NoError(t, err)

	var artifacts renderedOpenAPIArtifacts
	for _, file := range oFiles {
		sections := file.AllSections()
		require.Len(t, sections, 1)
		section := sections[0]
		var buf bytes.Buffer
		require.NoError(t, section.Write(&buf))
		validateOpenAPIVersion(t, buf.Bytes(), want)
		switch filepath.Ext(file.Path) {
		case ".json":
			artifacts.JSON = append([]byte(nil), buf.Bytes()...)
		case ".yaml":
			artifacts.YAML = append([]byte(nil), buf.Bytes()...)
		default:
			t.Fatalf("unexpected OpenAPI artifact %q", file.Path)
		}
	}

	require.NotEmpty(t, artifacts.JSON)
	require.NotEmpty(t, artifacts.YAML)
	return artifacts
}

func decodeOpenAPIJSON(t *testing.T, raw []byte) map[string]any {
	t.Helper()

	var spec map[string]any
	require.NoError(t, json.Unmarshal(raw, &spec))
	return spec
}

func lintRenderedContract(t *testing.T, spec map[string]any) {
	t.Helper()

	operationIDRE := regexp.MustCompile(`^[a-z0-9_.]+$`)
	topLevelTags := make(map[string]struct{})
	if tagsRaw, ok := spec["tags"]; ok {
		for _, tagRaw := range requireSlice(t, tagsRaw, "top-level tags") {
			tag := requireMap(t, tagRaw, "tag")
			topLevelTags[requireString(t, tag["name"], "tag name")] = struct{}{}
		}
	}

	paths := requireMap(t, spec["paths"], "paths")
	for path, pathRaw := range paths {
		pathItem := requireMap(t, pathRaw, "path item")
		for method, operationRaw := range pathItem {
			if !isHTTPOperation(method) {
				continue
			}
			operation := requireMap(t, operationRaw, "operation")
			operationID := requireString(t, operation["operationId"], "operationId")
			if !operationIDRE.MatchString(operationID) {
				t.Fatalf("operation %s %s has unsafe operationId %q", strings.ToUpper(method), path, operationID)
			}
			if tagsRaw, ok := operation["tags"]; ok {
				for _, tagRaw := range requireSlice(t, tagsRaw, "operation tags") {
					tag := requireString(t, tagRaw, "operation tag")
					if _, ok := topLevelTags[tag]; !ok {
						t.Fatalf("operation %s %s uses undeclared tag %q", strings.ToUpper(method), path, tag)
					}
				}
			}
			if asyncRaw, ok := operation[openAPIAsyncExtension]; ok {
				lintAsyncContract(t, path, strings.ToUpper(method), requireMap(t, asyncRaw, "async contract"))
			}
		}
	}

	walkRefs(spec, func(ref string) {
		if !strings.HasPrefix(ref, "#/") {
			return
		}
		if !refExists(spec, ref) {
			t.Fatalf("spec contains unresolved ref %q", ref)
		}
	})
}

func lintAsyncContract(t *testing.T, path string, method string, async map[string]any) {
	t.Helper()

	transport := requireString(t, async["transport"], "async transport")
	handshake := requireMap(t, async["handshake"], "async handshake")
	response := requireMap(t, handshake["response"], "async handshake response")
	status := requireInt(t, response["status"], "async handshake status")
	contentType := requireString(t, response["contentType"], "async handshake content type")

	switch transport {
	case "sse":
		require.Equal(t, 200, status, "SSE %s %s must publish a 200 handshake", method, path)
		require.Equal(t, "text/event-stream", contentType, "SSE %s %s must publish text/event-stream", method, path)
	case "websocket":
		require.Equal(t, 101, status, "WebSocket %s %s must publish a 101 handshake", method, path)
		require.Equal(t, "", contentType, "WebSocket %s %s must not publish a handshake content type", method, path)
	default:
		t.Fatalf("operation %s %s has unknown async transport %q", method, path, transport)
	}
}

func walkRefs(node any, visit func(string)) {
	switch actual := node.(type) {
	case map[string]any:
		for key, value := range actual {
			if key == "$ref" {
				ref, ok := value.(string)
				if !ok {
					continue
				}
				visit(ref)
				continue
			}
			walkRefs(value, visit)
		}
	case []any:
		for _, value := range actual {
			walkRefs(value, visit)
		}
	}
}

func refExists(spec map[string]any, ref string) bool {
	if !strings.HasPrefix(ref, "#/") {
		return false
	}
	current := any(spec)
	for _, rawToken := range strings.Split(strings.TrimPrefix(ref, "#/"), "/") {
		token := strings.ReplaceAll(strings.ReplaceAll(rawToken, "~1", "/"), "~0", "~")
		next, ok := current.(map[string]any)
		if !ok {
			return false
		}
		current, ok = next[token]
		if !ok {
			return false
		}
	}
	return true
}

func requireComponentSchema(t *testing.T, spec map[string]any, name string) map[string]any {
	t.Helper()
	components := requireMap(t, spec["components"], "components")
	schemas := requireMap(t, components["schemas"], "components.schemas")
	schema, ok := schemas[name]
	require.Truef(t, ok, "missing schema component %q", name)
	return requireMap(t, schema, "schema component")
}

func requireComponentResponse(t *testing.T, spec map[string]any, name string) map[string]any {
	t.Helper()
	components := requireMap(t, spec["components"], "components")
	responses := requireMap(t, components["responses"], "components.responses")
	response, ok := responses[name]
	require.Truef(t, ok, "missing response component %q", name)
	return requireMap(t, response, "response component")
}

func requireOperation(t *testing.T, spec map[string]any, path string, method string) map[string]any {
	t.Helper()

	paths := requireMap(t, spec["paths"], "paths")
	pathItem := requireMap(t, paths[path], "path item")
	return requireMap(t, pathItem[method], "operation")
}

func requireOperationSecurity(t *testing.T, operation map[string]any, schemeName string) {
	t.Helper()

	security := requireSlice(t, operation["security"], "operation security")
	require.NotEmpty(t, security)
	found := false
	for _, raw := range security {
		requirement := requireMap(t, raw, "security requirement")
		if _, ok := requirement[schemeName]; ok {
			found = true
			break
		}
	}
	require.Truef(t, found, "operation security missing scheme %q", schemeName)
}

func requireOperationParameter(t *testing.T, spec map[string]any, operation map[string]any, name string, in string) {
	t.Helper()

	parameters := requireSlice(t, operation["parameters"], "operation parameters")
	for _, raw := range parameters {
		parameter := requireMap(t, raw, "parameter")
		if ref, ok := parameter["$ref"].(string); ok {
			parameter = resolveRefMap(t, spec, ref)
		}
		if requireString(t, parameter["name"], "parameter name") != name {
			continue
		}
		require.Equal(t, in, requireString(t, parameter["in"], "parameter in"))
		return
	}
	t.Fatalf("operation missing %s parameter %q", in, name)
}

func resolveRefMap(t *testing.T, spec map[string]any, ref string) map[string]any {
	t.Helper()

	if !strings.HasPrefix(ref, "#/") {
		t.Fatalf("unsupported external ref %q", ref)
	}
	current := any(spec)
	for _, rawToken := range strings.Split(strings.TrimPrefix(ref, "#/"), "/") {
		token := strings.ReplaceAll(strings.ReplaceAll(rawToken, "~1", "/"), "~0", "~")
		current = requireMap(t, current, "ref segment")[token]
	}
	return requireMap(t, current, "resolved ref")
}

func requireMap(t *testing.T, value any, label string) map[string]any {
	t.Helper()
	mapped, ok := value.(map[string]any)
	require.Truef(t, ok, "%s should be a map, got %T", label, value)
	return mapped
}

func requireSlice(t *testing.T, value any, label string) []any {
	t.Helper()
	items, ok := value.([]any)
	require.Truef(t, ok, "%s should be a slice, got %T", label, value)
	return items
}

func requireString(t *testing.T, value any, label string) string {
	t.Helper()
	str, ok := value.(string)
	require.Truef(t, ok, "%s should be a string, got %T", label, value)
	return str
}

func requireInt(t *testing.T, value any, label string) int {
	t.Helper()
	switch actual := value.(type) {
	case float64:
		return int(actual)
	case int:
		return actual
	case json.Number:
		parsed, err := strconv.Atoi(actual.String())
		require.NoError(t, err)
		return parsed
	default:
		t.Fatalf("%s should be a number, got %T", label, value)
		return 0
	}
}

func isHTTPOperation(method string) bool {
	switch strings.ToLower(method) {
	case "get", "put", "post", "delete", "options", "head", "patch", "trace":
		return true
	default:
		return false
	}
}
