package openapiv3_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestPreparedEnumOpenAPI(t *testing.T) {
	var vacuum contractVacuum
	if os.Getenv("LOOM_OPENAPI_CONTRACT") != "" {
		vacuum = installContractVacuum(t)
	}
	for _, tc := range []struct {
		name string
		dsl  func()
	}{
		{"implicit", testdata.PreparedEnumProvenanceDSL},
		{"mapped", testdata.PreparedEnumMappedDSL},
		{"explicit", testdata.PreparedEnumExplicitDSL},
	} {
		t.Run(tc.name, func(t *testing.T) {
			artifacts := renderOpenAPIArtifacts(t, tc.dsl)
			if vacuum.binary != "" {
				for ext, data := range map[string][]byte{"json": artifacts.JSON, "yaml": artifacts.YAML} {
					path := filepath.Join(t.TempDir(), "openapi."+ext)
					require.NoError(t, os.WriteFile(path, data, 0o600))
					require.NoError(t, vacuum.lint(path))
				}
			}
		})
	}
	if vacuum.binary != "" {
		checkVacuumEnumValues(t, vacuum)
	}
}
