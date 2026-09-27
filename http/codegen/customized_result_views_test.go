package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

const customizedResultViewHTTPHarness = `package httpcustomviews_test

import (
	"encoding/json/v2"
	"testing"

	client "example.com/httpcustomviews/gen/http/test_service/client"
	views "example.com/httpcustomviews/gen/test_service/views"
)

func TestOverriddenViewConversion(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		wantString bool
		wantError bool
	}{
		{"optional absent", "{\"int\":7}", false, false},
		{"optional present", "{\"int\":7,\"string\":\"value\"}", true, false},
		{"required absent", "{}", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var body client.TestEndpointOverriddenResponseBodyOverridden
			if err := json.Unmarshal([]byte(tc.body), &body); err != nil {
				t.Fatal(err)
			}
			result := client.NewTestEndpointOverriddenResultTestEndpointOverriddenResultOK(&body)
			if (result.String != nil) != tc.wantString {
				t.Errorf("string = %v, want presence %v", result.String, tc.wantString)
			}
			if result.String != nil && *result.String != "value" {
				t.Errorf("string = %q", *result.String)
			}
			if !tc.wantError && (result.Int == nil || *result.Int != 7) {
				t.Errorf("int = %v, want 7", result.Int)
			}
			if err := views.ValidateResultTestEndpointOverriddenResultViewOverridden(result); (err != nil) != tc.wantError {
				t.Errorf("validation error = %v, wantError %v", err, tc.wantError)
			}
		})
	}
}
`

// TestCustomizedResultViewClientConversion keeps the transport body's selected
// view requiredness aligned with the generated client conversion.
func TestCustomizedResultViewClientConversion(t *testing.T) {
	code := renderClientTypesCode(t, testdata.ExplicitViewDSL)
	functions := strings.SplitN(code, "func NewTestEndpointOverridden", 2)
	require.Len(t, functions, 2)
	conversion := strings.SplitN(functions[1], "\nfunc ", 2)[0]
	require.Contains(t, conversion, "if actual, ok := body.String.Value(); ok {")
	require.NotContains(t, conversion, "body.Int.Value()")
}

// TestCustomizedResultViewsGeneratedModule compiles the HTTP service, server,
// and client with method-specific requiredness and explicit view overrides.
func TestCustomizedResultViewsGeneratedModule(t *testing.T) {
	root := RunHTTPDSL(t, testdata.ExplicitViewDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/httpcustomviews", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "view_test.go"), []byte(customizedResultViewHTTPHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "-race", "-count=1", "./...")
}
