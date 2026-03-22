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
	"text/template"

	"github.com/stretchr/testify/require"

	httpgen "github.com/CaliLuke/loom/http/codegen"
	"github.com/CaliLuke/loom/http/codegen/openapi"
	openapiv3 "github.com/CaliLuke/loom/http/codegen/openapi/v3"
	"github.com/CaliLuke/loom/http/codegen/testdata"
	"github.com/CaliLuke/loom/internal/testingx"
)

const (
	redoclyCLIVersion        = "2.24.1"
	openAPITypescriptVersion = "7.13.0"
	typeScriptVersion        = "5.9.3"
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
	}

	for _, tc := range cases {
		tc := tc
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
	lintCases := []struct {
		name string
		dsl  func()
	}{
		{name: "meal-planner", dsl: testdata.MealPlannerDSL},
		{name: "problem-links-async", dsl: testdata.OpenAPIProblemLinksAsyncDSL},
	}
	for _, tc := range lintCases {
		tc := tc
		t.Run("redocly-"+tc.name, func(t *testing.T) {
			artifacts := renderOpenAPIArtifacts(t, tc.dsl)
			workDir := filepath.Join(t.TempDir(), tc.name)
			require.NoError(t, os.MkdirAll(workDir, 0o750))
			yamlPath := filepath.Join(workDir, "openapi.yaml")
			require.NoError(t, os.WriteFile(yamlPath, artifacts.YAML, 0o600))
			_, err := testingx.RunCmd(workDir, "npx", "--yes", "@redocly/cli@"+redoclyCLIVersion, "lint", yamlPath)
			require.NoError(t, err)
		})
	}

	artifacts := renderOpenAPIArtifacts(t, testdata.OpenAPIProblemLinksAsyncDSL)
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

func renderOpenAPIArtifacts(t *testing.T, dsl func()) renderedOpenAPIArtifacts {
	t.Helper()

	openapi.Definitions = make(map[string]*openapi.Schema)
	root := httpgen.RunHTTPDSL(t, dsl)
	oFiles, err := openapiv3.Files(root)
	require.NoError(t, err)

	var artifacts renderedOpenAPIArtifacts
	for _, file := range oFiles {
		require.Len(t, file.SectionTemplates, 1)
		section := file.SectionTemplates[0]
		var buf bytes.Buffer
		tmpl := template.Must(template.New("openapi").Funcs(section.FuncMap).Parse(section.Source))
		require.NoError(t, tmpl.Execute(&buf, section.Data))
		validateOpenAPI(t, buf.Bytes())
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
