package testdatacompile

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExpectedFailureStages(t *testing.T) {
	for _, c := range []struct {
		name, stage, phase, output string
		valid                      bool
	}{
		{"DSL", "dsl", "gen", "stage eval.RunDSL: invalid field", true},
		{"DSL context", "dsl", "gen", "stage eval.Context.Errors: invalid field", true},
		{"analysis", "openapi-analysis", "gen", "stage generator.Generate: stage generate-initial-files path generator[2]: OpenAPI invalid field", true},
		{"wrong stage", "dsl", "gen", "stage generator.Generate: invalid field", false},
		{"wrong generator", "openapi-analysis", "gen", "stage generator.Generate: stage generate-initial-files path generator[1]: OpenAPI invalid field", false},
		{"panic", "openapi-analysis", "gen", "panic: stage generator.Generate: stage generate-initial-files path generator[2]: OpenAPI invalid field", false},
		{"missing diagnostic", "dsl", "gen", "stage eval.RunDSL: another error", false},
		{"wrong phase", "dsl", "build", "stage eval.RunDSL: invalid field", false},
		{"unexpected pass", "dsl", "", "", false},
		{"infrastructure", "dsl", "gen", "stage eval.RunDSL: invalid field: no space left on device", false},
		{"unknown stage", "unknown", "gen", "stage eval.RunDSL: invalid field", false},
	} {
		t.Run(c.name, func(t *testing.T) {
			err := checkExpectedFailure(expectation{Phase: "gen", Contains: "invalid field", Stage: c.stage}, observation{Phase: c.phase, Output: c.output})
			if c.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestTrackedFailureRejectsPanicsAndInfrastructure(t *testing.T) {
	want := expectation{Phase: "gen", Contains: "invalid field", Issue: "https://github.com/CaliLuke/loom/issues/624"}
	for _, prefix := range []string{"panic: ", "runtime error: ", "no space left on device: ", "context deadline exceeded: "} {
		t.Run(prefix, func(t *testing.T) {
			require.Error(t, checkExpectedFailure(want, observation{Phase: "gen", Output: prefix + "invalid field"}))
		})
	}
	require.NoError(t, checkExpectedFailure(want, observation{Phase: "gen", Output: "invalid field"}))
}
