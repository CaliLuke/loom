package selvedge

import (
	"context"
	"errors"
	"testing"

	sv "github.com/CaliLuke/selvedge"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/execution"
)

func TestWorkflowLookup(t *testing.T) {
	p, store := newAdmissionRole()
	wf := execution.Workflow[string, string]{Name: "echo"}
	require.NoError(t, Declare(p, wf))
	contract := p.core.contracts[wf.Name]
	reads := 0
	var failure error
	contract.read = func(context.Context, string) (any, error) {
		reads++
		if failure != nil {
			return nil, failure
		}
		d := store.stored["id"]
		return &sv.WorkflowResult[string]{InstanceID: "id", Status: string(d.Status), Output: "decoded", Error: errors.New("failure")}, nil
	}
	p.core.contracts[wf.Name] = contract
	require.NoError(t, p.Start(context.Background()))
	_, err := Lookup(context.Background(), p, wf, "missing")
	require.ErrorIs(t, err, sv.ErrInstanceNotFound)
	require.Zero(t, reads)
	store.stored["id"] = &sv.WorkflowDescription{InstanceID: "id", WorkflowName: "other", Status: sv.StatusCompleted}
	_, err = Lookup(context.Background(), p, wf, "id")
	require.Error(t, err)
	require.Zero(t, reads)
	store.stored["id"].WorkflowName = wf.Name
	for _, state := range []sv.WorkflowStatus{sv.StatusRunning, sv.StatusCompleted, sv.StatusFailed, sv.StatusCancelled} {
		store.stored["id"].Status = state
		job, err := Lookup(context.Background(), p, wf, "id")
		require.NoError(t, err)
		require.Equal(t, state, job.State)
		if state == sv.StatusCompleted {
			require.Equal(t, "decoded", job.Output)
		} else {
			require.Empty(t, job.Output)
		}
		if state == sv.StatusFailed || state == sv.StatusCancelled {
			require.Error(t, job.Err)
		} else {
			require.NoError(t, job.Err)
		}
	}
	failure = errors.New("native decode failure")
	_, err = Lookup(context.Background(), p, wf, "id")
	require.ErrorIs(t, err, failure)
	require.NoError(t, p.Shutdown(context.Background()))
}
