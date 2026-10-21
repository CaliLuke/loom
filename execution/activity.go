// Package execution defines typed activity contracts for workflow adapters.
// It does not own persistence, workflow scheduling, or application policy.
package execution

import "context"

type (
	// ActivityRegistrar installs activities before workers begin polling.
	ActivityRegistrar interface {
		// RegisterActivity validates and installs a concrete handler.
		RegisterActivity(context.Context, ActivityDefinition) error
	}

	// ActivityExecutor schedules named activities in a workflow scope.
	// Results must be decoded into the supplied pointer using the backend's codec.
	ActivityExecutor interface {
		// ExecuteActivity schedules and waits for a result, decoded into a pointer.
		ExecuteActivity(context.Context, ActivityCall, any) error
		// ExecuteActivityAsync schedules a command in this executor's workflow scope.
		ExecuteActivityAsync(context.Context, ActivityCall) (ActivityFuture, error)
	}

	// ActivityDefinition describes an activity independently of agent payload types.
	ActivityDefinition struct {
		// Name identifies the activity within its backend registry.
		Name string
		// Options supplies portable registration defaults.
		Options ActivityOptions
		// Handler must be func(context.Context, I) (O, error) or func(context.Context, I) error.
		// Use Activity[I, O].Register for compile-time checking of input and output.
		Handler any
	}

	// ActivityCall carries a typed input unchanged to the backend's serializer.
	ActivityCall struct {
		// Name identifies a registered activity.
		Name string
		// Input is passed to the backend codec without a wrapper envelope.
		Input any
		// Options overrides registration defaults according to backend rules.
		Options ActivityOptions
	}

	// ActivityFuture is the backend result boundary. Application code should use
	// Activity[I, O], which exposes Future[O] instead of result pointers.
	ActivityFuture interface {
		// Get waits using the captured workflow scope and decodes into a pointer.
		Get(context.Context, any) error
		// IsReady reports completion without blocking.
		IsReady() bool
	}

	// Activity binds a stable activity name to its input and output types.
	// Reusing the same value for registration and invocation keeps both ends typed.
	Activity[I, O any] struct {
		// Name must be stable across worker restarts and workflow replay.
		Name string
	}

	typedActivityFuture[O any] struct {
		future ActivityFuture
	}
)

// Register installs the typed handler and its default scheduling options.
func (a Activity[I, O]) Register(ctx context.Context, registrar ActivityRegistrar, opts ActivityOptions, fn func(context.Context, I) (O, error)) error {
	return registrar.RegisterActivity(ctx, ActivityDefinition{Name: a.Name, Options: opts, Handler: fn})
}

// Execute schedules the activity and waits for its typed result.
func (a Activity[I, O]) Execute(ctx context.Context, executor ActivityExecutor, input I, opts ActivityOptions) (O, error) {
	var out O
	err := executor.ExecuteActivity(ctx, ActivityCall{Name: a.Name, Input: input, Options: opts}, &out)
	return out, err
}

// ExecuteAsync schedules the activity without waiting for completion.
func (a Activity[I, O]) ExecuteAsync(ctx context.Context, executor ActivityExecutor, input I, opts ActivityOptions) (Future[O], error) {
	future, err := executor.ExecuteActivityAsync(ctx, ActivityCall{Name: a.Name, Input: input, Options: opts})
	if err != nil {
		return nil, err
	}
	return typedActivityFuture[O]{future: future}, nil
}

func (f typedActivityFuture[O]) Get(ctx context.Context) (O, error) {
	var out O
	err := f.future.Get(ctx, &out)
	return out, err
}

func (f typedActivityFuture[O]) IsReady() bool {
	return f.future.IsReady()
}
