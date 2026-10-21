package main

import (
	"encoding/json/v2"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/internal/designfingerprint"
)

// TestGeneratorCanonicalDesignOrigin resolves relative and import-path spellings
// through the package loader before embedding the origin in generation or vet.
func TestGeneratorCanonicalDesignOrigin(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/origin\n\ngo 1.27.3\n"), 0600))
	require.NoError(t, os.Mkdir(filepath.Join(dir, "design"), 0750))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "design", "design.go"), []byte("package design\n"), 0600))
	t.Chdir(dir)
	for _, command := range []string{"gen", "vet"} {
		for _, origin := range []string{"example.com/origin/design", "./design"} {
			generator := NewGenerator(command, origin, ".", false)
			require.Equal(t, "example.com/origin/design", generator.DesignPath)
		}
	}
}

// TestStagedManifestRequiresDesignProvenance rejects incomplete manifests before
// installation while accepting the same complete record emitted by generation.
func TestStagedManifestRequiresDesignProvenance(t *testing.T) {
	for _, tc := range []struct {
		name    string
		origin  string
		digest  string
		message string
	}{
		{"complete", "example.com/design", "digest", ""},
		{"absent origin", "", "digest", "invalid design package"},
		{"relative origin", "./design", "digest", "invalid design package"},
		{"missing digest", "example.com/design", "", "no design digest"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stage := t.TempDir()
			require.NoError(t, os.Mkdir(filepath.Join(stage, "gen"), 0750))
			data, err := json.Marshal(designfingerprint.Manifest{LoomVersion: "(devel)", DesignPackage: tc.origin, DesignDigest: tc.digest})
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(filepath.Join(stage, "gen", "loom.json"), data, 0600))
			transaction := generationTransaction{stage: stage}
			err = transaction.validate(nil)
			if tc.message == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.message)
			}
		})
	}
}
