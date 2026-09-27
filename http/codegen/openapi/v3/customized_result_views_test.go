package openapiv3

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

// TestCustomizedResultViewSchemas verifies that projected HTTP contracts use
// method-specific requirements and preserve authored view overrides.
func TestCustomizedResultViewSchemas(t *testing.T) {
	root := codegen.RunDSL(t, testdata.ExplicitViewDSL)
	bodies, types := buildBodyTypes(root.API, root.Types, root.ResultTypes)
	for _, tc := range []struct {
		method   string
		required []string
	}{
		{"testEndpointDefault", nil},
		{"testEndpointTiny", nil},
		{"testEndpointCustomized", []string{"string"}},
		{"testEndpointOverridden", []string{"int"}},
	} {
		t.Run(tc.method, func(t *testing.T) {
			schema := derefSchema(t, bodies["testService"][tc.method].ResponseBodies[200][0], types)
			require.ElementsMatch(t, tc.required, schema.Required)
		})
	}
}
