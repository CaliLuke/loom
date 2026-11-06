package selvedge

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"

	sv "github.com/CaliLuke/selvedge"

	"github.com/CaliLuke/loom/execution"
)

type (
	// ProducerOption configures native producer storage, migration and notifications.
	// Native worker options are ignored; client observability wraps operations.
	ProducerOption = sv.Option

	// Producer owns a producer-only native handle. Declare contracts before Start.
	// Its zero value is unusable; construct it with NewProducer.
	Producer struct {
		core *workflowCore
	}

	// Worker owns a native worker and its workflow and activity registrations.
	// Its zero value is unusable; construct it with NewWorker.
	Worker struct {
		core       *workflowCore
		activities *Activities
	}

	// Role is a producer or worker with a private native handle and one lifecycle.
	Role interface {
		role() *workflowCore
	}

	// Admission identifies an existing or newly inserted durable job.
	Admission struct {
		// ID is the caller-chosen durable instance identifier.
		ID string
		// Created reports that this call inserted the job, rather than reconciled it.
		Created bool
	}

	// Job is a typed snapshot. Output is meaningful only in the Completed state.
	Job[O any] struct {
		// ID identifies the durable instance.
		ID string
		// State is the native Running, Completed, Failed or Cancelled status.
		State sv.WorkflowStatus
		// Output is the decoded completed result; other states leave its zero value.
		Output O
		// Err is the recorded failure for Failed or Cancelled, not a lookup error.
		Err error
		// StartedAt records native admission time.
		StartedAt time.Time
		// UpdatedAt records the identity snapshot's last native update time.
		UpdatedAt time.Time
	}

	// AdmissionConflictError reports an existing job with a different identity.
	// ExistingInput can match a prior uncertain attempt; conflict alone does not
	// prove that the caller's logical request was never admitted.
	AdmissionConflictError struct {
		// ID is the occupied instance ID.
		ID string
		// Workflow is the requested workflow name.
		Workflow string
		// ExistingWorkflow is the persisted workflow name.
		ExistingWorkflow string
		// InputDiffers reports unequal frozen encoded input bytes.
		InputDiffers bool
		// ChildInstance reports persisted parent identity rather than a root job.
		ChildInstance bool
		// ExistingInput is a copy of stored bytes for prior-attempt reconciliation.
		ExistingInput []byte
	}

	// AdmissionUncertainError means this call may have committed Input under ID.
	// Retain Input across retries; a later rejected call does not prove absence.
	AdmissionUncertainError struct {
		// ID is the caller-chosen instance ID to reuse on retry.
		ID string
		// Input is a copy of this call's frozen bytes, not a fresh encoding.
		Input []byte
		// Err is the native admission or reconciliation-read failure.
		Err error
	}

	workflowState    uint8
	workflowContract struct {
		in, out    reflect.Type
		registered bool
		read       func(context.Context, string) (any, error)
	}
	workflowCore struct {
		mu                sync.Mutex
		state             workflowState
		contracts         map[string]workflowContract
		native            *workflowNative
		cleanupDone       chan struct{}
		cleanupResult     error
		shutdownRequested bool
		cancelLife        context.CancelFunc
		operations        sync.WaitGroup
	}
)

const (
	workflowNew workflowState = iota
	workflowStarting
	workflowRunning
	workflowStopping
	workflowStopped
)

var (
	// ErrNotRunning means the role cannot admit operations in its current state.
	ErrNotRunning = errors.New("selvedge: workflow role is not running")
	// ErrRegistrationSealed means the role has left its initial registration state.
	ErrRegistrationSealed = errors.New("selvedge: workflow registration is sealed")
	// ErrShutdownIncomplete means this caller stopped waiting; cleanup continues.
	ErrShutdownIncomplete = errors.New("selvedge: workflow shutdown wait interrupted")
	// ErrInputEncoding rejects input before insertion, including unstable or invalid UTF-8 JSON.
	ErrInputEncoding = errors.New("selvedge: invalid workflow input encoding")
)

// NewProducer configures an owned producer without opening storage until Start.
func NewProducer(opts ...ProducerOption) (*Producer, error) {
	return &Producer{core: newWorkflowCore(newProducerNative(opts))}, nil
}

// NewWorker configures an owned worker without starting execution until Start.
func NewWorker(opts ...sv.Option) (*Worker, error) {
	native := newWorkerNative(opts)
	activities, err := New(native.app)
	if err != nil {
		return nil, err
	}
	core := newWorkflowCore(native)
	activities.gate = core
	return &Worker{core: core, activities: activities}, nil
}

// Declare records the workflow's name and exact input/output types before Start.
// Repeating the same contract is idempotent; name/type collisions are errors.
func Declare[I, O any](r Role, wf execution.Workflow[I, O]) error {
	c := workflowRoleCore(r)
	if c == nil {
		return ErrNotRunning
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return declareWorkflow[I, O](c, wf.Name)
}

// RegisterWorkflow declares and installs one worker-owned native workflow.
func RegisterWorkflow[I, O any](w *Worker, wf execution.Workflow[I, O], fn func(*sv.WorkflowContext, I) (O, error)) error {
	if w == nil || w.core == nil {
		return ErrNotRunning
	}
	c := w.core
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state != workflowNew {
		return ErrRegistrationSealed
	}
	if fn == nil {
		return errors.New("selvedge: workflow handler is required")
	}
	if prior, ok := c.contracts[wf.Name]; ok && prior.registered {
		return fmt.Errorf("selvedge: workflow %q already registered", wf.Name)
	}
	if err := declareWorkflow[I, O](c, wf.Name); err != nil {
		return err
	}
	sv.RegisterWorkflow(c.native.app, sv.DefineWorkflow(wf.Name, fn))
	contract := c.contracts[wf.Name]
	contract.registered = true
	c.contracts[wf.Name] = contract
	return nil
}

// Start initializes producer storage. Cancelling ctx after success does not close it.
func (p *Producer) Start(ctx context.Context) error {
	return p.core.start(ctx)
}

// Shutdown drains admitted calls and waits for cleanup; ctx bounds only this wait.
func (p *Producer) Shutdown(ctx context.Context) error {
	return p.core.shutdown(ctx)
}

// Ready reports lifecycle readiness, not database connectivity.
func (p *Producer) Ready() error {
	return p.core.ready()
}

// Start initializes storage and starts the registered worker execution loops.
func (w *Worker) Start(ctx context.Context) error {
	return w.core.start(ctx)
}

// Shutdown drains admitted calls and waits for native cleanup without stopping other waiters.
func (w *Worker) Shutdown(ctx context.Context) error {
	return w.core.shutdown(ctx)
}

// Ready reports lifecycle readiness, not database connectivity.
func (w *Worker) Ready() error {
	return w.core.ready()
}

// Activities returns the registry sharing this worker's registration lifecycle.
func (w *Worker) Activities() *Activities {
	return w.activities
}

// Error describes the conflicting identity without including encoded input.
func (e *AdmissionConflictError) Error() string {
	return "workflow identity conflict"
}

// Unwrap makes a conflict match the native duplicate-instance sentinel.
func (e *AdmissionConflictError) Unwrap() error {
	return sv.ErrWorkflowAlreadyStarted
}

// Error describes uncertain admission without including input bytes or error details.
func (e *AdmissionUncertainError) Error() string {
	return "workflow admission uncertain"
}

// Unwrap preserves the native admission or read failure for errors.Is/As.
func (e *AdmissionUncertainError) Unwrap() error {
	return e.Err
}
func (p *Producer) role() *workflowCore {
	return p.core
}
func (w *Worker) role() *workflowCore {
	return w.core
}
func newWorkflowCore(native *workflowNative) *workflowCore {
	return &workflowCore{native: native, contracts: make(map[string]workflowContract), cleanupDone: make(chan struct{})}
}

func workflowRoleCore(r Role) *workflowCore {
	switch role := r.(type) {
	case *Producer:
		if role != nil {
			return role.core
		}
	case *Worker:
		if role != nil {
			return role.core
		}
	}
	return nil
}

func declareWorkflow[I, O any](c *workflowCore, name string) error {
	if c.state != workflowNew {
		return ErrRegistrationSealed
	}
	if len(name) == 0 || len(name) > 255 {
		return errors.New("selvedge: workflow name must contain 1 to 255 bytes")
	}
	in, out := reflect.TypeFor[I](), reflect.TypeFor[O]()
	if prior, ok := c.contracts[name]; ok {
		if prior.in != in || prior.out != out {
			return fmt.Errorf("selvedge: workflow %q has different input or output types", name)
		}
		return nil
	}
	c.contracts[name] = workflowContract{in: in, out: out, read: func(ctx context.Context, id string) (any, error) {
		if c.native.app != nil {
			return sv.GetWorkflowResult[O](ctx, c.native.app, id)
		}
		return sv.GetProducerWorkflowResult[O](ctx, c.native.producer, id)
	}}
	return nil
}
