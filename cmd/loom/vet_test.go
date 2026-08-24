package main

import (
	"bytes"
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

func TestParseVetOptionsRejectsUnknownFormat(t *testing.T) {
	var stderr bytes.Buffer
	_, err := parseVetOptions([]string{"example.com/service/design", "--format", "xml"}, &stderr)
	require.EqualError(t, err, `unknown vet format "xml": expected text, json, or sarif`)
}
