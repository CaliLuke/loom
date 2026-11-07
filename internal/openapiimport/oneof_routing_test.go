package openapiimport

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

// Owner labels identify the excluded capability. Assertions retain each owner's
// existing code, exact source location and explanatory wording; adding oneOf
// must never replace those with a blanket union-location rejection.
func TestOneOfDiagnosticOwnerRouting(t *testing.T) {
	const component = "#/components/schemas/Value"
	for _, tc := range []struct {
		owner, schema, code, path, wording string
	}{
		{"#1000 type arrays", `{"oneOf":[{"type":["string","integer"]},{"type":"boolean"}]}`, "schema-type-union", component + "/oneOf/0/type", "one concrete type and null"},
		{"#1006 composite defaults", `{"oneOf":[{"type":"object","properties":{"x":{"type":"string"}}},{"type":"string"}],"default":{"x":"a"}}`, "schema-keyword", component, "default value is not compatible"},
		{"#1010 null-only", `{"oneOf":[{"type":"null"}]}`, "schema-type", component, "null-only type (#1010)"},
		{"#1018 assertion siblings", `{"oneOf":[{"type":"string"},{"type":"integer"}],"minLength":1}`, "schema-composition", component + "/minLength", "oneOf sibling minLength"},
		{"#1018 intersections", `{"oneOf":[{"allOf":[{"type":"string"},{"minLength":1}]},{"type":"integer"}]}`, "schema-composition", component + "/oneOf/0", "composition cannot be normalized"},
		{"#1019 anyOf", `{"oneOf":[{"anyOf":[{"type":"string"},{"type":"integer"}]},{"type":"boolean"}]}`, "schema-composition", component + "/oneOf/0", "composition cannot be normalized"},
		{"#1020 negation", `{"oneOf":[{"type":"string","not":{"enum":["a"]}},{"type":"integer"}]}`, "schema-composition", component + "/oneOf/0", "composition cannot be normalized"},
		{"#993 unevaluated schema", `{"oneOf":[{"type":"object","properties":{"x":{"type":"string"}}},{"type":"string"}],"unevaluatedProperties":{"type":"integer"}}`, "advanced-schema", component, "advanced JSON Schema applicators"},
	} {
		t.Run(tc.owner, func(t *testing.T) {
			_, diagnostics, err := Analyze(schemaAdmissionSource("3.2.0", tc.schema, "component"))
			require.NoError(t, err)
			requireOneOfOwnerDiagnostic(t, diagnostics, tc.code, tc.path, tc.wording)
		})
	}
	const union = `{"oneOf":[{"type":"string"},{"type":"integer"}]}`
	for _, tc := range []struct {
		owner, operation, code, path, wording string
	}{
		{"#970 query content codec", `"parameters":[{"name":"value","in":"query","content":{"text/plain":{"schema":` + union + `}}}],"responses":{"204":{"description":"ok"}}`, "parameter-serialization", "#/operations/0/parameters/0", `field media type "text/plain" has no equivalent query codec`},
		{"#970 header content location", `"parameters":[{"name":"value","in":"header","content":{"application/json":{"schema":` + union + `}}}],"responses":{"204":{"description":"ok"}}`, "parameter-content", "#/paths/~1value/post/parameters/0/content", "only for named JSON query objects"},
		{"#972 item schema", `"responses":{"200":{"description":"ok","content":{"application/json-seq":{"schema":{"type":"string","format":"binary"},"itemSchema":` + union + `}}}}`, "media-item-schema", "#/paths/~1value/post/responses/200/content/application~1json-seq/itemSchema", "item schemas are not in the strict import subset"},
		{"#973 positional encoding", `"requestBody":{"content":{"multipart/form-data":{"schema":{"type":"object","properties":{"value":` + union + `}},"encoding":{"value":{"contentType":"application/json"}},"prefixEncoding":[]}}},"responses":{"204":{"description":"ok"}}`, "media-prefix-encoding", "#/paths/~1value/post/requestBody/content/multipart~1form-data/prefixEncoding", "prefixEncoding is not in the strict import subset"},
	} {
		t.Run(tc.owner, func(t *testing.T) {
			source := fmt.Appendf(nil, `{"openapi":"3.2.0","info":{"title":"Routed union","version":"1"},"paths":{"/value":{"post":{%s}}}}`, tc.operation)
			_, diagnostics, err := Analyze(source)
			require.NoError(t, err)
			requireOneOfOwnerDiagnostic(t, diagnostics, tc.code, tc.path, tc.wording)
		})
	}
}

func requireOneOfOwnerDiagnostic(t *testing.T, diagnostics Diagnostics, code, path, wording string) {
	t.Helper()
	found := false
	for _, diagnostic := range diagnostics {
		require.NotEqual(t, "schema-oneof-location", diagnostic.Code)
		if diagnostic.Code == code && diagnostic.Path == path {
			require.Contains(t, diagnostic.Message, wording)
			found = true
		}
	}
	require.True(t, found, "want %s at %s with %q, got %v", code, path, wording, diagnostics)
}
