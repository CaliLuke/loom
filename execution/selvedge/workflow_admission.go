package selvedge

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"reflect"

	sv "github.com/CaliLuke/selvedge"

	"github.com/CaliLuke/loom/execution"
)

// Submit encodes input once and inserts or reconciles the caller-chosen ID.
// Every outcome describes this call; earlier uncertain attempts may still exist.
func Submit[I, O any](ctx context.Context, r Role, wf execution.Workflow[I, O], id string, in I) (Admission, error) {
	c := workflowRoleCore(r)
	if _, err := admitWorkflow[I, O](c, wf.Name); err != nil {
		return Admission{}, err
	}
	defer c.operations.Done()
	if id == "" {
		return Admission{}, errors.New("selvedge: workflow instance ID is required")
	}
	frozen, err := encodeWorkflowInput(in)
	if err != nil {
		return Admission{}, err
	}
	err = c.native.submit(ctx, wf.Name, id, frozen)
	switch {
	case err == nil:
		return Admission{ID: id, Created: true}, nil
	case errors.Is(err, sv.ErrWorkflowStartRejected):
		return Admission{}, err
	case errors.Is(err, sv.ErrWorkflowAlreadyStarted):
		existing, readErr := c.native.describe(ctx, id)
		if readErr != nil {
			return Admission{}, &AdmissionUncertainError{ID: id, Input: frozen, Err: readErr}
		}
		child := existing.ParentInstanceID != "" || existing.ParentKey != ""
		differs := !bytes.Equal(existing.InputData, frozen)
		if existing.WorkflowName != wf.Name || child || differs {
			return Admission{}, &AdmissionConflictError{ID: id, Workflow: wf.Name, ExistingWorkflow: existing.WorkflowName, InputDiffers: differs, ChildInstance: child, ExistingInput: bytes.Clone(existing.InputData)}
		}
		return Admission{ID: id}, nil
	default:
		return Admission{}, &AdmissionUncertainError{ID: id, Input: frozen, Err: err}
	}
}

func admitWorkflow[I, O any](c *workflowCore, name string) (workflowContract, error) {
	if c == nil {
		return workflowContract{}, ErrNotRunning
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state != workflowRunning {
		return workflowContract{}, ErrNotRunning
	}
	contract, ok := c.contracts[name]
	if !ok || contract.in != reflect.TypeFor[I]() || contract.out != reflect.TypeFor[O]() {
		return workflowContract{}, fmt.Errorf("selvedge: workflow %q is not declared with these types", name)
	}
	c.operations.Add(1)
	return contract, nil
}
