package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/internal/testingx"

	grpcdata "github.com/CaliLuke/loom/grpc/codegen/testdata"
	httpdata "github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestDefaultApplicationArtifacts(t *testing.T) {
	output := os.Getenv("LOOM_DEFAULT_APPLICATION_OUTPUT")
	if output == "" {
		t.Skip("set LOOM_DEFAULT_APPLICATION_OUTPUT for isolated generation comparison")
	}
	for _, tc := range []struct {
		name string
		dsl  func()
	}{
		{"transports", defaultApplicationDSL},
		{"bytes", grpcdata.DefaultBytesDSL},
		{"cli", defaultApplicationCLIDSL},
		{"views", httpdata.ResultViewPresenceDSL},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := filepath.Join(output, tc.name)
			require.NoError(t, os.MkdirAll(dir, 0o750))
			renderDesignModule(t, dir, "example.com/defaultapplication", tc.dsl)
		})
	}
}

func TestDefaultApplicationArtifactBuilds(t *testing.T) {
	output := os.Getenv("LOOM_DEFAULT_APPLICATION_OUTPUT")
	if output == "" {
		t.Skip("set LOOM_DEFAULT_APPLICATION_OUTPUT to build existing comparison modules")
	}
	for _, name := range []string{"transports", "bytes", "cli", "views"} {
		t.Run(name, func(t *testing.T) {
			for _, args := range [][]string{{"mod", "tidy"}, {"build", "./..."}, {"vet", "./..."}} {
				output, err := testingx.RunCmd(filepath.Join(output, name), "go", args...)
				require.NoError(t, err, output)
			}
		})
	}
}
