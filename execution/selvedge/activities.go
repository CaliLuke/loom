// Package selvedge binds typed activities and durable workflows to Selvedge.
// Producer and Worker own their private handles and lifecycle; New(app) retains
// the caller-owned native path. Selvedge owns persistence, dispatch and codecs.
package selvedge

import (
	"context"
	"errors"
	"fmt"
	"sync"

	sv "github.com/CaliLuke/selvedge"

	"github.com/CaliLuke/loom/execution"
	"github.com/CaliLuke/loom/execution/adapter"
)

type (
	// Activities registers named handlers on one Selvedge app. A Worker-owned
	// registry shares its registration gate; New(app) retains caller ownership.
	Activities struct {
		app         *sv.App
		gate        *workflowCore
		mu          sync.RWMutex
		definitions map[string]invocation
	}

	invocation func(*sv.WorkflowContext, any, execution.ActivityOptions) (execution.ActivityFuture, error)

	scope struct {
		activities *Activities
		ctx        *sv.WorkflowContext
	}

	result[O any] struct {
		ctx    *sv.WorkflowContext
		future *sv.Future[O]
	}
)

// New binds activity registration to app without starting or stopping it.
func New(app *sv.App) (*Activities, error) {
	if app == nil {
		return nil, errors.New("selvedge: activity registration requires an app")
	}
	return &Activities{app: app, definitions: make(map[string]invocation)}, nil
}

// Register installs a typed activity with native Selvedge options and codecs.
// Its generic signature preserves the input and output types for Selvedge;
// Activities therefore does not implement execution.ActivityRegistrar.
func Register[I, O any](activities *Activities, activity execution.Activity[I, O], handler func(context.Context, I) (O, error), options sv.ActivityOptions) error {
	if activities == nil {
		return errors.New("selvedge: activity registration requires a registry")
	}
	if activities.gate != nil {
		activities.gate.mu.Lock()
		defer activities.gate.mu.Unlock()
		if activities.gate.state != workflowNew {
			return ErrRegistrationSealed
		}
	}
	if activity.Name == "" {
		return errors.New("selvedge: activity name is required")
	}
	if handler == nil {
		return errors.New("selvedge: activity handler is required")
	}
	activities.mu.Lock()
	defer activities.mu.Unlock()
	if _, exists := activities.definitions[activity.Name]; exists {
		return fmt.Errorf("selvedge: activity %q already registered", activity.Name)
	}
	definition := sv.DefineActivity(activity.Name, handler, sv.WithActivityDefaults[I, O](options))
	if err := sv.RegisterActivity(activities.app, definition); err != nil {
		return err
	}
	activities.definitions[activity.Name] = func(ctx *sv.WorkflowContext, input any, opts execution.ActivityOptions) (execution.ActivityFuture, error) {
		typed, ok := input.(I)
		if !ok {
			return nil, fmt.Errorf("selvedge: activity %q input must have type %T", activity.Name, *new(I))
		}
		future, err := definition.ExecuteAsync(ctx, typed, sv.WithActivityOptions(invocationActivityOptions(opts)))
		if err != nil {
			return nil, err
		}
		return result[O]{ctx: ctx, future: future}, nil
	}
	return nil
}

// Scope binds execution to ctx, including its cancellation and detached-scope
// behavior. The context.Context arguments of execution calls do not replace this
// native workflow context. Futures retain the scope that scheduled them.
func (a *Activities) Scope(ctx *sv.WorkflowContext) execution.ActivityExecutor {
	return scope{activities: a, ctx: ctx}
}

func (s scope) ExecuteActivity(ctx context.Context, call execution.ActivityCall, out any) error {
	future, err := s.ExecuteActivityAsync(ctx, call)
	if err != nil {
		return err
	}
	return future.Get(ctx, out)
}

func (s scope) ExecuteActivityAsync(_ context.Context, call execution.ActivityCall) (execution.ActivityFuture, error) {
	if err := adapter.ValidateCall(call.Name, call.Input); err != nil {
		return nil, err
	}
	s.activities.mu.RLock()
	invoke := s.activities.definitions[call.Name]
	s.activities.mu.RUnlock()
	if invoke == nil {
		return nil, fmt.Errorf("selvedge: activity %q is not registered", call.Name)
	}
	return invoke(s.ctx, call.Input, call.Options)
}

func (r result[O]) Get(_ context.Context, out any) error {
	value, err := r.future.Get(r.ctx)
	if err != nil {
		return err
	}
	return adapter.Assign(out, value)
}

func (r result[O]) IsReady() bool {
	return r.future.IsReady()
}
