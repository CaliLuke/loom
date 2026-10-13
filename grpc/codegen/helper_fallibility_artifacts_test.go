package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/grpc/codegen/testdata"
)

func TestHelperFallibilityArtifacts(t *testing.T) {
	output := os.Getenv("LOOM_HELPER_FALLIBILITY_OUTPUT")
	if output == "" {
		t.Skip("set LOOM_HELPER_FALLIBILITY_OUTPUT for isolated artifact comparison")
	}
	for _, tc := range []struct {
		name string
		dsl  func()
	}{
		{"plain-first", helperFallibilityDSL("sibling Any", false)},
		{"fallible-first", helperFallibilityDSL("sibling Any", true)},
		{"recursive", helperFallibilityDSL("recursive union Any", false)},
		{"any-errors", testdata.AnyErrorDSL},
		{"adapters", helperFallibilityAdaptersDSL},
		{"pure", helperFallibilityDSL("pure control", false)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := filepath.Join(output, tc.name)
			require.NoError(t, os.MkdirAll(dir, 0o700))
			renderGRPCModule(t, dir, "example.com/helpercontract", RunGRPCDSL(t, tc.dsl), resolveGRPCLoomSource(t))
		})
	}
}

func TestHelperFallibilityArtifactBuilds(t *testing.T) {
	output := os.Getenv("LOOM_HELPER_FALLIBILITY_OUTPUT")
	if output == "" {
		t.Skip("set LOOM_HELPER_FALLIBILITY_OUTPUT for isolated artifact compilation")
	}
	for _, name := range []string{"plain-first", "fallible-first", "recursive", "any-errors", "adapters", "pure"} {
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join(output, name)
			runGRPCGoCommand(t, dir, "mod", "tidy")
			runGRPCGoCommand(t, dir, "build", "./...")
			runGRPCGoCommand(t, dir, "vet", "./...")
		})
	}
}
