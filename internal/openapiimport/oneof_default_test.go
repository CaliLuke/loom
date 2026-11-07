package openapiimport

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Whole-union defaults remain outside the constructor contract. A scalar
// default on an individual branch is preserved, and singleton normalization
// leaves an ordinary scalar that can also retain its default.
func TestOneOfDefaultBoundary(t *testing.T) {
	for _, tc := range []struct {
		name, schema string
		accepted     bool
	}{
		{"whole union string", `{"oneOf":[{"type":"string"},{"type":"integer"}],"default":"a"}`, false},
		{"whole union integer", `{"oneOf":[{"type":"string"},{"type":"integer"}],"default":1}`, false},
		{"whole union ambiguous integer", `{"oneOf":[{"type":"integer"},{"type":"number"}],"default":1}`, false},
		{"branch scalar", `{"oneOf":[{"type":"string","default":"a"},{"type":"integer"}]}`, true},
		{"collapsed singleton", `{"oneOf":[{"type":"string"}],"default":"a"}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			document, diagnostics, err := Analyze(schemaAdmissionSource("3.2.0", tc.schema, "component"))
			require.NoError(t, err)
			if !tc.accepted {
				requireOneOfOwnerDiagnostic(t, diagnostics, "schema-keyword", "#/components/schemas/Value", "default value is not compatible")
				return
			}
			require.Empty(t, diagnostics)
			design, err := Render(document, Options{PackageName: "design"})
			require.NoError(t, err)
			require.Contains(t, string(design), `Default("a")`)
			if tc.name == "branch scalar" {
				require.Contains(t, string(design), "OneOf(func()")
				require.Contains(t, string(design), `Attribute("String", String, func()`)
			} else {
				require.NotContains(t, string(design), "OneOf(")
			}
		})
	}
}
