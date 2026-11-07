package openapiimport

import (
	"encoding/json/v2"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xeipuuv/gojsonschema"
)

type oneOfSemanticCase struct {
	wire  string
	valid bool
}

// These expectations are written from the source's exact-one rule, independently
// of either importer output or the validator. int32/int64 formats do not narrow
// the JSON Schema integer domain. Closure is deliberately excluded here.
func oneOfSemanticCases(name string) []oneOfSemanticCase {
	switch name {
	case "01-scalar":
		return []oneOfSemanticCase{{`null`, false}, {`"a"`, true}, {`1`, true}, {`1.5`, false}, {`{}`, false}}
	case "02-inline-array":
		return []oneOfSemanticCase{{`["a"]`, true}, {`[1]`, false}, {`[]`, true}, {`"a"`, true}, {`null`, false}}
	case "04-integer-number":
		return []oneOfSemanticCase{{`null`, false}, {`1`, false}, {`2147483648`, false}, {`1.5`, true}, {`"a"`, false}}
	case "05-overlap-third":
		return []oneOfSemanticCase{{`"a"`, false}, {`1`, true}, {`1.5`, false}, {`null`, false}}
	case "06-any", "12-any-first":
		return []oneOfSemanticCase{{`null`, true}, {`"a"`, false}, {`1`, true}, {`{}`, true}, {`[]`, true}}
	case "07-nullable-branch":
		return []oneOfSemanticCase{{`null`, true}, {`"a"`, true}, {`1`, true}, {`1.5`, false}, {`{}`, false}}
	case "08-bare-null":
		return []oneOfSemanticCase{{`null`, true}, {`"a"`, true}, {`1`, false}}
	case "09-duplicate-null":
		return []oneOfSemanticCase{{`null`, false}, {`{"x":"a"}`, true}, {`{}`, false}, {`"a"`, false}}
	case "10-strip-local-null", "11-declared-null":
		return []oneOfSemanticCase{{`null`, false}, {`"a"`, true}, {`1`, true}, {`1.5`, false}}
	case "13-any-null-rejected":
		return []oneOfSemanticCase{{`null`, false}, {`"a"`, false}, {`1`, true}, {`{}`, true}, {`[]`, true}}
	case "14-nested-local-null":
		return []oneOfSemanticCase{{`null`, false}, {`"a"`, false}, {`1`, true}, {`1.5`, false}}
	case "15-integer-width":
		return []oneOfSemanticCase{{`0`, false}, {`2147483647`, false}, {`2147483648`, false}, {`-2147483649`, false}, {`null`, false}, {`1.5`, false}}
	case "16-annotated-null":
		return []oneOfSemanticCase{{`null`, true}, {`{"x":"a"}`, true}, {`{}`, false}, {`"a"`, false}}
	case "19-nested-union":
		return []oneOfSemanticCase{{`{}`, true}, {`{"x":"a"}`, false}, {`{"z":"a"}`, false}, {`{"x":"a","z":"b"}`, true}, {`null`, false}}
	case "20-inline-object-field":
		return []oneOfSemanticCase{{`{"data":{}}`, true}, {`{"data":{"id":"a"}}`, false}, {`{"data":{},"y":1}`, false}, {`{"data":{"id":"a"},"y":1}`, true}}
	case "25-query-objects":
		return []oneOfSemanticCase{{`{"data":"a"}`, true}, {`{"error":"a"}`, true}, {`{"data":"a","error":"a"}`, false}, {`{}`, false}, {`{"data":1}`, false}}
	case "31-header-list":
		return []oneOfSemanticCase{{`[1]`, true}, {`["a"]`, false}, {`[]`, true}, {`1`, true}, {`"1"`, false}}
	case "35-form-root":
		return []oneOfSemanticCase{{`{"cat":"a"}`, true}, {`{"dog":"a"}`, true}, {`{"cat":"a","dog":"a"}`, false}, {`{}`, false}, {`{"cat":1}`, false}}
	case "44-typed-additional-members":
		return []oneOfSemanticCase{{`{"x":1,"extra":2}`, true}, {`{"x":1,"extra":"bad"}`, false}, {`{"extra":2}`, false}, {`{"x":"bad"}`, false}, {`"a"`, true}}
	default:
		return nil
	}
}

// TestOneOfSemanticPreservation checks the explicit source oracle without
// generation. Accepted rows check the same oracle against regenerated output in
// TestOneOfImportedDesignGenerates. Rejected rows have no invented output: their
// source domain and exact diagnostic establish why normalization cannot proceed.
func TestOneOfSemanticPreservation(t *testing.T) {
	for _, tc := range oneOfImportCases() {
		values := oneOfSemanticCases(tc.name)
		if len(values) == 0 {
			continue
		}
		t.Run(tc.name, func(t *testing.T) {
			source, err := os.ReadFile(filepath.Join("testdata", "oneof", tc.name+".json"))
			require.NoError(t, err)
			var document map[string]any
			require.NoError(t, json.Unmarshal(source, &document))
			requireOneOfSemanticValues(t, document, tc.name, values)
			_, diagnostics, err := Analyze(source)
			require.NoError(t, err)
			if len(tc.diagnostics) > 0 {
				require.Equal(t, tc.diagnostics, diagnostics)
			} else {
				require.Empty(t, diagnostics)
			}
		})
	}
}

func requireOneOfSemanticValues(t *testing.T, document map[string]any, name string, values []oneOfSemanticCase) {
	t.Helper()
	schema := oneOfSpecTarget(t, document, name)
	compiled, err := gojsonschema.NewSchema(gojsonschema.NewGoLoader(map[string]any{"allOf": []any{schema}, "components": document["components"]}))
	require.NoError(t, err)
	for _, tc := range values {
		result, err := compiled.Validate(gojsonschema.NewStringLoader(tc.wire))
		require.NoError(t, err)
		require.Equal(t, tc.valid, result.Valid(), "%s: membership of %s", name, tc.wire)
	}
}

// oneOfSpecTarget selects the actual union-bearing contract, rather than the
// unrelated scalar success response of parameter and request-body specimens.
func oneOfSpecTarget(t *testing.T, document map[string]any, name string) any {
	t.Helper()
	operation := document["paths"].(map[string]any)["/value"].(map[string]any)["post"].(map[string]any)
	switch name {
	case "25-query-objects", "27-query-deep-object-map", "31-header-list", "32-cookie-scalars":
		return operation["parameters"].([]any)[0].(map[string]any)["schema"]
	case "34-form-json", "35-form-root", "37-form-empty-required", "38-form-empty-optional", "39-multipart-empty", "40-form-captured-type":
		media := "application/x-www-form-urlencoded"
		if name == "39-multipart-empty" {
			media = "multipart/form-data"
		}
		schema := operation["requestBody"].(map[string]any)["content"].(map[string]any)[media].(map[string]any)["schema"]
		if name == "34-form-json" || name == "40-form-captured-type" {
			schema = resolveOneOfSpecReference(t, document, schema.(map[string]any))["properties"].(map[string]any)["value"]
		}
		return schema
	default:
		status, media := "200", "application/json"
		if name == "24-error-body" {
			status = "400"
		}
		if name == "42-raw-documentation" {
			media = "text/plain"
		}
		return operation["responses"].(map[string]any)[status].(map[string]any)["content"].(map[string]any)[media].(map[string]any)["schema"]
	}
}
