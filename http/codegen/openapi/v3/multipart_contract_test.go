package openapiv3_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestCapturedMultipartContracts(t *testing.T) {
	var vacuum contractVacuum
	if os.Getenv("LOOM_OPENAPI_CONTRACT") != "" {
		vacuum = installContractVacuum(t)
	}
	for _, version := range []struct{ target, version string }{{"3.1", "3.1.1"}, {"3.2", "3.2.0"}} {
		t.Run(version.target, func(t *testing.T) {
			artifacts := renderOpenAPIArtifactsForVersion(t, testdata.UntaggedMultipartDSL, version.target, version.version)
			validateOpenAPIVersion(t, artifacts.JSON, version.version)
			validateOpenAPIVersion(t, artifacts.YAML, version.version)
			spec := decodeOpenAPIJSON(t, artifacts.JSON)
			for _, path := range []string{"/root", "/nested", "/optional", "/Composite", "/WholeItems"} {
				body := requireMap(t, requireOperation(t, spec, path, "post")["requestBody"], "request body")
				require.Equal(t, path != "/optional", body["required"] == true)
				media := requireMap(t, requireMap(t, body["content"], "content")["multipart/form-data"], "multipart")
				require.NotEmpty(t, media["schema"])
			}
			body := requireMap(t, requireOperation(t, spec, "/fields", "post")["requestBody"], "fields body")
			media := requireMap(t, requireMap(t, body["content"], "content")["multipart/form-data"], "multipart")
			encoding := requireMap(t, media["encoding"], "encoding")
			require.Equal(t, "application/json", requireMap(t, encoding["json"], "json")["contentType"])
			styled := requireMap(t, encoding["styled"], "styled")
			require.Equal(t, "form", styled["style"])
			require.Equal(t, false, styled["explode"])
			require.NotContains(t, styled, "contentType")
			if vacuum.binary != "" {
				for extension, data := range map[string][]byte{"json": artifacts.JSON, "yaml": artifacts.YAML} {
					path := filepath.Join(t.TempDir(), "multipart."+extension)
					require.NoError(t, os.WriteFile(path, data, 0o600))
					require.NoError(t, vacuum.lint(path))
				}
			}
		})
	}
}
