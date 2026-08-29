package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseProfileDeduplicatesConsumerCoverage(t *testing.T) {
	profile := `mode: count
github.com/CaliLuke/loom/dsl/api.go:10.1,12.2 3 0
github.com/CaliLuke/loom/dsl/api.go:10.1,12.2 3 4
github.com/CaliLuke/loom/eval/run.go:20.1,21.2 2 0
github.com/CaliLuke/loom/expr/root.go:30.1,31.2 5 1
`

	blocks, err := parseProfile(strings.NewReader(profile))
	require.NoError(t, err)
	require.Len(t, blocks, 3)

	measurement, err := measureBoundary(blocks, boundary{
		Name: "design-semantics",
		Packages: []string{
			"github.com/CaliLuke/loom/dsl",
			"github.com/CaliLuke/loom/eval",
			"github.com/CaliLuke/loom/expr",
		},
	})
	require.NoError(t, err)
	require.Equal(t, 10, measurement.TotalStatements)
	require.Equal(t, 8, measurement.CoveredStatements)
	require.Equal(t, 8000, measurement.BasisPoints)
}

func TestMeasureBoundaryRejectsMissingPackage(t *testing.T) {
	blocks, err := parseProfile(strings.NewReader(`mode: count
github.com/CaliLuke/loom/dsl/api.go:10.1,12.2 3 1
`))
	require.NoError(t, err)

	_, err = measureBoundary(blocks, boundary{
		Name:     "missing",
		Packages: []string{"github.com/CaliLuke/loom/expr"},
	})
	require.ErrorContains(t, err, "has no statements")
}

func TestEvaluateMeasurementsReportsRegressions(t *testing.T) {
	config := baselineConfig{Boundaries: []boundary{
		{Name: "design-semantics", MinimumBasisPoints: 8100},
		{Name: "service-codegen", MinimumBasisPoints: 7500},
	}}
	measurements := map[string]measurement{
		"design-semantics": {BasisPoints: 8000},
		"service-codegen":  {BasisPoints: 7600},
	}

	regressions := evaluateMeasurements(config, measurements)
	require.Equal(t, []string{
		"design-semantics coverage is 80.00%, below checked-in baseline 81.00%",
	}, regressions)
}

func TestValidateConfigRejectsDuplicateBoundary(t *testing.T) {
	config := baselineConfig{
		CoverPackages: []string{"github.com/CaliLuke/loom/dsl"},
		TestPackages:  []string{"./dsl"},
		Boundaries: []boundary{
			{Name: "design", Packages: []string{"github.com/CaliLuke/loom/dsl"}, Rationale: "first"},
			{Name: "design", Packages: []string{"github.com/CaliLuke/loom/dsl"}, Rationale: "duplicate"},
		},
	}

	err := validateConfig(config)
	require.ErrorContains(t, err, `duplicate boundary "design"`)
}
