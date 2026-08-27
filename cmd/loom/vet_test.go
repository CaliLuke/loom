package main

import (
	"bytes"
	"os"
	"os/exec"
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

func TestVetDesignBuildsFromTidyConsumerWithoutChangingModuleFiles(t *testing.T) {
	moduleDir := writeTidyVetConsumer(t, `package design

import . "github.com/CaliLuke/loom/dsl"

var _ = API("service", func() {})
	`)

	goModBefore, err := os.ReadFile(filepath.Join(moduleDir, "go.mod"))
	require.NoError(t, err)
	goSumBefore, err := os.ReadFile(filepath.Join(moduleDir, "go.sum"))
	require.NoError(t, err)

	t.Chdir(moduleDir)
	_, err = vetDesign("example.com/service/design", false)
	require.NoError(t, err)

	goModAfter, err := os.ReadFile(filepath.Join(moduleDir, "go.mod"))
	require.NoError(t, err)
	goSumAfter, err := os.ReadFile(filepath.Join(moduleDir, "go.sum"))
	require.NoError(t, err)
	require.Equal(t, goModBefore, goModAfter)
	require.Equal(t, goSumBefore, goSumAfter)
	temporaryHelpers, err := filepath.Glob(filepath.Join(moduleDir, "loom*"))
	require.NoError(t, err)
	require.Empty(t, temporaryHelpers)
}

func TestVetDesignRemovesFailedHelperInDebugMode(t *testing.T) {
	moduleDir := writeTidyVetConsumer(t, `package design

import . "github.com/CaliLuke/loom/dsl"

var _ = API("service", func() {})
var _ = missingIdentifier
	`)
	t.Chdir(moduleDir)

	_, err := vetDesign("example.com/service/design", true)
	require.Error(t, err)
	temporaryHelpers, globErr := filepath.Glob(filepath.Join(moduleDir, "loom*"))
	require.NoError(t, globErr)
	require.Empty(t, temporaryHelpers)
}

func writeTidyVetConsumer(t *testing.T, designSource string) string {
	t.Helper()
	repositoryRoot, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	moduleDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(moduleDir, "design"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(moduleDir, "go.mod"), []byte(
		"module example.com/service\n\n"+
			"go 1.27\n\n"+
			"require github.com/CaliLuke/loom v0.0.0\n\n"+
			"replace github.com/CaliLuke/loom => "+repositoryRoot+"\n",
	), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(moduleDir, "design", "design.go"), []byte(designSource), 0o644))

	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = moduleDir
	tidy.Env = append(os.Environ(), "GOWORK=off")
	output, err := tidy.CombinedOutput()
	require.NoError(t, err, string(output))
	return moduleDir
}
