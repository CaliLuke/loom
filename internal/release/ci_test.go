package release

import (
	"context"
	"encoding/json/v2"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLatestRunDoesNotReuseOlderSuccess(t *testing.T) {
	runs := []ciRun{
		{ID: 1, SHA: "source", Branch: "main", Event: "push", Conclusion: "success"},
		{ID: 2, SHA: "source", Branch: "main", Event: "push", Conclusion: "failure"},
		{ID: 3, SHA: "other", Branch: "main", Event: "push", Conclusion: "success"},
		{ID: 4, SHA: "source", Branch: "main", Event: "pull_request", Conclusion: "success"},
	}
	run, err := latestRun(runs, "source")
	require.NoError(t, err)
	require.Equal(t, int64(2), run.ID)
	_, err = latestRun(runs, "missing")
	require.Error(t, err)
}

func TestRequiredJobs(t *testing.T) {
	for _, tc := range []struct {
		name string
		jobs []ciJob
		good bool
	}{
		{"complete", []ciJob{{"ci", "completed", "success"}, {"Release eligibility", "completed", "success"}}, true},
		{"skipped matrix entry", []ciJob{{"testdata-compile (expr)", "completed", "skipped"}, {"Release eligibility", "completed", "success"}}, false},
		{"cancelled", []ciJob{{"ci", "completed", "cancelled"}}, false},
		{"no gate", []ciJob{{"ci", "completed", "success"}}, false},
		{"empty", nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) { require.Equal(t, tc.good, validateJobs("test.yml", tc.jobs) == nil) })
	}
}

func TestCheckCI(t *testing.T) {
	for _, tc := range []struct {
		name, status, conclusion string
		pending, bad             bool
	}{
		{"green", "completed", "success", false, false},
		{"pending", "in_progress", "", true, false},
		{"red", "completed", "failure", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := publisher{run: func(_ context.Context, name string, args ...string) ([]byte, error) {
				require.Equal(t, "gh", name)
				if strings.Contains(args[1], "/workflows/") {
					return json.Marshal(map[string]any{"workflow_runs": []ciRun{{ID: 42, Attempt: 2, SHA: "source", Branch: "main", Event: "push", Status: tc.status, Conclusion: tc.conclusion}}})
				}
				require.Contains(t, args[1], "/attempts/2/jobs")
				return []byte(`[{"jobs":[{"name":"Release eligibility","status":"completed","conclusion":"success"}]}]`), nil
			}}
			evidence, pending, err := p.checkCI(context.Background(), "source")
			require.Equal(t, tc.bad, err != nil)
			require.Equal(t, tc.pending, pending)
			if !tc.bad && !tc.pending {
				require.Len(t, evidence.Runs, 2)
			}
		})
	}
}

func TestReleaseLookupDoesNotTreatNetworkFailureAsAbsence(t *testing.T) {
	p := publisher{run: func(context.Context, string, ...string) ([]byte, error) {
		return nil, errors.New("network unavailable")
	}}
	_, err := p.releaseIfExists(context.Background(), "v1.10.0-alpha.6")
	require.ErrorContains(t, err, "network unavailable")
}
