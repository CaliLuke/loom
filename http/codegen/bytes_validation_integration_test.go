package codegen

import (
	_ "embed"
	"os"
	"path/filepath"
	"testing"

	codegentestdata "github.com/CaliLuke/loom/codegen/testdata"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/bytes_validation_http_test.go.txt
var bytesValidationHTTPHarness string

// TestBytesValidationGeneratedHTTP compiles both retained and normalized byte
// aliases, including response/view validators and query/header locals.
func TestBytesValidationGeneratedHTTP(t *testing.T) {
	for _, tc := range []struct {
		name    string
		dsl     func()
		runtime bool
	}{
		{"canonical", codegentestdata.BytesValidationHTTPDSL, true},
		{"unpreserved", codegentestdata.BytesValidationHTTPUnpreservedDSL, true},
		{"minimal", codegentestdata.BytesValidationHTTPMinimalDSL, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := RunHTTPDSL(t, tc.dsl)
			dir := t.TempDir()
			renderHTTPModule(t, dir, "example.com/bytesvalidation", root)
			if tc.runtime {
				require.NoError(t, os.WriteFile(filepath.Join(dir, "bytes_test.go"), []byte(bytesValidationHTTPHarness), 0o600))
			}
			runGoCommand(t, dir, "mod", "tidy")
			runGoCommand(t, dir, "build", "./...")
			runGoCommand(t, dir, "vet", "./...")
			runGoCommand(t, dir, "test", "-race", "-count=1", "./...")
		})
	}
}
