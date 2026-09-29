package valuecontract

import (
	"encoding/json/v2"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

type (
	// conformanceReport counts assertion groups, not corpus inputs. A group
	// records the Go subtest verdict. The mandatory event verifier additionally
	// rejects nested skips, which testing.T does not propagate to its parent.
	conformanceReport struct {
		Version int                    `json:"version"`
		Cases   []conformanceAssertion `json:"cases"`
	}
	conformanceAssertion struct {
		Name   string `json:"name"`
		Owner  string `json:"owner"`
		Passed bool   `json:"passed"`
	}
)

func newConformanceReport(t *testing.T) *conformanceReport {
	t.Helper()
	path := os.Getenv("LOOM_VALUE_CONFORMANCE_REPORT")
	require.NotEmpty(t, path, "enabled conformance requires a report destination")
	report := &conformanceReport{Version: 1}
	t.Cleanup(func() {
		data, err := json.Marshal(report, json.Deterministic(true))
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(path, append(data, '\n'), 0o600))
	})
	return report
}

func (r *conformanceReport) run(t *testing.T, name, owner, executable string, check func(*testing.T, string)) {
	t.Helper()
	passed := false
	t.Run(name, func(t *testing.T) {
		t.Cleanup(func() {
			passed = !t.Failed() && !t.Skipped()
		})
		check(t, executable)
	})
	r.Cases = append(r.Cases, conformanceAssertion{Name: name, Owner: owner, Passed: passed})
}

func TestConformanceReportRecordsExecutedGroups(t *testing.T) {
	report := &conformanceReport{Version: 1}
	executed := false
	report.run(t, "actual helper", "resolution", "reference", func(t *testing.T, executable string) {
		require.Equal(t, "reference", executable)
		executed = true
	})
	require.True(t, executed)
	require.Equal(t, []conformanceAssertion{{Name: "actual helper", Owner: "resolution", Passed: true}}, report.Cases)
}

// TestConformanceReportOutcomes uses a child test process because an intentional
// failed assertion must fail its process as well as its persisted report.
func TestConformanceReportOutcomes(t *testing.T) {
	for _, outcome := range []string{"pass", "fail", "skip"} {
		t.Run(outcome, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "report.json")
			cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestConformanceReportChild$")
			cmd.Env = append(os.Environ(), "LOOM_CONFORMANCE_REPORT_TEST="+outcome, "LOOM_VALUE_CONFORMANCE_REPORT="+path)
			output, err := cmd.CombinedOutput()
			if outcome == "fail" {
				require.Error(t, err, string(output))
			} else {
				require.NoError(t, err, string(output))
			}
			data, err := os.ReadFile(path)
			require.NoError(t, err)
			var report conformanceReport
			require.NoError(t, json.Unmarshal(data, &report))
			require.Equal(t, 1, report.Version)
			require.Equal(t, []conformanceAssertion{{Name: "child", Owner: "resolution", Passed: outcome == "pass"}}, report.Cases)
		})
	}
}

func TestConformanceReportChild(t *testing.T) {
	outcome := os.Getenv("LOOM_CONFORMANCE_REPORT_TEST")
	if outcome == "" {
		return
	}
	report := newConformanceReport(t)
	report.run(t, "child", "resolution", "", func(t *testing.T, _ string) {
		if outcome == "skip" {
			t.Skip("intentional report skip control")
		}
		t.Run("nested assertion", func(t *testing.T) {
			if outcome == "fail" {
				t.Error("intentional report rejection control")
			}
		})
	})
}
