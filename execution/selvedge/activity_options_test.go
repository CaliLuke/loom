package selvedge_test

import (
	"context"
	"testing"
	"time"

	"github.com/CaliLuke/loom/execution"
	adapter "github.com/CaliLuke/loom/execution/selvedge"
	sv "github.com/CaliLuke/selvedge"
	"github.com/CaliLuke/selvedge/retry"
	"github.com/stretchr/testify/require"
)

func TestPortableRetryOverridePreservesNativeMaximumInterval(t *testing.T) {
	app := newApp(t)
	activities, err := adapter.New(app)
	require.NoError(t, err)
	activity := execution.Activity[metadata, time.Duration]{Name: "metadata.read"}
	policy := retry.Exponential(3, time.Millisecond, 7*time.Second, 2)
	require.NoError(t, adapter.Register(activities, activity, func(ctx context.Context, _ metadata) (time.Duration, error) {
		return sv.GetActivityInfo(ctx).Options.RetryPolicy.MaxInterval, nil
	}, sv.ActivityOptions{RetryPolicy: policy}))
	workflow := sv.DefineWorkflow("metadata.refresh", func(ctx *sv.WorkflowContext, input metadata) (time.Duration, error) {
		return activity.Execute(ctx.Context(), activities.Scope(ctx), input, execution.ActivityOptions{
			RetryPolicy: execution.RetryPolicy{InitialInterval: time.Second},
		})
	})
	sv.RegisterWorkflow(app, workflow)
	ctx := startApp(t, app)
	id, err := sv.StartWorkflow(ctx, app, workflow, metadata{Key: "asset.png"})
	require.NoError(t, err)
	result, err := sv.WaitWorkflow[time.Duration](ctx, app, id)
	require.NoError(t, err)
	require.NoError(t, result.Error)
	require.Equal(t, 7*time.Second, result.Output)
}

func TestActivityTimeoutPolicy(t *testing.T) {
	for _, tc := range []struct {
		name       string
		definition time.Duration
		invocation time.Duration
	}{
		{name: "native default", definition: 20 * time.Millisecond},
		{name: "portable override", definition: time.Hour, invocation: 20 * time.Millisecond},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app := newApp(t, sv.WithTimerCheckInterval(time.Millisecond))
			activities, err := adapter.New(app)
			require.NoError(t, err)
			activity := execution.Activity[metadata, string]{Name: "metadata.read"}
			require.NoError(t, adapter.Register(activities, activity, func(ctx context.Context, _ metadata) (string, error) {
				<-ctx.Done()
				return "", ctx.Err()
			}, sv.ActivityOptions{
				StartToCloseTimeout: tc.definition,
				RetryPolicy:         retry.Fixed(1, time.Millisecond),
			}))
			workflow := sv.DefineWorkflow("metadata.refresh", func(ctx *sv.WorkflowContext, input metadata) (string, error) {
				return activity.Execute(ctx.Context(), activities.Scope(ctx), input, execution.ActivityOptions{
					StartToCloseTimeout: tc.invocation,
				})
			})
			sv.RegisterWorkflow(app, workflow)
			ctx := startApp(t, app)
			id, err := sv.StartWorkflow(ctx, app, workflow, metadata{})
			require.NoError(t, err)
			result, err := sv.WaitWorkflow[string](ctx, app, id)
			require.NoError(t, err)
			require.ErrorIs(t, result.Error, context.DeadlineExceeded)
			var timeout *sv.ActivityTimeoutError
			require.ErrorAs(t, result.Error, &timeout)
			require.Equal(t, sv.TimeoutStartToClose, timeout.Kind)
			tasks, err := sv.ListActivityTasks(ctx, app, id)
			require.NoError(t, err)
			require.Len(t, tasks, 1)
			require.Equal(t, 20*time.Millisecond, tasks[0].Options.StartToCloseTimeout)
		})
	}
}
