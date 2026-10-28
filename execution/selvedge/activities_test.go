package selvedge_test

import (
	"context"
	"errors"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CaliLuke/loom/execution"
	adapter "github.com/CaliLuke/loom/execution/selvedge"
	sv "github.com/CaliLuke/selvedge"
	"github.com/CaliLuke/selvedge/retry"
	"github.com/stretchr/testify/require"
)

type metadata struct {
	Key string `json:"key"`
}

func TestNativePolicyAndTypedResults(t *testing.T) {
	app := newApp(t)
	activities, err := adapter.New(app)
	require.NoError(t, err)
	activity := execution.Activity[metadata, string]{Name: "metadata.read"}
	var attempts atomic.Int32
	require.NoError(t, adapter.Register(activities, activity, func(_ context.Context, input metadata) (string, error) {
		if attempts.Add(1) == 1 {
			return "", errors.New("temporary object store failure")
		}
		return input.Key, nil
	}, sv.ActivityOptions{RetryPolicy: retry.Fixed(2, time.Millisecond)}))
	workflow := sv.DefineWorkflow("metadata.refresh", func(ctx *sv.WorkflowContext, input metadata) (string, error) {
		future, err := activity.ExecuteAsync(context.Background(), activities.Scope(ctx), input, execution.ActivityOptions{})
		if err != nil {
			return "", err
		}
		out, err := future.Get(context.Background())
		if err != nil {
			return "", err
		}
		again, err := future.Get(context.Background())
		if err == nil && (again != out || !future.IsReady()) {
			return "", errors.New("completed future changed")
		}
		return out, err
	})
	sv.RegisterWorkflow(app, workflow)
	ctx := startApp(t, app)
	id, err := sv.StartWorkflow(ctx, app, workflow, metadata{Key: "asset.png"})
	require.NoError(t, err)
	result, err := sv.WaitWorkflow[string](ctx, app, id)
	require.NoError(t, err)
	require.NoError(t, result.Error)
	require.Equal(t, "asset.png", result.Output)
	require.EqualValues(t, 2, attempts.Load())
}

func TestNativePermanentErrorPolicy(t *testing.T) {
	app := newApp(t)
	activities, err := adapter.New(app)
	require.NoError(t, err)
	activity := execution.Activity[metadata, string]{Name: "metadata.read"}
	permanent := errors.New("permanent")
	policy := retry.Fixed(3, time.Millisecond)
	policy.NonRetryableErrors = []error{permanent}
	var attempts atomic.Int32
	require.NoError(t, adapter.Register(activities, activity, func(context.Context, metadata) (string, error) {
		attempts.Add(1)
		return "", permanent
	}, sv.ActivityOptions{RetryPolicy: policy}))
	workflow := sv.DefineWorkflow("metadata.refresh", func(ctx *sv.WorkflowContext, input metadata) (string, error) {
		return activity.Execute(context.Background(), activities.Scope(ctx), input, execution.ActivityOptions{})
	})
	sv.RegisterWorkflow(app, workflow)
	ctx := startApp(t, app)
	id, err := sv.StartWorkflow(ctx, app, workflow, metadata{Key: "asset.png"})
	require.NoError(t, err)
	result, err := sv.WaitWorkflow[string](ctx, app, id)
	require.NoError(t, err)
	require.ErrorContains(t, result.Error, permanent.Error())
	require.EqualValues(t, 1, attempts.Load())
}

func newApp(t *testing.T, options ...sv.Option) *sv.App {
	t.Helper()
	defaults := make([]sv.Option, 0, 3+len(options))
	defaults = append(defaults,
		sv.WithDatabase("file:"+filepath.Join(t.TempDir(), "workflow.db")),
		sv.WithWaitPollInterval(time.Millisecond),
		sv.WithActivityPollInterval(time.Millisecond),
	)
	return sv.NewApp(append(defaults, options...)...)
}

func startApp(t *testing.T, app *sv.App) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	require.NoError(t, app.Start(ctx))
	t.Cleanup(func() {
		require.NoError(t, app.Shutdown(context.Background()))
	})
	return ctx
}
