package selvedge

import (
	"context"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	sv "github.com/CaliLuke/selvedge"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/execution"
)

type workflowMemory struct {
	mu      sync.Mutex
	stored  map[string]*sv.WorkflowDescription
	inserts atomic.Int32
}

type workflowStatefulInput struct{ calls *atomic.Int32 }

func (v workflowStatefulInput) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`{"attempt":%d}`, v.calls.Add(1))), nil
}

func newAdmissionRole() (*Producer, *workflowMemory) {
	store := &workflowMemory{stored: make(map[string]*sv.WorkflowDescription)}
	n := &workflowNative{
		start: func(context.Context) error {
			return nil
		},
		shutdown: func(context.Context) error {
			return nil
		},
	}
	n.submit = func(_ context.Context, name, id string, input []byte) error {
		store.mu.Lock()
		defer store.mu.Unlock()
		if store.stored[id] != nil {
			return sv.ErrWorkflowAlreadyStarted
		}
		store.inserts.Add(1)
		store.stored[id] = &sv.WorkflowDescription{InstanceID: id, WorkflowName: name, InputData: append([]byte(nil), input...), Status: sv.StatusRunning}
		return nil
	}
	n.describe = func(_ context.Context, id string) (*sv.WorkflowDescription, error) {
		store.mu.Lock()
		defer store.mu.Unlock()
		d := store.stored[id]
		if d == nil {
			return nil, &sv.WorkflowNotFoundError{InstanceID: id}
		}
		copy := *d
		copy.InputData = append([]byte(nil), d.InputData...)
		return &copy, nil
	}
	return &Producer{core: newWorkflowCore(n)}, store
}

func TestWorkflowAdmission(t *testing.T) {
	t.Run("same and conflicting identities", func(t *testing.T) {
		p, store := newAdmissionRole()
		wf := execution.Workflow[string, string]{Name: "echo"}
		require.NoError(t, Declare(p, wf))
		require.NoError(t, p.Start(context.Background()))
		first, err := Submit(context.Background(), p, wf, "id", "value")
		require.NoError(t, err)
		require.Equal(t, Admission{ID: "id", Created: true}, first)
		again, err := Submit(context.Background(), p, wf, "id", "value")
		require.NoError(t, err)
		require.Equal(t, Admission{ID: "id"}, again)
		_, err = Submit(context.Background(), p, wf, "id", "other")
		var conflict *AdmissionConflictError
		require.ErrorAs(t, err, &conflict)
		require.ErrorIs(t, err, sv.ErrWorkflowAlreadyStarted)
		require.True(t, conflict.InputDiffers)
		require.Equal(t, []byte(`"value"`), conflict.ExistingInput)
		store.stored["id"].WorkflowName = "other"
		_, err = Submit(context.Background(), p, wf, "id", "value")
		require.ErrorAs(t, err, &conflict)
		require.Equal(t, "other", conflict.ExistingWorkflow)
		require.False(t, conflict.InputDiffers)
		store.stored["id"].WorkflowName = wf.Name
		store.stored["id"].ParentInstanceID = "parent"
		_, err = Submit(context.Background(), p, wf, "id", "value")
		require.ErrorAs(t, err, &conflict)
		require.True(t, conflict.ChildInstance)
		require.EqualValues(t, 1, store.inserts.Load())
		require.NoError(t, p.Shutdown(context.Background()))
	})
	t.Run("one encoding and uncertain earlier identity", func(t *testing.T) {
		p, store := newAdmissionRole()
		wf := execution.Workflow[workflowStatefulInput, string]{Name: "stateful"}
		require.NoError(t, Declare(p, wf))
		insert := p.core.native.submit
		lost := errors.New("lost acknowledgement")
		var sends atomic.Int32
		p.core.native.submit = func(ctx context.Context, name, id string, input []byte) error {
			err := insert(ctx, name, id, input)
			if sends.Add(1) == 1 {
				return lost
			}
			return err
		}
		require.NoError(t, p.Start(context.Background()))
		var calls atomic.Int32
		_, err := Submit(context.Background(), p, wf, "id", workflowStatefulInput{&calls})
		var uncertain *AdmissionUncertainError
		require.ErrorAs(t, err, &uncertain)
		require.ErrorIs(t, err, lost)
		require.EqualValues(t, 1, calls.Load())
		require.Equal(t, []byte(`{"attempt":1}`), uncertain.Input)
		_, err = Submit(context.Background(), p, wf, "id", workflowStatefulInput{&calls})
		var conflict *AdmissionConflictError
		require.ErrorAs(t, err, &conflict)
		require.EqualValues(t, 2, calls.Load())
		require.Equal(t, uncertain.Input, conflict.ExistingInput)
		p.core.native.submit = func(context.Context, string, string, []byte) error {
			return sv.ErrWorkflowStartRejected
		}
		_, err = Submit(context.Background(), p, wf, "id", workflowStatefulInput{&calls})
		require.ErrorIs(t, err, sv.ErrWorkflowStartRejected)
		require.Equal(t, uncertain.Input, []byte(store.stored["id"].InputData))
		require.NoError(t, p.Shutdown(context.Background()))
	})
	t.Run("reject before IO", func(t *testing.T) {
		p, store := newAdmissionRole()
		wf := execution.Workflow[jsontext.Value, string]{Name: "raw"}
		require.NoError(t, Declare(p, wf))
		_, err := Submit(context.Background(), p, wf, "id", jsontext.Value(`1`))
		require.ErrorIs(t, err, ErrNotRunning)
		require.NoError(t, p.Start(context.Background()))
		for _, input := range []jsontext.Value{jsontext.Value(`{`), {'"', 0xff, '"'}} {
			_, err := Submit(context.Background(), p, wf, "id", input)
			require.ErrorIs(t, err, ErrInputEncoding)
		}
		_, err = Submit(context.Background(), p, wf, "", jsontext.Value(`1`))
		require.Error(t, err)
		_, err = Submit(context.Background(), p, execution.Workflow[jsontext.Value, string]{Name: "missing"}, "id", jsontext.Value(`1`))
		require.Error(t, err)
		_, err = Submit(context.Background(), p, execution.Workflow[string, string]{Name: wf.Name}, "id", "wrong type")
		require.Error(t, err)
		require.Zero(t, store.inserts.Load())
		require.NoError(t, p.Shutdown(context.Background()))
	})
	t.Run("duplicate read failure remains uncertain", func(t *testing.T) {
		p, _ := newAdmissionRole()
		wf := execution.Workflow[string, string]{Name: "echo"}
		require.NoError(t, Declare(p, wf))
		require.NoError(t, p.Start(context.Background()))
		p.core.native.submit = func(context.Context, string, string, []byte) error {
			return sv.ErrWorkflowAlreadyStarted
		}
		p.core.native.describe = func(context.Context, string) (*sv.WorkflowDescription, error) {
			return nil, context.DeadlineExceeded
		}
		_, err := Submit(context.Background(), p, wf, "id", "input")
		var uncertain *AdmissionUncertainError
		require.ErrorAs(t, err, &uncertain)
		require.ErrorIs(t, err, context.DeadlineExceeded)
		require.Equal(t, []byte(`"input"`), uncertain.Input)
		require.NoError(t, p.Shutdown(context.Background()))
	})
}

func TestWorkflowAdmissionRace(t *testing.T) {
	p, store := newAdmissionRole()
	wf := execution.Workflow[string, string]{Name: "race"}
	require.NoError(t, Declare(p, wf))
	require.NoError(t, p.Start(context.Background()))
	var group sync.WaitGroup
	var created atomic.Int32
	errs := make(chan error, 16)
	for range 16 {
		group.Go(func() {
			a, err := Submit(context.Background(), p, wf, "id", "same")
			if a.Created {
				created.Add(1)
			}
			errs <- err
		})
	}
	group.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	require.EqualValues(t, 1, created.Load())
	require.EqualValues(t, 1, store.inserts.Load())
	require.NoError(t, p.Shutdown(context.Background()))
}

func TestWorkflowAdmissionErrorPrivacy(t *testing.T) {
	secret := "secret-payload"
	uncertain := &AdmissionUncertainError{ID: "id", Input: []byte(secret), Err: errors.New(secret)}
	conflict := &AdmissionConflictError{ID: "id", Workflow: "wf", ExistingWorkflow: "wf", ExistingInput: []byte(secret)}
	require.NotContains(t, uncertain.Error(), secret)
	require.NotContains(t, conflict.Error(), secret)
}
