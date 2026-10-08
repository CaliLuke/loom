package openapiimport

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestComponentReferencePointerSyntax(t *testing.T) {
	const prefix = "#/components/schemas/"
	for _, tc := range []struct {
		name    string
		segment string
		decoded string
		invalid bool
	}{
		{name: "plain", segment: "Pet", decoded: "Pet"},
		{name: "escaped", segment: "Pet~1~0ID", decoded: "Pet/~ID"},
		{name: "escape once", segment: "Pet~01ID", decoded: "Pet~1ID"},
		{name: "unicode", segment: "café", decoded: "café"},
		{name: "empty component", invalid: true},
		{name: "nested path", segment: "Pet/name", invalid: true},
		{name: "trailing escape", segment: "Pet~", invalid: true},
		{name: "unknown escape", segment: "Pet~2", invalid: true},
		{name: "invalid UTF-8", segment: "Pet\xff", invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			name, err := localComponentReferenceName(prefix+tc.segment, prefix)
			if tc.invalid {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.decoded, name)
			require.Equal(t, tc.segment, escapeJSONPointer(name))
		})
	}
	_, err := localComponentReferenceName("#/components/parameters/Pet", prefix)
	require.ErrorContains(t, err, "wrong kind")
	require.Empty(t, escapeJSONPointer(""))
}

func TestComponentDiagnosticPointerOwnership(t *testing.T) {
	const prefix = "#/components/schemas/"
	retained := map[string]struct{}{"Pet/~ID": {}, "": {}, "café": {}}
	for _, tc := range []struct {
		path string
		keep bool
	}{
		{prefix + "Pet~1~0ID/properties/name/type", true},
		{prefix + "café/type", true},
		{prefix + "/type", true},
		{prefix + "Other/type", false},
		{"#/paths/~1pets", true},
		{prefix + "Other~2/type", true},
		{prefix + "Other\xff/type", true},
	} {
		t.Run(tc.path, func(t *testing.T) {
			require.Equal(t, tc.keep, componentDiagnosticRetained(tc.path, prefix, retained))
		})
	}
}
