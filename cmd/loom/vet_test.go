package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	loomvet "github.com/CaliLuke/loom/vet"
	"github.com/stretchr/testify/require"
)

func TestRunVet(t *testing.T) {
	original := analyzeVetDesign
	t.Cleanup(func() { analyzeVetDesign = original })

	t.Run("clean", func(t *testing.T) {
		analyzeVetDesign = func(path string, debug bool) (loomvet.Report, error) {
			require.Equal(t, "example.com/service/design", path)
			require.False(t, debug)
			return loomvet.Report{}, nil
		}
		var stdout bytes.Buffer
		var stderr bytes.Buffer

		exitCode := runVet([]string{"example.com/service/design"}, &stdout, &stderr)

		require.Zero(t, exitCode)
		require.Empty(t, stdout.String())
		require.Empty(t, stderr.String())
	})

	t.Run("JSON finding", func(t *testing.T) {
		analyzeVetDesign = func(_ string, debug bool) (loomvet.Report, error) {
			require.True(t, debug)
			return loomvet.Report{Diagnostics: []loomvet.Diagnostic{{
				Rule:     loomvet.RuleRouteOutsideDesign,
				Severity: loomvet.SeverityError,
				Message:  "manual route",
				Location: loomvet.Location{Path: "router.go", Line: 9},
			}}}, nil
		}
		var stdout bytes.Buffer
		var stderr bytes.Buffer

		exitCode := runVet([]string{"example.com/service/design", "--format", "json", "--debug"}, &stdout, &stderr)

		require.Equal(t, 1, exitCode)
		require.JSONEq(t, `{"diagnostics":[{"rule":"route-outside-design","severity":"error","message":"manual route","location":{"path":"router.go","line":9}}]}`, stdout.String())
		require.Empty(t, stderr.String())
	})
}

func TestRunVetHelp(t *testing.T) {
	original := analyzeVetDesign
	t.Cleanup(func() { analyzeVetDesign = original })
	analyzeVetDesign = func(_ string, _ bool) (loomvet.Report, error) {
		t.Error("vet analysis must not run for help")
		return loomvet.Report{}, nil
	}

	for _, args := range [][]string{
		{"-h"},
		{"--help"},
		{"example.com/service/design", "-h"},
		{"example.com/service/design", "--help"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			exitCode := runVet(args, &stdout, &stderr)

			require.Zero(t, exitCode)
			require.Empty(t, stdout.String())
			require.Contains(t, stderr.String(), "Usage:\n  loom vet PACKAGE")
		})
	}
}

func TestRunVetRequiresPackage(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := runVet(nil, &stdout, &stderr)

	require.Equal(t, 1, exitCode)
	require.Empty(t, stdout.String())
	require.Contains(t, stderr.String(), "usage: loom vet PACKAGE")
}

func TestParseVetOptionsRejectsUnknownFormat(t *testing.T) {
	var stderr bytes.Buffer
	_, err := parseVetOptions([]string{"example.com/service/design", "--format", "xml"}, &stderr)
	require.EqualError(t, err, `unknown vet format "xml": expected text, json, or sarif`)
}

func TestVetGeneratorUsesDesignModuleDirectory(t *testing.T) {
	moduleDir := t.TempDir()
	generator := &Generator{
		Command:       "vet",
		DesignPath:    "example.com/service/design",
		Output:        ".",
		DesignVersion: 3,
		moduleDir:     moduleDir,
		bin:           "loom",
	}
	require.NoError(t, generator.Write(false))
	t.Cleanup(func() { require.NoError(t, generator.Remove()) })

	require.Equal(t, moduleDir, filepath.Dir(generator.tmpDir))
	source, err := os.ReadFile(filepath.Join(generator.tmpDir, "main.go"))
	require.NoError(t, err)
	require.True(t, strings.Contains(string(source), `loomvet.Analyze(expr.Root, "`+moduleDir+`")`))
}
