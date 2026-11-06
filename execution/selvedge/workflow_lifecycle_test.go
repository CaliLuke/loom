package selvedge

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	sv "github.com/CaliLuke/selvedge"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/execution"
)

type workflowLifecycleProbe struct {
	role           *Producer
	life           context.Context
	cleanupEntered chan struct{}
	cleanupRelease chan struct{}
	cleanupCalls   atomic.Int32
}

func newLifecycleProbe() *workflowLifecycleProbe {
	p := &workflowLifecycleProbe{cleanupEntered: make(chan struct{}), cleanupRelease: make(chan struct{})}
	n := &workflowNative{}
	n.start = func(ctx context.Context) error {
		p.life = ctx
		return nil
	}
	n.shutdown = func(context.Context) error {
		p.cleanupCalls.Add(1)
		close(p.cleanupEntered)
		<-p.cleanupRelease
		return nil
	}
	p.role = &Producer{core: newWorkflowCore(n)}
	return p
}

func awaitWorkflow(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatal("workflow synchronization timed out")
	}
}

func TestWorkflowLifecycle(t *testing.T) {
	t.Run("lifetime and independent shutdown waiters", func(t *testing.T) {
		p := newLifecycleProbe()
		require.ErrorIs(t, p.role.Ready(), ErrNotRunning)
		ctx, cancel := context.WithCancel(context.Background())
		require.NoError(t, p.role.Start(ctx))
		cancel()
		require.NoError(t, p.life.Err())
		require.NoError(t, p.role.Ready())
		require.Error(t, p.role.Start(context.Background()))
		require.ErrorIs(t, p.role.Shutdown(ctx), ErrShutdownIncomplete)
		awaitWorkflow(t, p.cleanupEntered)
		require.ErrorIs(t, p.role.Ready(), ErrNotRunning)
		waiting := make(chan error, 1)
		go func() {
			waiting <- p.role.Shutdown(context.Background())
		}()
		close(p.cleanupRelease)
		require.NoError(t, <-waiting)
		require.NoError(t, p.role.Shutdown(ctx))
		require.EqualValues(t, 1, p.cleanupCalls.Load())
		require.Error(t, p.role.Start(context.Background()))
	})
	t.Run("shutdown before start", func(t *testing.T) {
		p := newLifecycleProbe()
		require.NoError(t, p.role.Shutdown(context.Background()))
		require.Error(t, p.role.Start(context.Background()))
		require.Zero(t, p.cleanupCalls.Load())
	})
	t.Run("native interruptions are not final", func(t *testing.T) {
		p := newLifecycleProbe()
		var calls atomic.Int32
		p.role.core.native.shutdown = func(ctx context.Context) error {
			require.NoError(t, ctx.Err())
			if calls.Add(1) == 1 {
				return errors.Join(sv.ErrShutdownInterrupted, context.DeadlineExceeded)
			}
			return nil
		}
		require.NoError(t, p.role.Start(context.Background()))
		require.NoError(t, p.role.Shutdown(context.Background()))
		require.EqualValues(t, 2, calls.Load())
	})
	for _, failure := range []error{context.Canceled, context.DeadlineExceeded} {
		t.Run("final "+failure.Error(), func(t *testing.T) {
			p := newLifecycleProbe()
			var calls atomic.Int32
			final := errors.Join(errors.New("storage close failed"), failure)
			p.role.core.native.shutdown = func(context.Context) error {
				calls.Add(1)
				return final
			}
			require.NoError(t, p.role.Start(context.Background()))
			require.ErrorIs(t, p.role.Shutdown(context.Background()), final)
			require.ErrorIs(t, p.role.Shutdown(context.Background()), final)
			require.EqualValues(t, 1, calls.Load())
		})
	}
}

func TestWorkflowRegistration(t *testing.T) {
	w, err := NewWorker()
	require.NoError(t, err)
	wf := execution.Workflow[string, string]{Name: "test.workflow"}
	require.NoError(t, Declare(w, wf))
	require.NoError(t, Declare(w, wf))
	require.Error(t, Declare(w, execution.Workflow[string, int]{Name: wf.Name}))
	require.Error(t, Declare(w, execution.Workflow[string, int]{}))
	require.Error(t, Declare(w, execution.Workflow[string, string]{Name: strings.Repeat("x", 256)}))
	require.NoError(t, Declare(w, execution.Workflow[string, string]{Name: strings.Repeat("x", 255)}))
	require.Error(t, RegisterWorkflow(w, wf, nil))
	require.NoError(t, RegisterWorkflow(w, wf, func(_ *sv.WorkflowContext, s string) (string, error) {
		return s, nil
	}))
	require.Error(t, RegisterWorkflow(w, wf, func(_ *sv.WorkflowContext, s string) (string, error) {
		return s, nil
	}))
	retained := w.Activities()
	w.core.native.start = func(context.Context) error {
		return nil
	}
	w.core.native.shutdown = func(context.Context) error {
		return nil
	}
	require.NoError(t, w.Start(context.Background()))
	require.ErrorIs(t, Declare(w, wf), ErrRegistrationSealed)
	require.ErrorIs(t, RegisterWorkflow(w, wf, func(_ *sv.WorkflowContext, s string) (string, error) {
		return s, nil
	}), ErrRegistrationSealed)
	require.ErrorIs(t, Register(retained, execution.Activity[string, string]{Name: "late"}, func(_ context.Context, s string) (string, error) {
		return s, nil
	}, sv.ActivityOptions{}), ErrRegistrationSealed)
	require.NoError(t, w.Shutdown(context.Background()))
	require.ErrorIs(t, Declare(w, wf), ErrRegistrationSealed)
}

func TestWorkflowStartShutdownRace(t *testing.T) {
	t.Run("shutdown during initialization", func(t *testing.T) {
		p := newLifecycleProbe()
		entered, release := make(chan struct{}), make(chan struct{})
		p.role.core.native.start = func(ctx context.Context) error {
			p.life = ctx
			close(entered)
			<-release
			return nil
		}
		started := make(chan error, 1)
		go func() {
			started <- p.role.Start(context.Background())
		}()
		awaitWorkflow(t, entered)
		require.ErrorIs(t, p.role.Ready(), ErrNotRunning)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		require.ErrorIs(t, p.role.Shutdown(ctx), ErrShutdownIncomplete)
		close(release)
		require.ErrorIs(t, <-started, ErrNotRunning)
		awaitWorkflow(t, p.cleanupEntered)
		close(p.cleanupRelease)
		require.NoError(t, p.role.Shutdown(context.Background()))
	})
	t.Run("initialization cancellation", func(t *testing.T) {
		p := newLifecycleProbe()
		entered := make(chan struct{})
		p.role.core.native.start = func(ctx context.Context) error {
			close(entered)
			<-ctx.Done()
			return ctx.Err()
		}
		ctx, cancel := context.WithCancel(context.Background())
		started := make(chan error, 1)
		go func() {
			started <- p.role.Start(ctx)
		}()
		awaitWorkflow(t, entered)
		cancel()
		require.ErrorIs(t, <-started, context.Canceled)
		require.ErrorIs(t, p.role.Ready(), ErrNotRunning)
		require.NoError(t, p.role.Shutdown(context.Background()))
		require.Zero(t, p.cleanupCalls.Load())
	})
}

func TestWorkflowOperationDrain(t *testing.T) {
	p, _ := newAdmissionRole()
	wf := execution.Workflow[string, string]{Name: "drain"}
	require.NoError(t, Declare(p, wf))
	entered, release, closed := make(chan struct{}), make(chan struct{}), make(chan struct{})
	p.core.native.submit = func(context.Context, string, string, []byte) error {
		close(entered)
		<-release
		return nil
	}
	p.core.native.shutdown = func(context.Context) error {
		close(closed)
		return nil
	}
	require.NoError(t, p.Start(context.Background()))
	result := make(chan error, 1)
	go func() {
		_, err := Submit(context.Background(), p, wf, "id", "input")
		result <- err
	}()
	awaitWorkflow(t, entered)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.ErrorIs(t, p.Shutdown(ctx), ErrShutdownIncomplete)
	select {
	case <-closed:
		t.Error("native cleanup ran before admitted operation drained")
	default:
	}
	_, err := Submit(context.Background(), p, wf, "late", "input")
	require.ErrorIs(t, err, ErrNotRunning)
	close(release)
	require.NoError(t, <-result)
	require.NoError(t, p.Shutdown(context.Background()))
	awaitWorkflow(t, closed)
}
