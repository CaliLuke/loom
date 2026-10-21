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
	}, sv.ActivityOptions{StartToCloseTimeout: time.Second, RetryPolicy: policy}))
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
