package openapiimport

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOneOfFormFieldCodecs(t *testing.T) {
	for _, tc := range []struct {
		name, media, schema, encoding, retained string
	}{
		{"explicit text", "application/x-www-form-urlencoded", `{"oneOf":[{"type":"string","const":"a"},{"type":"string","const":"b"}]}`, `,"encoding":{"value":{"contentType":"text/plain"}}`, "text/plain"},
		{"source scalar default", "application/x-www-form-urlencoded", `{"type":"string","oneOf":[{"type":"string","const":"a"},{"type":"string","const":"b"}]}`, "", `Meta("http:encoding:sourceType", "string")`},
		{"source array default", "application/x-www-form-urlencoded", `{"type":"array","oneOf":[{"type":"array","items":{"type":"string"}},{"type":"array","items":{"type":"integer"}}]}`, "", `Meta("http:encoding:sourceType", "array")`},
		{"explicit form style", "application/x-www-form-urlencoded", `{"oneOf":[{"type":"string"},{"type":"integer"}]}`, `,"encoding":{"value":{"style":"form","explode":false}}`, "form"},
		{"multipart JSON", "multipart/form-data", `{"oneOf":[{"type":["string","null"]},{"type":"integer"}]}`, `,"encoding":{"value":{"contentType":"application/json"}}`, "application/json"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := fmt.Appendf(nil, `{"openapi":"3.2.0","info":{"title":"Codecs","version":"1"},"paths":{"/value":{"post":{"operationId":"value","requestBody":{"required":true,"content":{%q:{"schema":{"type":"object","properties":{"value":%s}}%s}}},"responses":{"204":{"description":"done"}}}}}}`, tc.media, tc.schema, tc.encoding)
			document, diagnostics, err := Analyze(source)
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			design, err := Render(document, Options{PackageName: "design"})
			require.NoError(t, err)
			require.Contains(t, string(design), tc.retained)
			requireRenderedDesignGenerates(t, design)
		})
	}
}
