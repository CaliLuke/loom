package selvedge_test

import (
	"context"
	"testing"

	"github.com/CaliLuke/loom/execution"
	adapter "github.com/CaliLuke/loom/execution/selvedge"
	sv "github.com/CaliLuke/selvedge"
	"github.com/stretchr/testify/require"
)

func TestRegistrationRejectsInvalidHandlersAndDuplicateNames(t *testing.T) {
	_, err := adapter.New(nil)
	require.Error(t, err)
	activities, err := adapter.New(newApp(t))
	require.NoError(t, err)
	var nilHandler func(context.Context, metadata) (string, error)
	require.Error(t, adapter.Register(activities, execution.Activity[metadata, string]{Name: "metadata.read"}, nilHandler, sv.ActivityOptions{}))
	activity := execution.Activity[metadata, string]{Name: "metadata.read"}
	handler := func(_ context.Context, input metadata) (string, error) {
		return input.Key, nil
	}
	require.Error(t, adapter.Register(activities, execution.Activity[metadata, string]{}, handler, sv.ActivityOptions{}))
	require.NoError(t, adapter.Register(activities, activity, handler, sv.ActivityOptions{}))
	require.Error(t, adapter.Register(activities, activity, handler, sv.ActivityOptions{}))
}

func TestNativeConflictDoesNotPublishLocalDefinition(t *testing.T) {
	app := newApp(t)
	native := sv.DefineActivity("metadata.read", func(context.Context, metadata) (string, error) {
		return "native", nil
	})
	require.NoError(t, sv.RegisterActivity(app, native))
	activities, err := adapter.New(app)
	require.NoError(t, err)
	activity := execution.Activity[metadata, string]{Name: "metadata.read"}
	err = adapter.Register(activities, activity, func(context.Context, metadata) (string, error) {
		return "conflicting", nil
	}, sv.ActivityOptions{})
	require.ErrorIs(t, err, sv.ErrActivityNameConflict)
	workflow := sv.DefineWorkflow("probe", func(ctx *sv.WorkflowContext, input metadata) (string, error) {
		_, err := activity.Execute(context.Background(), activities.Scope(ctx), input, execution.ActivityOptions{})
		if err == nil {
			return "published", nil
		}
		return "rejected", nil
	})
	sv.RegisterWorkflow(app, workflow)
	ctx := startApp(t, app)
	id, err := sv.StartWorkflow(ctx, app, workflow, metadata{Key: "asset.png"})
	require.NoError(t, err)
	result, err := sv.WaitWorkflow[string](ctx, app, id)
	require.NoError(t, err)
	require.NoError(t, result.Error)
	require.Equal(t, "rejected", result.Output)
	tasks, err := sv.ListActivityTasks(ctx, app, id)
	require.NoError(t, err)
	require.Empty(t, tasks)
}
