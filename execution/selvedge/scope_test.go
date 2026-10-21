package selvedge_test

import (
	"context"
	"errors"
	"testing"

	"github.com/CaliLuke/loom/execution"
	adapter "github.com/CaliLuke/loom/execution/selvedge"
	sv "github.com/CaliLuke/selvedge"
	"github.com/stretchr/testify/require"
)

func TestActivityScopesStayIndependent(t *testing.T) {
	app := newApp(t)
	activities, err := adapter.New(app)
	require.NoError(t, err)
	activity := execution.Activity[metadata, string]{Name: "metadata.read"}
	require.NoError(t, adapter.Register(activities, activity, func(_ context.Context, input metadata) (string, error) {
		return input.Key, nil
	}, sv.ActivityOptions{}))
	workflow := sv.DefineWorkflow("scopes", func(ctx *sv.WorkflowContext, input metadata) (string, error) {
		parent, err := activity.ExecuteAsync(ctx.Context(), activities.Scope(ctx), input, execution.ActivityOptions{})
		if err != nil {
			return "", err
		}
		child, cancel := sv.WithCancel(ctx)
		cancel()
		if _, err := activity.Execute(ctx.Context(), activities.Scope(child), input, execution.ActivityOptions{}); !errors.Is(err, context.Canceled) {
			return "", errors.New("canceled scope admitted an activity")
		}
		detached, err := activity.Execute(ctx.Context(), activities.Scope(sv.Detached(child)), input, execution.ActivityOptions{})
		if err != nil {
			return "", err
		}
		out, err := parent.Get(ctx.Context())
		if err == nil && out != detached {
			return "", errors.New("activity scopes changed results")
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
}
