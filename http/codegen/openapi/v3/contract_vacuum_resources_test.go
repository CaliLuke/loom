package openapiv3_test

import (
	"encoding/json/v2"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
	"github.com/CaliLuke/loom/internal/testingx"
)

func TestContractVacuumResourceBackend(t *testing.T) {
	if os.Getenv("LOOM_OPENAPI_CONTRACT") == "" {
		t.Skip("requires pinned Vacuum; run with LOOM_OPENAPI_CONTRACT=1")
	}
	vacuum := installContractVacuum(t)
	artifacts := renderOpenAPIArtifacts(t, testdata.OpenAPI32FeaturesDSL)
	t.Run("ordinary-and-invalid-parity", func(t *testing.T) {
		cases := append([]contractVacuumInvalidCase{{name: "ordinary"}}, vacuumInvalidContracts()...)
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				spec := decodeOpenAPIJSON(t, artifacts.JSON)
				if tc.mutate != nil {
					tc.mutate(t, spec)
				}
				entry := writeVacuumJSON(t, spec)
				report, motorErr := vacuum.motor(entry)
				output := filepath.Join(t.TempDir(), "cli.json")
				_, cliErr := testingx.RunCmd(filepath.Dir(entry), vacuum.binary, "lint", "--no-update-check", "--no-banner", "--no-style", "--errors", "--fail-severity", "error", "--ruleset", vacuum.ruleset, "--format", "json", "--output", output, entry)
				require.Equal(t, cliErr == nil, motorErr == nil)
				var cli []struct {
					Code  string `json:"code"`
					Range struct {
						Start struct {
							Line int `json:"line"`
							Char int `json:"character"`
						} `json:"start"`
					} `json:"range"`
				}
				b, err := os.ReadFile(output)
				require.NoError(t, err)
				require.NoError(t, json.Unmarshal(b, &cli))
				want := make([]string, 0, len(cli))
				for _, row := range cli {
					want = append(want, fmt.Sprintf("%s:%d:%d", row.Code, row.Range.Start.Line, row.Range.Start.Char))
				}
				got := make([]string, 0)
				for _, row := range report.Findings {
					if row.Severity == "error" {
						got = append(got, fmt.Sprintf("%s:%d:%d", row.Rule, row.Line, row.Column))
					}
				}
				slices.Sort(want)
				slices.Sort(got)
				require.Equal(t, want, got)
				if tc.name == "ordinary" {
					require.NoError(t, motorErr)
				} else {
					require.Error(t, motorErr)
					require.NotEmpty(t, got)
				}
				logVacuumReport(t, report)
			})
		}
	})
	t.Run("explicit-json-and-yaml-resources", func(t *testing.T) {
		entry := writeVacuumBytes(t, artifacts.JSON)
		for _, tc := range []struct {
			name string
			data []byte
		}{{"yaml", []byte("openapi: 3.2.0\ninfo: {title: Shared, version: '1'}\npaths: {}\n")}, {"json", []byte(`{"openapi":"3.2.0","info":{"title":"Shared","version":"1","description":"JSON escaped slash \/"},"paths":{}}`)}} {
			t.Run(tc.name, func(t *testing.T) {
				doc := contractDocument{URI: "https://resources.test/shared." + tc.name, Path: writeVacuumBytes(t, tc.data)}
				report, err := vacuum.motor(entry, doc)
				require.NoError(t, err)
				require.Len(t, report.Resources, 2)
				for _, resource := range report.Resources {
					require.Equal(t, doc.URI, resource.URI)
					require.NotEmpty(t, resource.ContentSHA256)
				}
				require.NoError(t, vacuum.lint(entry, doc))
				logVacuumReport(t, report)
			})
		}
	})
	t.Run("used-resource-with-references", func(t *testing.T) {
		spec := decodeOpenAPIJSON(t, []byte(`{"openapi": "3.2.0", "info": {"title": "References", "version": "1"}, "paths": {"/items": {"get": {"operationId": "items", "responses": {"200": {"description": "Success", "content": {"application/json": {"schema": {"$ref": "https://resources.test/shared.yaml#/components/schemas/Wrapper"}}}}}}}}}`))
		resource := []byte(`{"openapi":"3.2.0","info":{"title":"Shared","version":"1"},"paths":{},"components":{"schemas":{"Value":{"type":"string"},"Wrapper":{"type":"object","properties":{"value":{"$ref":"#/components/schemas/Value"}}}}}}`)
		doc := contractDocument{URI: "https://resources.test/shared.yaml", Path: writeVacuumBytes(t, resource)}
		report, err := vacuum.motor(writeVacuumJSON(t, spec), doc)
		logVacuumReport(t, report)
		require.NoError(t, err)
		require.Len(t, report.Resources, 2)
		for _, admitted := range report.Resources {
			require.Equal(t, doc.URI, admitted.URI)
			require.NotEmpty(t, admitted.ContentSHA256)
			if admitted.Role == "resolved" {
				require.NotEqual(t, admitted.ContentSHA256, admitted.IndexedContentSHA256, "used nested reference must resolve")
			} else {
				require.Equal(t, "unresolved", admitted.Role)
				require.Equal(t, admitted.ContentSHA256, admitted.IndexedContentSHA256)
			}
		}
	})
	t.Run("invalid-resource-inputs", func(t *testing.T) {
		entry := writeVacuumBytes(t, artifacts.JSON)
		good := contractDocument{URI: "https://resources.test/shared.yaml", Path: writeVacuumBytes(t, []byte("openapi: 3.2.0\ninfo: {title: Shared, version: '1'}\npaths: {}\n"))}
		for _, tc := range []struct {
			name    string
			docs    []contractDocument
			message string
		}{
			{"duplicate", []contractDocument{good, good}, "duplicate"},
			{"equivalent-duplicate", []contractDocument{good, {URI: "https://RESOURCES.test/./shared.yaml", Path: good.Path}}, "duplicate"},
			{"relative-uri", []contractDocument{{URI: "shared.yaml", Path: good.Path}}, "absolute"},
			{"fragment-uri", []contractDocument{{URI: good.URI + "#/info", Path: good.Path}}, "fragment"},
			{"malformed-yaml", []contractDocument{{URI: good.URI, Path: writeVacuumBytes(t, []byte("{broken:"))}}, "parse"},
			{"empty-document", []contractDocument{{URI: good.URI, Path: writeVacuumBytes(t, nil)}}, "parse"},
			{"missing-file", []contractDocument{{URI: good.URI, Path: filepath.Join(t.TempDir(), "absent.yaml")}}, "read"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				report, err := vacuum.motor(entry, tc.docs...)
				require.Error(t, err)
				require.Contains(t, strings.Join(report.Errors, " "), tc.message)
				logVacuumReport(t, report)
			})
		}
	})
	t.Run("unadmitted-reference-is-an-error", func(t *testing.T) {
		spec := decodeOpenAPIJSON(t, artifacts.JSON)
		schemas := requireMap(t, requireMap(t, spec["components"], "components")["schemas"], "schemas")
		schemas["CatalogCreated"] = map[string]any{"$ref": "https://not-admitted.test/schemas.yaml#/Missing"}
		report, err := vacuum.motor(writeVacuumJSON(t, spec))
		require.Error(t, err)
		require.Contains(t, fmt.Sprint(report.Findings), "resolving-references")
		logVacuumReport(t, report)
	})

	t.Run("motor-preparation-errors-fail", func(t *testing.T) {
		report, err := vacuum.motor(writeVacuumBytes(t, []byte("{broken:")))
		require.Error(t, err)
		require.NotEmpty(t, report.Errors)
		logVacuumReport(t, report)
	})

	t.Run("security-negative-and-name-controls", func(t *testing.T) {
		security := []byte("openapi: 3.2.0\ninfo: {title: Shared, version: '1'}\npaths: {}\ncomponents:\n  securitySchemes:\n    ApiKey: {type: apiKey, in: header, name: X-Key}\n")
		target := contractDocument{URI: "https://auth.test/security.yaml", Path: writeVacuumBytes(t, security)}
		for _, tc := range []struct {
			name, key string
			docs      []contractDocument
			local     bool
			valid     bool
		}{
			{"absent-target", "https://auth.test/security.yaml#/components/securitySchemes/ApiKey", nil, false, false},
			{"missing-scheme", "https://auth.test/security.yaml#/components/securitySchemes/Missing", []contractDocument{target}, false, false},
			{"uri-like-local-name", "security.yaml", nil, true, true},
			{"local-name-before-loaded-non-scheme-document", "security.yaml", []contractDocument{{URI: "https://entry.test/security.yaml", Path: target.Path}}, true, true},
		} {
			t.Run(tc.name, func(t *testing.T) {
				spec := vacuumSecurityDocument(tc.key)
				if tc.local {
					spec["components"] = map[string]any{"securitySchemes": map[string]any{tc.key: map[string]any{"type": "apiKey", "in": "header", "name": "X-Local"}}}
				}
				report, err := vacuum.motor(writeVacuumJSON(t, spec), tc.docs...)
				if tc.valid {
					require.NoError(t, err)
				} else {
					require.Error(t, err)
					require.Contains(t, fmt.Sprint(report.Findings), "oas3-operation-security-defined")
				}
				require.Len(t, report.Resources, 2*len(tc.docs))
				logVacuumReport(t, report)
			})
		}
	})
}

// TestExternalSecurityDocumentsPassVacuum checks the external security correctness pin.
// It validates the unchanged generated document with its admitted external security resource.
func TestExternalSecurityDocumentsPassVacuum(t *testing.T) {
	if os.Getenv("LOOM_OPENAPI_CONTRACT") == "" {
		t.Skip("requires pinned Vacuum; run with LOOM_OPENAPI_CONTRACT=1")
	}
	vacuum := installContractVacuum(t)
	entry := os.Getenv("LOOM_EXTERNAL_SECURITY_SPEC")
	target := os.Getenv("LOOM_EXTERNAL_SECURITY_DOCUMENT")
	require.NotEmpty(t, entry, "supply unchanged retained symmetry_identity.json")
	require.NotEmpty(t, target, "supply exact remote/security.yaml")
	report, err := vacuum.motor(entry, contractDocument{URI: "https://auth.test/security.yaml", Path: target})
	require.Len(t, report.Resources, 2)
	logVacuumReport(t, report)
	require.NoError(t, err, "external security references must validate with admitted resources")
}

func vacuumSecurityDocument(key string) map[string]any {
	return map[string]any{"openapi": "3.2.0", "$self": "https://entry.test/api.yaml", "info": map[string]any{"title": "Security control", "version": "1"}, "paths": map[string]any{"/items": map[string]any{"get": map[string]any{"operationId": "items", "security": []any{map[string]any{key: []string{}}}, "responses": map[string]any{"204": map[string]any{"description": "Success"}}}}}}
}
func writeVacuumJSON(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value, json.Deterministic(true))
	require.NoError(t, err)
	return writeVacuumBytes(t, data)
}
func writeVacuumBytes(t *testing.T, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "input.json")
	require.NoError(t, os.WriteFile(path, data, 0o600))
	return path
}
func logVacuumReport(t *testing.T, report contractVacuumReport) {
	t.Helper()
	data, err := json.Marshal(report, json.Deterministic(true))
	require.NoError(t, err)
	t.Log(string(data))
}
