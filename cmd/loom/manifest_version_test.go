package main

import (
	"encoding/json/v2"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestManifestUsesGeneratorModuleVersion(t *testing.T) {
	for _, tc := range []struct {
		version string
		valid   bool
	}{
		{"v1.10.0-alpha.6", true}, {"v1.10.0", true}, {"(devel)", true}, {"", false}, {"wrong", false},
	} {
		t.Run(tc.version, func(t *testing.T) {
			stage := t.TempDir()
			require.NoError(t, os.Mkdir(filepath.Join(stage, "gen"), 0o755))
			data, err := json.Marshal(generationManifest{LoomVersion: tc.version, DesignDigest: "test-digest"})
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(filepath.Join(stage, "gen", "loom.json"), data, 0o644))
			transaction := generationTransaction{stage: stage}
			require.Equal(t, tc.valid, transaction.validate(nil) == nil)
		})
	}
}
