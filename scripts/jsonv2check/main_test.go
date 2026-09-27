package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScanRepositoryWithRemovedTrackedFile(t *testing.T) {
	for _, directory := range []bool{false, true} {
		name := "deleted"
		if directory {
			name = "replaced with directory"
		}
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			missing := filepath.Join(root, "removed.go")
			require.NoError(t, os.WriteFile(missing, []byte("package specimen\n"), 0o600))
			for _, args := range [][]string{{"init", "--quiet"}, {"add", "removed.go"}} {
				cmd := exec.Command("git", args...)
				cmd.Dir = root
				output, err := cmd.CombinedOutput()
				require.NoError(t, err, "%s", output)
			}
			require.NoError(t, os.Remove(missing))
			if directory {
				require.NoError(t, os.Mkdir(missing, 0o700))
			}
			// An untracked file must still be scanned after a tracked file is deleted.
			source := []byte(`package specimen
import "encoding/json/v2"
type Value struct{}
func (Value) MarshalJSON() ([]byte, error) {
    return json.Marshal(map[string]any{"key": true})
}
`)
			require.NoError(t, os.WriteFile(filepath.Join(root, "z_present.go"), source, 0o600))

			findings, err := scanRepository(root)
			if directory {
				require.ErrorContains(t, err, "read removed.go")
			} else {
				require.NoError(t, err)
				require.Equal(t, []finding{{Path: "z_present.go", Line: 5, Column: 12}}, findings)
			}
		})
	}
}

func TestInspectSourceRejectsMarshalWithoutDeterministicOption(t *testing.T) {
	source := []byte(`package specimen

import json "encoding/json/v2"

type Value struct{}

func (Value) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{"zulu": true, "alpha": true})
}
`)

	findings, err := inspectSource("specimen.go", source)
	require.NoError(t, err)
	require.Equal(t, []finding{{Path: "specimen.go", Line: 8, Column: 9}}, findings)
}

func TestInspectSourceRejectsUnrelatedMarshalOption(t *testing.T) {
	source := []byte(`package specimen

import json "encoding/json/v2"

type Value struct{}

func (Value) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{"zulu": true, "alpha": true}, json.FormatNilMapAsNull(true))
}
`)

	findings, err := inspectSource("specimen.go", source)
	require.NoError(t, err)
	require.Equal(t, []finding{{Path: "specimen.go", Line: 8, Column: 9}}, findings)
}

func TestInspectSourceAcceptsExplicitOptionsAndUnrelatedHelpers(t *testing.T) {
	source := []byte(`package specimen

import json "encoding/json/v2"

type Value struct{}

func (Value) MarshalJSON() ([]byte, error) {
	if false {
		return helper.Marshal(map[string]any{"zulu": true, "alpha": true})
	}
	return json.Marshal(
		map[string]any{"zulu": true, "alpha": true},
		json.Deterministic(true),
	)
}
`)

	findings, err := inspectSource("specimen.go", source)
	require.NoError(t, err)
	require.Empty(t, findings)
}

func TestInspectSourceParsesGeneratedDeclarationFragments(t *testing.T) {
	source := []byte(`import json "encoding/json/v2"

type Value struct{}

func (Value) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{"zulu": true, "alpha": true})
}
`)

	findings, err := inspectSource("specimen.golden", source)
	require.NoError(t, err)
	require.Equal(t, []finding{{Path: "specimen.golden", Line: 6, Column: 9}}, findings)
}

func TestInspectSourceChecksGeneratedFragmentsWithoutImports(t *testing.T) {
	source := []byte(`type Value struct{}

func (Value) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{"zulu": true, "alpha": true})
}
`)

	findings, err := inspectSource("specimen.golden", source)
	require.NoError(t, err)
	require.Equal(t, []finding{{Path: "specimen.golden", Line: 4, Column: 9}}, findings)
}
