package selvedge

import (
	"context"
	"fmt"

	sv "github.com/CaliLuke/selvedge"

	"github.com/CaliLuke/loom/execution"
)

// Lookup checks workflow identity before decoding the native typed result.
// Native missing-instance and decoding errors are lookup errors, not Job.Err.
func Lookup[I, O any](ctx context.Context, r Role, wf execution.Workflow[I, O], id string) (Job[O], error) {
	c := workflowRoleCore(r)
	contract, err := admitWorkflow[I, O](c, wf.Name)
	if err != nil {
		return Job[O]{}, err
	}
	defer c.operations.Done()
	desc, err := c.native.describe(ctx, id)
	if err != nil {
		return Job[O]{}, err
	}
	if desc.WorkflowName != wf.Name {
		return Job[O]{}, fmt.Errorf("selvedge: instance %q belongs to workflow %q, not %q", id, desc.WorkflowName, wf.Name)
	}
	value, err := contract.read(ctx, id)
	if err != nil {
		return Job[O]{}, err
	}
	result, ok := value.(*sv.WorkflowResult[O])
	if !ok || result == nil {
		return Job[O]{}, fmt.Errorf("selvedge: invalid result for workflow %q", wf.Name)
	}
	job := Job[O]{ID: id, State: sv.WorkflowStatus(result.Status), StartedAt: desc.StartedAt, UpdatedAt: desc.UpdatedAt}
	switch job.State {
	case sv.StatusCompleted:
		job.Output = result.Output
	case sv.StatusFailed, sv.StatusCancelled:
		job.Err = result.Error
	}
	return job, nil
}
